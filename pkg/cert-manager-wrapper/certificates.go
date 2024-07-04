package certmanagerwrapper

import (
	"context"
	"fmt"
	"time"

	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/k8s"

	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CertManagerClient struct {
	client *versioned.Clientset
}

func NewCertManagerClient() (*CertManagerClient, error) {
	restClient, err := k8s.PrepareInClusterK8SClient()
	if err != nil {
		return nil, fmt.Errorf("error creating cert-manager client: %v", err)
	}

	certManagerClient, err := versioned.NewForConfig(restClient)
	if err != nil {
		return nil, fmt.Errorf("error creating cert-manager client: %v", err)
	}
	return &CertManagerClient{client: certManagerClient}, nil
}

func (cmc *CertManagerClient) CheckIfCertificateIsReady(certificateName string, namespace string) (bool, error) {
	cert, err := cmc.client.CertmanagerV1().Certificates(namespace).Get(context.TODO(), certificateName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("error getting certificate: %v", err)
	}

	for _, condition := range cert.Status.Conditions {
		if condition.Type == certmanagerv1.CertificateConditionReady && condition.Status == cmmeta.ConditionTrue {
			return true, nil
		}
	}

	return false, fmt.Errorf("certificate %s is not ready", certificateName)
}

func (cmc *CertManagerClient) CreateFakeCertificateRequest(cert *certmanagerv1.Certificate) error {
	certRequest := &certmanagerv1.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cert.Name + "-fake-cr",
			Namespace: cert.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cert, certmanagerv1.SchemeGroupVersion.WithKind("Certificate")),
			},
		},
		Spec: certmanagerv1.CertificateRequestSpec{
			Request:   []byte("fake-csr"),
			IssuerRef: cert.Spec.IssuerRef,
		},
		Status: certmanagerv1.CertificateRequestStatus{
			Conditions: []certmanagerv1.CertificateRequestCondition{
				{
					Type:               certmanagerv1.CertificateRequestConditionReady,
					Status:             cmmeta.ConditionTrue,
					Reason:             "Fake",
					Message:            "This is a fake CertificateRequest to satisfy cert-manager",
					LastTransitionTime: &metav1.Time{Time: time.Now()},
				},
			},
		},
	}

	_, err := cmc.client.CertmanagerV1().CertificateRequests(cert.Namespace).Create(context.TODO(), certRequest, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating fake CertificateRequest: %v", err)
	}
	return nil
}
