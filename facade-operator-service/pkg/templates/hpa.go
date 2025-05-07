package templates

import (
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	hpav2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var hpaLabelsToPropagate = []string{
	"app.kubernetes.io/part-of",
}

type HPATemplate struct {
	DeploymentName string
	Namespace      string
	CrLabels       map[string]string
	OwnerReference metav1.OwnerReference

	MinReplicas        *int32
	MaxReplicas        int32
	AverageUtilization *int32

	ScaleUpStabilizationWindowSeconds *int32
	ScaleUpScalingPolicySelect        *hpav2.ScalingPolicySelect
	ScaleUpPolicies                   []hpav2.HPAScalingPolicy

	ScaleDownStabilizationWindowSeconds *int32
	ScaleDownScalingPolicySelect        *hpav2.ScalingPolicySelect
	ScaleDownPolicies                   []hpav2.HPAScalingPolicy
}

func (h *HPATemplate) GetObject() *hpav2.HorizontalPodAutoscaler {
	return &hpav2.HorizontalPodAutoscaler{
		ObjectMeta: h.getObjectMeta(),
		Spec:       h.getSpec(),
	}
}

func (h *HPATemplate) getObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      h.DeploymentName,
		Namespace: h.Namespace,
		Labels: h.propagateLabels(map[string]string{
			"app.kubernetes.io/name":                h.DeploymentName,
			"app.kubernetes.io/managed-by":          "operator",
			"app.kubernetes.io/managed-by-operator": "facade-operator",
		}),
		OwnerReferences: []metav1.OwnerReference{h.OwnerReference},
	}
}

func (h *HPATemplate) propagateLabels(labels map[string]string) map[string]string {
	for _, label := range hpaLabelsToPropagate {
		if labelVal, ok := h.CrLabels[label]; ok {
			labels[label] = labelVal
		} else {
			labels[label] = utils.Unknown
		}
	}
	return labels
}

func (h *HPATemplate) getSpec() hpav2.HorizontalPodAutoscalerSpec {
	return hpav2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: hpav2.CrossVersionObjectReference{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
			Name:       h.DeploymentName,
		},
		MinReplicas: h.MinReplicas,
		MaxReplicas: h.MaxReplicas,
		Metrics: []hpav2.MetricSpec{
			{
				Type: hpav2.ResourceMetricSourceType,
				Resource: &hpav2.ResourceMetricSource{
					Name: "cpu",
					Target: hpav2.MetricTarget{
						Type:               hpav2.UtilizationMetricType,
						AverageUtilization: h.AverageUtilization,
					},
				},
			},
		},
		Behavior: &hpav2.HorizontalPodAutoscalerBehavior{
			ScaleUp: &hpav2.HPAScalingRules{
				StabilizationWindowSeconds: h.ScaleUpStabilizationWindowSeconds,
				SelectPolicy:               h.ScaleUpScalingPolicySelect,
				Policies:                   h.ScaleUpPolicies,
			},
			ScaleDown: &hpav2.HPAScalingRules{
				StabilizationWindowSeconds: h.ScaleDownStabilizationWindowSeconds,
				SelectPolicy:               h.ScaleDownScalingPolicySelect,
				Policies:                   h.ScaleDownPolicies,
			},
		},
	}
}
