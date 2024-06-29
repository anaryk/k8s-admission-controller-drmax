package certmanagerwrapper

import (
	"context"
	"fmt"

	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/k8s"

	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CheckIfCertificateIsReady(certificateName string, namespace string) (bool, error) {
	restClient, err := k8s.PrepareLocalKubeconfigK8SClient()
	if err != nil {
		return false, fmt.Errorf("error creating cert-manager client: %v", err)
	}

	certManagerClient, err := versioned.NewForConfig(restClient)
	if err != nil {
		return false, fmt.Errorf("error creating cert-manager client: %v", err)
	}

	cert, err := certManagerClient.CertmanagerV1().Certificates(namespace).Get(context.TODO(), certificateName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("error getting certificate: %v", err)
	}

	for _, condition := range cert.Status.Conditions {
		if condition.Type == certmanagerv1.CertificateConditionReady {
			return true, nil
		}
	}
	return false, fmt.Errorf("certificate %s is not ready", certificateName)
}
