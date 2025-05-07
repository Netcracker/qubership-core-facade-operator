package templates

import (
	"github.com/netcracker/qubership-core-facade-operator/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type FacadeService struct {
	Name            string
	Namespace       string
	Labels          map[string]string
	NameSelector    string
	Port            int32
	GatewayPorts    []facade.GatewayPorts
	MasterCR        string
	MasterCRVersion string
	MasterCRKind    string
	MasterCRUID     types.UID
}

func (f FacadeService) GetService() *corev1.Service {
	labels := map[string]string{
		"name":                                  f.Name,
		"app.kubernetes.io/managed-by":          "operator",
		"app.kubernetes.io/managed-by-operator": "facade-operator",
	}
	if labelVal, ok := f.Labels["app.kubernetes.io/name"]; ok && f.MasterCR != "" {
		labels["app.kubernetes.io/name"] = labelVal
	} else {
		labels["app.kubernetes.io/name"] = f.Name
	}
	if labelVal, ok := f.Labels["app.kubernetes.io/part-of"]; ok {
		labels["app.kubernetes.io/part-of"] = labelVal
	} else {
		labels["app.kubernetes.io/part-of"] = utils.Unknown
	}
	controller := false
	return &corev1.Service{
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
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": f.NameSelector,
			},
			Ports:     f.getPorts(),
			ClusterIP: f.getClusterIp(),
		},
	}
}

func (f FacadeService) getClusterIp() string {
	if utils.GetServiceType() == utils.HeadLess {
		return "None"
	}

	return ""
}

func (f FacadeService) getPorts() []corev1.ServicePort {
	if f.GatewayPorts != nil && len(f.GatewayPorts) > 0 {
		var ports []corev1.ServicePort
		for _, port := range f.GatewayPorts {
			protocol := "TCP"
			if port.Protocol != "" {
				protocol = strings.ToUpper(port.Protocol)
			}
			ports = append(ports, corev1.ServicePort{
				Name:       port.Name,
				Port:       port.Port,
				Protocol:   corev1.Protocol(protocol),
				TargetPort: intstr.IntOrString{IntVal: port.Port},
			})
		}
		return ports
	}

	ports := []corev1.ServicePort{
		{
			Name:       "web",
			Port:       f.Port,
			Protocol:   "TCP",
			TargetPort: intstr.IntOrString{IntVal: 8080},
		},
	}

	return ports
}
