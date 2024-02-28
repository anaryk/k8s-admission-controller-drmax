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
	chalange, ok := obj.(*acmecertmanager.Challenge)
	if !ok {
		return &kwhmutating.MutatorResult{}, nil
	}
	if chalange.Status.State == acmecertmanager.Errored {
		m.logger.Infof("Challenge %s jump to errored state. Mutating to pending state!", chalange.Name)
		chalange.Status.State = acmecertmanager.Expired
		return &kwhmutating.MutatorResult{MutatedObject: chalange}, nil
	} else if chalange.Status.State != "" {
		m.logger.Debugf("Challenge %s is in state %s", chalange.Name, chalange.Status.State)
		return &kwhmutating.MutatorResult{}, nil
	} else {
		return &kwhmutating.MutatorResult{}, nil
	}
}
