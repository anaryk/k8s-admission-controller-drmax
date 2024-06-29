package mutating

import (
	"context"

	azurewrapper "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/azure"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	exist, err := azureKv.SecretExists(context.TODO(), cert.Name+"--"+cert.Namespace)
	if err != nil {
		m.logger.Errorf("Error checking if certificate is ready: %v", err)
	}
	if exist {
		m.logger.Infof("Certificate %s is marked for caching. Caching certificate!", cert.Name)
		cert.Status.Conditions = append(cert.Status.Conditions, certmanager.CertificateCondition{
			Type:    certmanager.CertificateConditionReady,
			Status:  "True",
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
		return &kwhmutating.MutatorResult{MutatedObject: cert}, nil
	}
	return &kwhmutating.MutatorResult{}, nil
}
