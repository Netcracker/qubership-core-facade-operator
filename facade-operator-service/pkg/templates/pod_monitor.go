package templates

import (
	v1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/monitoring/v1"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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
		Interval: "30s",
		Port:     "admin",
		Scheme:   "http",
		Path:     "/stats/prometheus",
	}

	return []v1.PodMetricsEndpoint{podMetricEndpoint}
}
