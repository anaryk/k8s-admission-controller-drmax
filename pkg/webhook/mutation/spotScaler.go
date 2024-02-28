package mutating

import (
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	apps "k8s.io/api/apps/v1"
)

func SpotScalerMutateWebhook(logger kwhlog.Logger) (kwhwebhook.Webhook, error) {
	mutators := []kwhmutating.Mutator{
		&SpotScalerMutator{logger: logger},
	}

	return kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "multiwebhook-spotScalerutator",
		Obj:     &apps.Deployment{},
		Mutator: kwhmutating.NewChain(logger, mutators...),
		Logger:  logger,
	})
}
