package mutating

import (
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
)

func SpotScalerMutateWebhook(logger kwhlog.Logger) (kwhwebhook.Webhook, error) {
	mutators := []kwhmutating.Mutator{
		&SpotScalerMutator{logger: logger},
	}

	return kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "multiwebhook-spotScalerutator",
		Obj:     &corev1.Pod{},
		Mutator: kwhmutating.NewChain(logger, mutators...),
		Logger:  logger,
	})
}
