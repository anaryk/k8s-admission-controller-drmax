package mutating

import (
	"context"
	"strings"

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
	if chalange.Status.State == acmecertmanager.Errored && strings.Contains(chalange.Status.Reason, "429") {
		chalange.Status.State = acmecertmanager.Pending
		chalange.Status.Reason = "Mutated by DrMax admission webhook, bacause previous order ended up in error state due to ZeroSSL nginx proxy overload (due error)"
		m.logger.Infof("--- MUTATED --- Challenge %s is mutated back to pending state", chalange.Name)
		return &kwhmutating.MutatorResult{MutatedObject: chalange}, nil
	} else if chalange.Status.State != "" {
		m.logger.Debugf("Challenge %s is in state %s", chalange.Name, chalange.Status.State)
		return &kwhmutating.MutatorResult{}, nil
	} else {
		return &kwhmutating.MutatorResult{}, nil
	}
}
