package mutating

import (
	acmecertmanager "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
)

func CertOrderMutateWebhook(logger kwhlog.Logger) (kwhwebhook.Webhook, error) {
	mutators := []kwhmutating.Mutator{
		&certOrderMutator{logger: logger},
	}

	return kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "multiwebhook-certOrderMutator",
		Obj:     &acmecertmanager.Order{},
		Mutator: kwhmutating.NewChain(logger, mutators...),
		Logger:  logger,
	})
}
