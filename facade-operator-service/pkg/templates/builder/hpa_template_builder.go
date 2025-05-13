package builder

import (
	"context"
	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	hpav2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type HPATemplateBuilder interface {
	Build(ctx context.Context, req ctrl.Request, cr facade.MeshGateway, deploymentName string) *hpav2.HorizontalPodAutoscaler
}

type HPATemplateBuilderImpl struct {
	logger logging.Logger

	//hpa
	hpaDefaultMinReplicas        int32
	hpaDefaultMaxReplicas        int32
	hpaDefaultAverageUtilization int32
	hpaDefaultSelectPolicy       hpav2.ScalingPolicySelect

	//hpa - scale up
	hpaDefaultScaleUpStabilizationWindowSeconds int32
	hpaDefaultScaleUpPercentValue               int32
	hpaDefaultScaleUpPercentPeriodSeconds       int32
	hpaDefaultScaleUpPodsValue                  int32
	hpaDefaultScaleUpPodsPeriodSeconds          int32

	//hpa - scale down
	hpaDefaultScaleDownStabilizationWindowSeconds int32
	hpaDefaultScaleDownPercentValue               int32
	hpaDefaultScaleDownPercentPeriodSeconds       int32
	hpaDefaultScaleDownPodsValue                  int32
	hpaDefaultScaleDownPodsPeriodSeconds          int32
}

func NewHPATemplateBuilder() HPATemplateBuilder {
	return &HPATemplateBuilderImpl{
		logger:                       logging.GetLogger("HPATemplateBuilderImpl"),
		hpaDefaultMinReplicas:        utils.GetInt32EnvValueOrDefault(utils.HpaDefaultMinReplicasEnvName, int32(1)),
		hpaDefaultMaxReplicas:        utils.GetInt32EnvValueOrDefault(utils.HpaDefaultMaxReplicasEnvName, int32(9999)),
		hpaDefaultAverageUtilization: utils.GetInt32EnvValueOrDefault(utils.HpaDefaultAverageUtilizationEnvName, int32(75)),
		hpaDefaultSelectPolicy:       hpav2.DisabledPolicySelect,

		hpaDefaultScaleUpStabilizationWindowSeconds: utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleUpStabilizationWindowSecondsEnvName, int32(60)),
		hpaDefaultScaleUpPercentValue:               utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleUpPercentValueEnvName, int32(-1)),
		hpaDefaultScaleUpPercentPeriodSeconds:       utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleUpPercentPeriodSecondsEnvName, int32(-1)),
		hpaDefaultScaleUpPodsValue:                  utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleUpPodsValueEnvName, int32(1)),
		hpaDefaultScaleUpPodsPeriodSeconds:          utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleUpPodsPeriodSecondsEnvName, int32(60)),

		hpaDefaultScaleDownStabilizationWindowSeconds: utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleDownStabilizationWindowSecondsEnvName, int32(300)),
		hpaDefaultScaleDownPercentValue:               utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleDownPercentValueEnvName, int32(-1)),
		hpaDefaultScaleDownPercentPeriodSeconds:       utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleDownPercentPeriodSecondsEnvName, int32(-1)),
		hpaDefaultScaleDownPodsValue:                  utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleDownPodsValueEnvName, int32(1)),
		hpaDefaultScaleDownPodsPeriodSeconds:          utils.GetInt32EnvValueOrDefault(utils.HpaDefaultScaleDownPodsPeriodSecondsEnvName, int32(60)),
	}
}

func (b *HPATemplateBuilderImpl) Build(ctx context.Context, req ctrl.Request, cr facade.MeshGateway, deploymentName string) *hpav2.HorizontalPodAutoscaler {
	hpa := cr.GetSpec().Hpa
	hpaTemplate := templates.HPATemplate{
		DeploymentName: deploymentName,
		Namespace:      cr.GetNamespace(),
		CrLabels:       cr.GetLabels(),
		OwnerReference: metav1.OwnerReference{
			APIVersion: cr.GetAPIVersion(),
			Kind:       cr.GetKind(),
			Name:       cr.GetName(),
			UID:        cr.GetUID(),
			Controller: utils.GetPointer(false),
		},

		MinReplicas:        utils.GetPointer(b.getValueOrDefault(ctx, req, hpa.MinReplicas, b.hpaDefaultMinReplicas, "hpa.minReplicas")),
		MaxReplicas:        b.getValueOrDefault(ctx, req, hpa.MaxReplicas, b.hpaDefaultMaxReplicas, "hpa.maxReplicas"),
		AverageUtilization: utils.GetPointer(b.getValueOrDefault(ctx, req, hpa.AverageCpuUtilization, b.hpaDefaultAverageUtilization, "hpa.averageCpuUtilization")),

		ScaleUpStabilizationWindowSeconds: utils.GetPointer(b.getValueOrDefault(
			ctx, req,
			hpa.ScaleUpBehavior.StabilizationWindowSeconds,
			b.hpaDefaultScaleUpStabilizationWindowSeconds,
			"hpa.scaleUpBehavior.stabilizationWindowSeconds",
		)),
		ScaleUpScalingPolicySelect: utils.GetPointer(b.getPolicySelect(ctx, req, hpa.ScaleUpBehavior.SelectPolicy)),
		ScaleUpPolicies:            b.getScaleUpBehaviorPolicies(ctx, req, hpa.ScaleUpBehavior.Policies),

		ScaleDownStabilizationWindowSeconds: utils.GetPointer(b.getValueOrDefault(
			ctx, req,
			hpa.ScaleDownBehavior.StabilizationWindowSeconds,
			b.hpaDefaultScaleDownStabilizationWindowSeconds,
			"hpa.scaleDownBehavior.stabilizationWindowSeconds",
		)),
		ScaleDownScalingPolicySelect: utils.GetPointer(b.getPolicySelect(ctx, req, hpa.ScaleDownBehavior.SelectPolicy)),
		ScaleDownPolicies:            b.getScaleDownBehaviorPolicies(ctx, req, hpa.ScaleDownBehavior.Policies),
	}

	return hpaTemplate.GetObject()
}

func (b *HPATemplateBuilderImpl) getPolicySelect(ctx context.Context, req ctrl.Request, policy string) hpav2.ScalingPolicySelect {
	switch policy {
	case string(hpav2.MaxChangePolicySelect):
		return hpav2.MaxChangePolicySelect
	case string(hpav2.MinChangePolicySelect):
		return hpav2.MinChangePolicySelect
	case string(hpav2.DisabledPolicySelect):
		return hpav2.DisabledPolicySelect
	default:
		b.logger.WarnC(ctx, "[%v] Can not parse scale policy with value '%s'. Use default value '%s'", req.NamespacedName, policy, string(b.hpaDefaultSelectPolicy))
		return b.hpaDefaultSelectPolicy
	}
}

func (b *HPATemplateBuilderImpl) getScaleUpBehaviorPolicies(ctx context.Context, req ctrl.Request, policies []facade.HPAPolicies) []hpav2.HPAScalingPolicy {
	return b.getBehaviorPolicies(
		ctx, req,
		policies,
		b.hpaDefaultScaleUpPodsValue,
		b.hpaDefaultScaleUpPodsPeriodSeconds,
		b.hpaDefaultScaleUpPercentValue,
		b.hpaDefaultScaleUpPercentPeriodSeconds,
		"hpa.scaleUpBehavior.",
	)
}

func (b *HPATemplateBuilderImpl) getScaleDownBehaviorPolicies(ctx context.Context, req ctrl.Request, policies []facade.HPAPolicies) []hpav2.HPAScalingPolicy {
	return b.getBehaviorPolicies(
		ctx, req,
		policies,
		b.hpaDefaultScaleDownPodsValue,
		b.hpaDefaultScaleDownPodsPeriodSeconds,
		b.hpaDefaultScaleDownPercentValue,
		b.hpaDefaultScaleDownPercentPeriodSeconds,
		"hpa.scaleDownBehavior.",
	)
}

func (b *HPATemplateBuilderImpl) getBehaviorPolicies(ctx context.Context, req ctrl.Request,
	policies []facade.HPAPolicies, defaultPodsValue, defaultPodsPeriodSeconds,
	defaultPercentValue, defaultPercentPeriodSeconds int32, fieldPrefix string,
) []hpav2.HPAScalingPolicy {
	if len(policies) == 0 {
		b.logger.WarnC(ctx, "[%v] field '%s' is empty. Will be used default value", req.NamespacedName, fieldPrefix+"policies")
		return []hpav2.HPAScalingPolicy{
			{
				Type:          hpav2.PodsScalingPolicy,
				Value:         defaultPodsValue,
				PeriodSeconds: defaultPodsPeriodSeconds,
			},
		}
	}

	result := make([]hpav2.HPAScalingPolicy, len(policies))
	for i, policy := range policies {
		policyType := b.getPolicyType(ctx, req, policy.Type)
		result[i] = hpav2.HPAScalingPolicy{
			Type: policyType,
			Value: b.getValueOrDefault(
				ctx, req,
				policy.Value,
				b.getDefaultByPolicyType(policyType, defaultPodsValue, defaultPercentValue),
				fieldPrefix+"policies.value",
			),
			PeriodSeconds: b.getValueOrDefault(
				ctx, req,
				policy.PeriodSeconds,
				b.getDefaultByPolicyType(policyType, defaultPodsPeriodSeconds, defaultPercentPeriodSeconds),
				fieldPrefix+"policies.periodSeconds",
			),
		}
	}

	return result
}

func (b *HPATemplateBuilderImpl) getDefaultByPolicyType(policyType hpav2.HPAScalingPolicyType, defaultPods, defaultPercent int32) int32 {
	switch policyType {
	case hpav2.PodsScalingPolicy:
		return defaultPods
	default:
		return defaultPercent
	}
}

func (b *HPATemplateBuilderImpl) getPolicyType(ctx context.Context, req ctrl.Request, policy string) hpav2.HPAScalingPolicyType {
	switch policy {
	case string(hpav2.PodsScalingPolicy):
		return hpav2.PodsScalingPolicy
	case string(hpav2.PercentScalingPolicy):
		return hpav2.PercentScalingPolicy
	default:
		b.logger.WarnC(
			ctx, "[%v] Can not parse policy type with value '%s'. Use default value '%s'",
			req.NamespacedName, policy, string(hpav2.PodsScalingPolicy),
		)
		return hpav2.PodsScalingPolicy
	}
}

func (b *HPATemplateBuilderImpl) getValueOrDefault(ctx context.Context, req ctrl.Request, value any, defaultValue int32, fieldName string) int32 {
	convertedValue, err := utils.ConvertToInt32(value)
	if err != nil {
		b.logger.WarnC(
			ctx, "[%v] Can not parse '%s' with value '%s' from HPA config. Will be used default value '%d'.",
			req.NamespacedName, fieldName, fmt.Sprintf("%v", value), defaultValue,
		)
		return defaultValue
	}
	return convertedValue
}
