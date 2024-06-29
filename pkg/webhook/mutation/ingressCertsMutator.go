package mutating

import (
	"context"

	certmanagerwrapper "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/cert-manager-wrapper"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ingressCertsMutator struct {
	logger kwhlog.Logger
}

func (m *ingressCertsMutator) Mutate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	ingressObj, ok := obj.(*v1.Ingress)
	if !ok {
		return &kwhmutating.MutatorResult{}, nil
	}
	if ingressObj.Annotations["admissions.drmax.gl/cache-certs"] == "true" {
		m.logger.Infof("Ingress %s has cache-certs annotation. checking if certificate is issued!", ingressObj.Name)
		existReady, err := certmanagerwrapper.CheckIfCertificateIsReady(ingressObj.Spec.TLS[0].SecretName, ingressObj.Namespace)
		if err != nil {
			m.logger.Errorf("Error checking if certificate is ready: %v", err)
		}
		if existReady && ingressObj.Annotations["admissions.drmax.gl/cert-cached"] != "true" {
			m.logger.Infof("Certificate for ingress %s is ready. Marking this ingress and certificate for save to cache", ingressObj.Name)
			ingressObj.Annotations["admissions.drmax.gl/cert-scheduled-for-save"] = "true"
			return &kwhmutating.MutatorResult{MutatedObject: ingressObj}, nil
		} else {
			m.logger.Infof("Certificate for ingress %s is not ready or already loaded from cache!", ingressObj.Name)
		}
	}
	return &kwhmutating.MutatorResult{}, nil
}
