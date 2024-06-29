package mutating

import (
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
)

func CertificateCacheMutateWebhook(logger kwhlog.Logger, keyVaultName string) (kwhwebhook.Webhook, error) {
	mutators := []kwhmutating.Mutator{
		&certificateCaheMutator{logger: logger, keyVaultName: keyVaultName},
	}

	return kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "multiwebhook-certificateCaheMutator",
		Obj:     &certmanager.Certificate{},
		Mutator: kwhmutating.NewChain(logger, mutators...),
		Logger:  logger,
	})
}
