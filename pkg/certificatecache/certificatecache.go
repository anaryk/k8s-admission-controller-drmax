package certificatecache

import (
	"context"
	"fmt"
	"time"

	azurewrapper "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/azure"
	certmanagerwrapper "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/cert-manager-wrapper"
	"github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

type CertificateCacheManager struct {
	k8sClient         *kubernetes.Clientset
	keyVaultClient    *azurewrapper.KeyVaultClient
	certManagerClient *versioned.Clientset
	logger            kwhlog.Logger
}

func NewCertificateCacheManager(k8sClient *kubernetes.Clientset, keyVaultClient *azurewrapper.KeyVaultClient, certManagerClient *versioned.Clientset, logger kwhlog.Logger) *CertificateCacheManager {
	return &CertificateCacheManager{
		k8sClient:         k8sClient,
		keyVaultClient:    keyVaultClient,
		certManagerClient: certManagerClient,
		logger:            logger,
	}
}

func (ccm *CertificateCacheManager) CheckAndCacheCertificates() error {
	ingressList, err := ccm.k8sClient.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list ingress objects: %w", err)
	}

	for _, ingress := range ingressList.Items {
		if ingress.Annotations["admissions.drmax.gl/cert-scheduled-for-save"] == "true" &&
			ingress.Annotations["admissions.drmax.gl/cache-certs"] == "true" {

			secretName := ingress.Spec.TLS[0].SecretName
			namespace := ingress.Namespace

			existReady, err := certmanagerwrapper.CheckIfCertificateIsReady(secretName, namespace)
			if err != nil {
				//comment this error due to the fact that it is not a critical error
				//its only spamming while some cert are not ready for longer time
				//ccm.logger.Errorf("failed to check if certificate is ready: %v", err)
				continue
			}

			if existReady {
				// Get the Kubernetes Secret
				secret, err := ccm.k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
				if err != nil {
					ccm.logger.Errorf("failed to get Kubernetes secret: %v", err)
					continue
				}

				cert := secret.Data["tls.crt"]
				key := secret.Data["tls.key"]

				// Store the cert and key in Azure Key Vault
				vaultSecretName := fmt.Sprintf("%s--%s", secretName, namespace)
				err = ccm.keyVaultClient.StoreSecret(context.Background(), vaultSecretName, cert, key)
				if err != nil {
					ccm.logger.Errorf("failed to store secret in key vault: %v", err)
					continue
				}

				err = ccm.updateIngressAnnotations(&ingress, map[string]string{
					"admissions.drmax.gl/cert-cached":             "true",
					"admissions.drmax.gl/cert-scheduled-for-save": "false",
				})
				if err != nil {
					ccm.logger.Errorf("failed to update ingress annotations: %v", err)
				}

				ccm.logger.Infof("certificate for ingress %s is stored in Azure KeyVault and correctly marked using annotations", ingress.Name)

			}
		}
	}

	return nil
}

func (ccm *CertificateCacheManager) CleanupExpiringCertificates() error {
	ingressList, err := ccm.k8sClient.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list ingress objects: %w", err)
	}

	for _, ingress := range ingressList.Items {
		if ingress.Annotations["admissions.drmax.gl/cert-cached"] == "true" {
			secretName := ingress.Spec.TLS[0].SecretName
			namespace := ingress.Namespace

			secret := fmt.Sprintf("%s--%s", secretName, namespace)
			expiry, err := ccm.keyVaultClient.GetCertificateExpiry(context.Background(), secret)
			if err != nil {
				ccm.logger.Errorf("failed to get certificate expiry from key vault: %v", err)
				continue
			}

			if time.Now().AddDate(0, 1, 0).After(expiry) {
				ccm.logger.Debugf("certificate for ingress %s is expiring in less then one month", ingress.Name)
				err = ccm.keyVaultClient.DeleteSecret(context.Background(), secret)
				if err != nil {
					ccm.logger.Errorf("failed to delete secret from key vault: %v", err)
					continue
				}

				err = ccm.updateIngressAnnotations(&ingress, map[string]string{
					"admissions.drmax.gl/cert-cached":             "false",
					"admissions.drmax.gl/cert-scheduled-for-save": "true",
				})
				if err != nil {
					ccm.logger.Errorf("failed to update ingress annotations: %v", err)
				}

				err = ccm.updateCertificateAnnotations(secretName, namespace, map[string]string{
					"admissions.drmax.gl/cert-cached": "false",
				})
				if err != nil {
					ccm.logger.Errorf("failed to update certificate annotations: %v", err)
				}

				// Delete the Kubernetes Secret results insto cert manager mark Certificate as not ready and validation
				// skip this certificate to be fully populated.

				err = ccm.k8sClient.CoreV1().Secrets(namespace).Delete(context.Background(), secretName, metav1.DeleteOptions{})
				if err != nil {
					ccm.logger.Errorf("failed to delete secret from k8s: %v", err)
				}

				ccm.logger.Infof("certificate for ingress %s is expired and deleted from Azure KeyVault (Deleting also secret to set certificate tu Ready -> False state)", ingress.Name)
			}

			ccm.logger.Debugf("certificate for ingress %s is not expiring in less then one month (Time of expire %s, Time of cache removal %s)", ingress.Name, expiry.String(), expiry.AddDate(0, 1, 0).String())
		}
	}

	return nil
}

func (ccm *CertificateCacheManager) updateIngressAnnotations(ingress *v1.Ingress, annotations map[string]string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		for key, value := range annotations {
			ingress.Annotations[key] = value
		}
		_, err := ccm.k8sClient.NetworkingV1().Ingresses(ingress.Namespace).Update(context.TODO(), ingress, metav1.UpdateOptions{})
		return err
	})
}

func (ccm *CertificateCacheManager) updateCertificateAnnotations(certName, namespace string, annotations map[string]string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cert, err := ccm.certManagerClient.CertmanagerV1().Certificates(namespace).Get(context.TODO(), certName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for key, value := range annotations {
			cert.Annotations[key] = value
		}
		_, err = ccm.certManagerClient.CertmanagerV1().Certificates(namespace).Update(context.TODO(), cert, metav1.UpdateOptions{})
		return err
	})
}
