package mutating

import (
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
		Mutator: kwhmutating.NewChain(logger, mutators...),
		Logger:  logger,
	})
}
