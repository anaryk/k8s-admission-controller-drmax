package mutating

import (
	"context"

	azurewrapper "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/azure"
	certmanagerwrapper "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/cert-manager-wrapper"
	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/k8s"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type certificateCaheMutator struct {
	logger       kwhlog.Logger
	keyVaultName string
}

func (m *certificateCaheMutator) Mutate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	cert, ok := obj.(*certmanager.Certificate)
	azureKv, _ := azurewrapper.NewKeyVaultClient(m.keyVaultName)
	if !ok {
		return &kwhmutating.MutatorResult{}, nil
	}
	k8sRestClient, err := k8s.PrepareInClusterK8SClient()
	if err != nil {
		m.logger.Errorf("Error creating k8s rest client: %v", err)
		return &kwhmutating.MutatorResult{}, nil
	}
	k8sClient, err := kubernetes.NewForConfig(k8sRestClient)
	if err != nil {
		m.logger.Errorf("Error creating k8s client: %v", err)
		return &kwhmutating.MutatorResult{}, nil
	}
	certManagerClient, err := certmanagerwrapper.NewCertManagerClient()
	if err != nil {
		m.logger.Errorf("Error creating cert-manager client: %v", err)
		return &kwhmutating.MutatorResult{}, nil
	}

	// Check if the certificate has ownerReferences pointing to an Ingress
	var ingress *v1.Ingress
	for _, ownerRef := range cert.GetOwnerReferences() {
		if ownerRef.Kind == "Ingress" {
			ingress, err = k8sClient.NetworkingV1().Ingresses(cert.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
			if err != nil {
				m.logger.Errorf("Error getting Ingress object: %v", err)
				return &kwhmutating.MutatorResult{}, err
			}
			break
		}
	}
	if ingress == nil {
		m.logger.Infof("Certificate %s does not have an Ingress owner reference, skipping mutation", cert.Name)
		return &kwhmutating.MutatorResult{}, nil
	}

	if ingress.Annotations["admissions.drmax.gl/cache-certs"] != "true" {
		m.logger.Infof("Ingress %s does not have the required annotation, skipping mutation", ingress.Name)
		return &kwhmutating.MutatorResult{}, nil
	}

	exist, err := azureKv.SecretExists(context.TODO(), cert.Name+"--"+cert.Namespace)
	if err != nil {
		m.logger.Errorf("Error checking if certificate is ready: %v", err)
	}
	if exist {
		m.logger.Infof("Certificate %s is found in cache and will be used generated as Secret and Certificate object", cert.Name)
		cert.Status.Conditions = append(cert.Status.Conditions, certmanager.CertificateCondition{
			Type:    certmanager.CertificateConditionReady,
			Status:  cmmeta.ConditionTrue,
			Reason:  "Cached",
			Message: "Certificate is cached",
		})
		expiry, err := azureKv.GetCertificateExpiry(context.TODO(), cert.Name+"--"+cert.Namespace)
		if err != nil {
			m.logger.Errorf("Error getting certificate expiry: %v", err)
		}
		cert.Status.RenewalTime = &metav1.Time{Time: expiry.AddDate(0, 0, -14)}
		err = azureKv.SaveSecretToK8s(context.TODO(), cert.Name+"--"+cert.Namespace, cert.Spec.SecretName, cert.Namespace)
		if err != nil {
			m.logger.Errorf("Error saving secret to k8s: %v", err)
		}
		if cert.Annotations == nil {
			cert.Annotations = make(map[string]string)
		}
		cert.Annotations["admissions.drmax.gl/cert-cached"] = "true"
		cert.Annotations["admissions.drmax.gl/cert-cache-name"] = cert.Name + "--" + cert.Namespace
		cert.Annotations["admissions.drmax.gl/cert-cache-namespace"] = cert.Namespace
		cert.Annotations["admissions.drmax.gl/time-of-sync"] = metav1.Now().String()
		m.logger.Infof(" -- MUTATED -- Certificate %s is loaded from KeyVault!", cert.Name)

		// Create a fake CertificateRequest and mark it as Ready
		err = certManagerClient.CreateFakeCertificateRequest(cert)
		if err != nil {
			m.logger.Errorf("Error creating fake CertificateRequest: %v", err)
			return &kwhmutating.MutatorResult{}, err
		}

		return &kwhmutating.MutatorResult{MutatedObject: cert}, nil
	}
	return &kwhmutating.MutatorResult{}, nil
}
