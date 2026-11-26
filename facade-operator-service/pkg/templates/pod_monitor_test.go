package templates

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	v1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/monitoring/v1"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetPodMonitor_WithAllFields(t *testing.T) {
	podMonitor := buildPodMonitor("test-monitor", "default", "test-app", "test-part", "test-selector")

	result := podMonitor.GetPodMonitor()

	assert.NotNil(t, result)
	assert.Equal(t, "test-monitor", result.Name)
	assert.Equal(t, "default", result.Namespace)
	assert.Equal(t, "test-app", result.Labels["k8s-app"])
	assert.Equal(t, "test-part", result.Labels["app.kubernetes.io/part-of"])
	assert.Len(t, result.OwnerReferences, 1)
	assert.Equal(t, "test-selector", result.Spec.Selector.MatchLabels["name"])
}

func TestGetPodMonitor_WithEmptyPartOfLabel(t *testing.T) {
	podMonitor := buildPodMonitor("test-monitor", "default", "test-app", "", "test-selector")

	result := podMonitor.GetPodMonitor()

	assert.Equal(t, utils.Unknown, result.Labels["app.kubernetes.io/part-of"])
}

func TestGetPodMonitor_LabelsContainRequiredKeys(t *testing.T) {
	podMonitor := buildPodMonitor("test-monitor", "default", "test-app", "test-part", "test-selector")

	result := podMonitor.GetPodMonitor()
	labels := result.Labels

	assert.Equal(t, "monitoring", labels["app.kubernetes.io/component"])
	assert.Equal(t, "operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "facade-operator", labels["app.kubernetes.io/managed-by-operator"])
	assert.Equal(t, "victoriametrics-operator", labels["app.kubernetes.io/processed-by-operator"])
}

func TestGetPodMetricsEndpoint_ReturnsCorrectConfig(t *testing.T) {
	podMonitor := buildPodMonitor("test-monitor", "default", "test-app", "test-part", "test-selector")

	endpoints := podMonitor.getPodMetricsEndpoint()

	assert.Len(t, endpoints, 1)
	endpoint := endpoints[0]
	assert.Equal(t, "30s", endpoint.Interval)
	assert.Equal(t, "admin", endpoint.Port)
	assert.Equal(t, "http", endpoint.Scheme)
	assert.Equal(t, "/stats/prometheus", endpoint.Path)
}

func TestCheckMandatoryParameters_WithValidConfig(t *testing.T) {
	config := buildValidMetricRelabelingConfig()

	err := checkMandatoryParameters(config)

	assert.NoError(t, err)
}

func TestCheckMandatoryParameters_MissingAction(t *testing.T) {
	config := []v1.MetricRelabelingConfig{
		{
			Action:       "",
			Regex:        ".*",
			SourceLabels: []string{"__name__"},
		},
	}

	err := checkMandatoryParameters(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mandatory field 'action'")
}

func TestCheckMandatoryParameters_MissingRegex(t *testing.T) {
	config := []v1.MetricRelabelingConfig{
		{
			Action:       "keep",
			Regex:        "",
			SourceLabels: []string{"__name__"},
		},
	}

	err := checkMandatoryParameters(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mandatory field 'regex'")
}

func TestCheckMandatoryParameters_MissingSourceLabels(t *testing.T) {
	config := []v1.MetricRelabelingConfig{
		{
			Action:       "keep",
			Regex:        ".*",
			SourceLabels: []string{},
		},
	}

	err := checkMandatoryParameters(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mandatory field 'sourceLabels'")
}

func TestGetMetricRelabeling_WithEmptyEnvVar(t *testing.T) {
	oldEnv := utils.MetricRelabeling
	defer func() { utils.MetricRelabeling = oldEnv }()

	utils.MetricRelabeling = ""

	result := getMetricRelabeling()

	assert.Nil(t, result)
}

func TestGetMetricRelabeling_WithValidConfig(t *testing.T) {
	oldEnv := utils.MetricRelabeling
	defer func() { utils.MetricRelabeling = oldEnv }()

	config := buildValidMetricRelabelingConfig()
	jsonData, _ := json.Marshal(config)
	utils.MetricRelabeling = base64.StdEncoding.EncodeToString(jsonData)

	result := getMetricRelabeling()

	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "keep", result[0].Action)
	assert.Equal(t, "metric_.*", result[0].Regex)
}

func TestGetMetricRelabeling_WithInvalidJSON(t *testing.T) {
	oldEnv := utils.MetricRelabeling
	defer func() { utils.MetricRelabeling = oldEnv }()

	utils.MetricRelabeling = base64.StdEncoding.EncodeToString([]byte("not valid json"))

	result := getMetricRelabeling()

	assert.Nil(t, result)
}

func buildPodMonitor(name, namespace, nameLabel, partOfLabel, nameSelector string) FacadePodMonitor {
	return FacadePodMonitor{
		Name:            name,
		NameSpace:       namespace,
		NameLabel:       nameLabel,
		PartOfLabel:     partOfLabel,
		NameSelector:    nameSelector,
		MasterCR:        "test-cr",
		MasterCRVersion: "v1",
		MasterCRKind:    "TestKind",
		MasterCRUID:     types.UID("test-uid-123"),
	}
}

func buildValidMetricRelabelingConfig() []v1.MetricRelabelingConfig {
	return []v1.MetricRelabelingConfig{
		{
			Action:       "keep",
			Regex:        "metric_.*",
			SourceLabels: []string{"__name__"},
		},
	}
}
