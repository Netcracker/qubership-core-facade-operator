package templates

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type FacadeConfigMap struct {
	Name            string
	Namespace       string
	PartOfLabel     string
	MasterCR        string
	MasterCRVersion string
	MasterCRKind    string
	MasterCRUID     types.UID
}

func (f FacadeConfigMap) GetConfigMap() *corev1.ConfigMap {
	labels := map[string]string{
		"app.kubernetes.io/managed-by":          "operator",
		"app.kubernetes.io/managed-by-operator": "facade-operator",
	}
	if f.PartOfLabel != "" {
		labels["app.kubernetes.io/part-of"] = f.PartOfLabel
	} else {
		labels["app.kubernetes.io/part-of"] = utils.Unknown
	}
	controller := false
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Name,
			Namespace: f.Namespace,
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
		Data: map[string]string{
			"prometheus.url.metrics": "http://%(ip)s:9901/stats/prometheus",
		},
	}
}
