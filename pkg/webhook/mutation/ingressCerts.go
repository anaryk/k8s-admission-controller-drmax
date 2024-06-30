package mutating

import (
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	v1 "k8s.io/api/networking/v1"
)

func IngressCertsMutateWebhook(logger kwhlog.Logger, keyVaultName string) (kwhwebhook.Webhook, error) {
	mutators := []kwhmutating.Mutator{
		&ingressCertsMutator{logger: logger, keyVaultName: keyVaultName},
	}

	return kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "multiwebhook-ingressCertsMutator",
		Obj:     &v1.Ingress{},
		Mutator: kwhmutating.NewChain(logger, mutators...),
		Logger:  logger,
	})
}
