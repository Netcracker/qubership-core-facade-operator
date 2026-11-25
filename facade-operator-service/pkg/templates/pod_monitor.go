package templates

import (
	"encoding/base64"
	"encoding/json"
	v1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/monitoring/v1"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var logger = logging.GetLogger("PodMonitor")

type FacadePodMonitor struct {
	Name            string
	NameSpace       string
	NameLabel       string
	PartOfLabel     string
	NameSelector    string
	TLSEnable       bool
	MasterCR        string
	MasterCRVersion string
	MasterCRKind    string
	MasterCRUID     types.UID
}

func (f FacadePodMonitor) GetPodMonitor() *v1.PodMonitor {
	metricRelabelAction := configloader.GetOrDefaultString("metricRelabel.action", "default_keep")
	metricRelabelRegex := configloader.GetOrDefaultString("metricRelabel.regex", "default_regex")

	logger.Infof("Metric relabeling action: %s", metricRelabelAction)
	logger.Infof("Metric relabeling regex: %s", metricRelabelRegex)

	labels := map[string]string{
		"k8s-app":                                 f.NameLabel,
		"app.kubernetes.io/name":                  f.NameLabel,
		"app.kubernetes.io/component":             "monitoring",
		"app.kubernetes.io/managed-by":            "operator",
		"app.kubernetes.io/managed-by-operator":   "facade-operator",
		"app.kubernetes.io/processed-by-operator": "victoriametrics-operator",
	}
	if f.PartOfLabel != "" {
		labels["app.kubernetes.io/part-of"] = f.PartOfLabel
	} else {
		labels["app.kubernetes.io/part-of"] = utils.Unknown
	}
	controller := false
	return &v1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Name,
			Namespace: f.NameSpace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: f.MasterCRVersion,
					Kind:       f.MasterCRKind,
					Name:       f.MasterCR,
					UID:        f.MasterCRUID,
					Controller: &controller,
				},
			},
		},
		Spec: v1.PodMonitorSpec{
			PodMetricsEndpoints: f.getPodMetricsEndpoint(),
			JobLabel:            "k8s-app",
			NamespaceSelector: &v1.NamespaceSelector{
				MatchNames: []string{f.NameSpace},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": f.NameSelector,
				},
			},
		},
	}
}

func (f FacadePodMonitor) getPodMetricsEndpoint() []v1.PodMetricsEndpoint {
	podMetricEndpoint := v1.PodMetricsEndpoint{
		Interval:          "30s",
		Port:              "admin",
		Scheme:            "http",
		Path:              "/stats/prometheus",
		MetricRelabelings: getMetricRelabeling(),
	}

	// Alternative approach - hardcode metric relabeling config and check for non-empty MetricRelabeling env var
	if utils.MetricRelabeling != "" {
		podMetricEndpoint.MetricRelabelings = []v1.MetricRelabelingConfig{
			{
				Action:       "keep",
				Regex:        "envoy_.*_(ms|time|duration)_bucket;(1|10|100|1000|10000|60000)",
				SourceLabels: []string{"__name__", "le"},
			},
		}
	}

	return []v1.PodMetricsEndpoint{podMetricEndpoint}
}

func getMetricRelabeling() []v1.MetricRelabelingConfig {
	if utils.MetricRelabeling == "" {
		return nil
	}

	jsonBytes, err := base64.StdEncoding.DecodeString(utils.MetricRelabeling)
	if err != nil {
		logger.Errorf("Error during decoding env variable METRIC_RELABELING: %v", err)
		return nil
	}

	var metricRelabeling []v1.MetricRelabelingConfig
	err = json.Unmarshal(jsonBytes, &metricRelabeling)
	if err != nil {
		logger.Errorf("Error during unmarshaling env variable METRIC_RELABELING: %v", err)
		return nil
	}

	return metricRelabeling
}
