package validating

import (
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

// NewDeploymentWebhook returns a new deployment validationg webhook.
func NewDeploymentWebhook(minReplicas, maxReplicas int, logger kwhlog.Logger) (kwhwebhook.Webhook, error) {

	// Create validators.
	repVal := &deploymentReplicasValidator{
		maxReplicas: maxReplicas,
		minReplicas: minReplicas,
		logger:      logger,
	}

	vals := []kwhvalidating.Validator{
		repVal,
	}

	return kwhvalidating.NewWebhook(
		kwhvalidating.WebhookConfig{
			ID:        "multiwebhook-deploymentValidator",
			Obj:       &extensionsv1beta1.Deployment{},
			Validator: kwhvalidating.NewChain(logger, vals...),
			Logger:    logger,
		})
}
