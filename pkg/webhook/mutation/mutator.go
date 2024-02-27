package mutating

import (
	"context"

	acmecertmanager "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CertOrderMutateWebhook fixing issue with order state on ZeroSSL
type certOrderMutator struct {
	logger kwhlog.Logger
}

func (m *certOrderMutator) Mutate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	order, ok := obj.(*acmecertmanager.Order)
	if !ok {
		return &kwhmutating.MutatorResult{}, nil
	}

	if order.Status.State == acmecertmanager.Errored {
		m.logger.Infof("Order %s jump to errored state. Mutating to pending state!", order.Name)
		order.Status.State = acmecertmanager.Pending
		m.logger.Infof("Order %s mutated to pending state!", order.Name)
	} else {
		m.logger.Debugf("Order %s is in %s state. No need to mutate.", order.Name, order.Status.Reason)
	}
	return &kwhmutating.MutatorResult{MutatedObject: obj}, nil
}
