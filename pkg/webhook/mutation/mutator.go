package mutating

import (
	"context"
	"strconv"
	"strings"

	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/utils"
	acmecertmanager "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
		m.logger.Infof("Challenge %s jump to errored state. Mutating to pending state!", chalange.Name)
		chalange.Status.State = acmecertmanager.Pending
		chalange.Status.Reason = "Mutated by DrMax admission webhook, bacause previous order ended up in error state due to ZeroSSL nginx proxy overload (due error)"
		return &kwhmutating.MutatorResult{MutatedObject: chalange}, nil
	} else if chalange.Status.State != "" {
		m.logger.Debugf("Challenge %s is in state %s", chalange.Name, chalange.Status.State)
		return &kwhmutating.MutatorResult{}, nil
	} else {
		return &kwhmutating.MutatorResult{}, nil
	}
}

//SpotScalerMutateWebhook creates a new mutating webhook for SpotScaler

type SpotScalerMutator struct {
	logger kwhlog.Logger
}

func (m *SpotScalerMutator) Mutate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	deployment, okDep := obj.(*apps.Deployment)
	pod, okPod := obj.(*corev1.Pod)
	if !okDep || !okPod {
		return &kwhmutating.MutatorResult{}, nil
	}
	var numberOfguaranteedAnnotation string = "spot-scaler.drmax.global/guaranteed"
	var numberOfbesteffordAnnotation string = "spot-scaler.drmax.global/bestefford"
	var enabledAnnotation string = "spot-scaler.drmax.global/enabled"

	//Validate deployment annotation and check if soutable fot mutation
	if deployment.Annotations[enabledAnnotation] == "true" && deployment.Annotations[numberOfguaranteedAnnotation] != "" && deployment.Annotations[numberOfbesteffordAnnotation] != "" {
		if utils.ValidateIntFiled(deployment.Annotations[numberOfguaranteedAnnotation]) && utils.ValidateIntFiled(deployment.Annotations[numberOfbesteffordAnnotation]) {
			m.logger.Debugf("Deployment %s have all required annotation for correct work", deployment.Name)
		} else {
			m.logger.Debugf("Deployment %s have all required annotation for correct work", deployment.Name)
			return &kwhmutating.MutatorResult{}, nil
		}
		numberOfguaranteed, _ := strconv.Atoi(deployment.Annotations[numberOfguaranteedAnnotation])
		numberOfguaranteed32 := int32(numberOfguaranteed)
		if *deployment.Spec.Replicas >= numberOfguaranteed32 {
			pod.Spec.Tolerations = append(deployment.Spec.Template.Spec.Tolerations, corev1.Toleration{
				Key:      "kubernetes.azure.com/scalesetpriority",
				Value:    "spot",
				Operator: corev1.TolerationOpEqual,
				Effect:   corev1.TaintEffectNoSchedule,
			})
			return &kwhmutating.MutatorResult{MutatedObject: pod}, nil
		}
	} else {
		m.logger.Debugf("Deployment %s dont have all required annotation for correct work", deployment.Name)
		return nil, nil
	}
	return &kwhmutating.MutatorResult{}, nil
}
