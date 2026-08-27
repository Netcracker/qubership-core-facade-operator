package templates

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
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

// GetSharedService returns the service shaped exactly as the helm chart declares it:
// selector driven by SERVICE_MESH_TYPE, ClusterIP service type regardless of K8S_SERVICE_TYPE,
// and no ownerReferences — the chart owns it permanently.
func (f FacadeService) GetSharedService() *corev1.Service {
	selector := map[string]string{
		"app": f.NameSelector,
	}
	if utils.IsIstioMeshType() {
		// f.Name doubles as the istio Gateway name: the chart keeps ISTIO_EGRESS_GATEWAY_NAME
		// and EGRESS_GATEWAY_SERVICE_NAME both equal to "egress-gateway". If they ever diverge
		// the gateway name needs its own env var.
		selector = map[string]string{
			"gateway.networking.k8s.io/gateway-name": f.Name,
		}
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Name,
			Namespace: f.Namespace,
			Labels:    f.getLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			Ports:     f.getPorts(),
			ClusterIP: "",
		},
	}
}

func (f FacadeService) getLabels() map[string]string {
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
	return labels
}

func (f FacadeService) GetService() *corev1.Service {
	controller := false
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Name,
			Namespace: f.Namespace,
			Labels:    f.getLabels(),
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
