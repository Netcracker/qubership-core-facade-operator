package builder

import (
	"context"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	hpav2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestBuildDefault(t *testing.T) {
	hpa := facade.HPA{}

	builder := NewHPATemplateBuilder()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "testName",
		},
	}
	cr := &facadeV1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: "core.netcracker.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cr-name",
			UID:  "12345",
		},
		Spec: facade.FacadeServiceSpec{
			Hpa: hpa,
		},
	}
	result := builder.Build(context.Background(), req, cr, "deploymentName")

	assert.Equal(t, result.ObjectMeta.Name, "deploymentName")
	validateMap(t, result.ObjectMeta.Labels, map[string]string{
		"app.kubernetes.io/part-of":             "unknown",
		"app.kubernetes.io/name":                "deploymentName",
		"app.kubernetes.io/managed-by":          "operator",
		"app.kubernetes.io/managed-by-operator": "facade-operator",
	})

	assert.Equal(t, len(result.ObjectMeta.OwnerReferences), 1)
	assert.Equal(t, result.ObjectMeta.OwnerReferences[0].APIVersion, cr.APIVersion)
	assert.Equal(t, result.ObjectMeta.OwnerReferences[0].Kind, cr.Kind)
	assert.Equal(t, result.ObjectMeta.OwnerReferences[0].Name, cr.Name)
	assert.Equal(t, result.ObjectMeta.OwnerReferences[0].UID, cr.UID)

	assert.Equal(t, result.Spec.ScaleTargetRef.Kind, "Deployment")
	assert.Equal(t, result.Spec.ScaleTargetRef.Name, "deploymentName")
	assert.Equal(t, result.Spec.ScaleTargetRef.APIVersion, "apps/v1")

	assert.Equal(t, result.Spec.MinReplicas, utils.GetPointer(int32(1)))
	assert.Equal(t, result.Spec.MaxReplicas, int32(9999))

	assert.Equal(t, result.Spec.Metrics[0].Type, hpav2.ResourceMetricSourceType)
	assert.Equal(t, string(result.Spec.Metrics[0].Resource.Name), "cpu")
	assert.Equal(t, result.Spec.Metrics[0].Resource.Target.Type, hpav2.UtilizationMetricType)
	assert.Equal(t, result.Spec.Metrics[0].Resource.Target.AverageUtilization, utils.GetPointer(int32(75)))

	assert.Equal(t, result.Spec.Behavior.ScaleUp.StabilizationWindowSeconds, utils.GetPointer(int32(60)))
	assert.Equal(t, result.Spec.Behavior.ScaleUp.SelectPolicy, utils.GetPointer(hpav2.DisabledPolicySelect))
	assert.Equal(t, len(result.Spec.Behavior.ScaleUp.Policies), 1)
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[0].Type, hpav2.PodsScalingPolicy)
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[0].Value, int32(1))
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[0].PeriodSeconds, int32(60))

	assert.Equal(t, result.Spec.Behavior.ScaleDown.StabilizationWindowSeconds, utils.GetPointer(int32(300)))
	assert.Equal(t, result.Spec.Behavior.ScaleDown.SelectPolicy, utils.GetPointer(hpav2.DisabledPolicySelect))
	assert.Equal(t, len(result.Spec.Behavior.ScaleDown.Policies), 1)
	assert.Equal(t, result.Spec.Behavior.ScaleDown.Policies[0].Type, hpav2.PodsScalingPolicy)
	assert.Equal(t, result.Spec.Behavior.ScaleDown.Policies[0].Value, int32(1))
	assert.Equal(t, result.Spec.Behavior.ScaleDown.Policies[0].PeriodSeconds, int32(60))
}

func TestBuild(t *testing.T) {
	os.Setenv("GATEWAY_HPA_MIN_REPLICAS", "100500")
	defer os.Unsetenv("GATEWAY_HPA_MIN_REPLICAS")
	hpa := facade.HPA{
		MaxReplicas:           "2",
		AverageCpuUtilization: 1,
		ScaleUpBehavior: facade.HPABehavior{
			StabilizationWindowSeconds: "10",
			SelectPolicy:               "Max",
			Policies: []facade.HPAPolicies{
				{
					Type:          "Percent",
					Value:         "10test",
					PeriodSeconds: 10,
				},
				{
					Type:          "Pods",
					Value:         10,
					PeriodSeconds: 10,
				},
				{
					Value: 11,
				},
			},
		},
		ScaleDownBehavior: facade.HPABehavior{
			StabilizationWindowSeconds: 11,
		},
	}

	builder := NewHPATemplateBuilder()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "testName",
		},
	}
	cr := &facadeV1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: "core.netcracker.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cr-name",
			UID:  "12345",
		},
		Spec: facade.FacadeServiceSpec{
			Hpa: hpa,
		},
	}
	result := builder.Build(context.Background(), req, cr, "deploymentName")

	assert.Equal(t, result.ObjectMeta.Name, "deploymentName")
	validateMap(t, result.ObjectMeta.Labels, map[string]string{
		"app.kubernetes.io/part-of":             "unknown",
		"app.kubernetes.io/name":                "deploymentName",
		"app.kubernetes.io/managed-by":          "operator",
		"app.kubernetes.io/managed-by-operator": "facade-operator",
	})

	assert.Equal(t, len(result.ObjectMeta.OwnerReferences), 1)
	assert.Equal(t, result.ObjectMeta.OwnerReferences[0].APIVersion, cr.APIVersion)
	assert.Equal(t, result.ObjectMeta.OwnerReferences[0].Kind, cr.Kind)
	assert.Equal(t, result.ObjectMeta.OwnerReferences[0].Name, cr.Name)
	assert.Equal(t, result.ObjectMeta.OwnerReferences[0].UID, cr.UID)

	assert.Equal(t, result.Spec.ScaleTargetRef.Kind, "Deployment")
	assert.Equal(t, result.Spec.ScaleTargetRef.Name, "deploymentName")
	assert.Equal(t, result.Spec.ScaleTargetRef.APIVersion, "apps/v1")

	assert.Equal(t, result.Spec.MinReplicas, utils.GetPointer(int32(100500)))
	assert.Equal(t, result.Spec.MaxReplicas, int32(2))

	assert.Equal(t, result.Spec.Metrics[0].Type, hpav2.ResourceMetricSourceType)
	assert.Equal(t, string(result.Spec.Metrics[0].Resource.Name), "cpu")
	assert.Equal(t, result.Spec.Metrics[0].Resource.Target.Type, hpav2.UtilizationMetricType)
	assert.Equal(t, result.Spec.Metrics[0].Resource.Target.AverageUtilization, utils.GetPointer(int32(1)))

	assert.Equal(t, result.Spec.Behavior.ScaleUp.StabilizationWindowSeconds, utils.GetPointer(int32(10)))
	assert.Equal(t, result.Spec.Behavior.ScaleUp.SelectPolicy, utils.GetPointer(hpav2.MaxChangePolicySelect))
	assert.Equal(t, len(result.Spec.Behavior.ScaleUp.Policies), 3)
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[0].Type, hpav2.PercentScalingPolicy)
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[0].Value, int32(-1))
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[0].PeriodSeconds, int32(10))
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[1].Type, hpav2.PodsScalingPolicy)
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[1].Value, int32(10))
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[1].PeriodSeconds, int32(10))
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[2].Type, hpav2.PodsScalingPolicy)
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[2].Value, int32(11))
	assert.Equal(t, result.Spec.Behavior.ScaleUp.Policies[2].PeriodSeconds, int32(60))

	assert.Equal(t, result.Spec.Behavior.ScaleDown.StabilizationWindowSeconds, utils.GetPointer(int32(11)))
	assert.Equal(t, result.Spec.Behavior.ScaleDown.SelectPolicy, utils.GetPointer(hpav2.DisabledPolicySelect))
	assert.Equal(t, len(result.Spec.Behavior.ScaleDown.Policies), 1)
	assert.Equal(t, result.Spec.Behavior.ScaleDown.Policies[0].Type, hpav2.PodsScalingPolicy)
	assert.Equal(t, result.Spec.Behavior.ScaleDown.Policies[0].Value, int32(1))
	assert.Equal(t, result.Spec.Behavior.ScaleDown.Policies[0].PeriodSeconds, int32(60))
}

func validateMap(t *testing.T, current map[string]string, expected map[string]string) {
	assert.Equal(t, len(current), len(expected))
	for k, v := range current {
		assert.Equal(t, v, expected[k])
	}
}
