package templates

import (
	"os"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	defaultGatewaySystemNamespace = "gateway-system"
	defaultGatewaySystemName      = "default-external-gateway"
	edgeRouterName                = "edge-router"
)

type HTTPRoute struct {
	Name            string
	Namespace       string
	Labels          map[string]string
	Annotations     map[string]string
	Hostname        string
	ServiceName     string
	Port            int32
	ParentName      string
	ParentNamespace string
	MasterCR        string
	MasterCRVersion string
	MasterCRKind    string
	MasterCRUID     types.UID
}

func (b *IngressTemplateBuilder) BuildHTTPRouteTemplate(ingressSpec facade.IngressSpec, cr facade.MeshGateway, gatewayServiceName string) (HTTPRoute, error) {
	httpRouteName, gwPort, err := b.BuildNameAndPort(ingressSpec, cr, gatewayServiceName)
	if err != nil {
		return HTTPRoute{}, err
	}

	return HTTPRoute{
		Name:            httpRouteName,
		Namespace:       cr.GetNamespace(),
		Labels:          b.buildIngressLabels(cr.GetLabels()["app.kubernetes.io/part-of"]),
		Annotations:     b.buildHTTPRouteAnnotations(gatewayServiceName),
		Hostname:        ingressSpec.Hostname,
		ServiceName:     gatewayServiceName,
		Port:            gwPort,
		ParentName:      b.getHTTPRouteParentName(),
		ParentNamespace: b.getHTTPRouteParentNamespace(),
		MasterCR:        cr.GetName(),
		MasterCRVersion: cr.GetAPIVersion(),
		MasterCRKind:    cr.GetKind(),
		MasterCRUID:     cr.GetUID(),
	}, nil
}

func (h HTTPRoute) BuildK8sHTTPRoute() *gatewayv1.HTTPRoute {
	controller := false
	pathPrefix := gatewayv1.PathMatchPathPrefix
	hostname := gatewayv1.Hostname(h.Hostname)
	kindService := gatewayv1.Kind("Service")
	namespace := gatewayv1.Namespace(h.ParentNamespace)
	parentName := gatewayv1.ObjectName(h.ParentName)

	return &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: "gateway.networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        h.Name,
			Namespace:   h.Namespace,
			Annotations: h.Annotations,
			Labels:      h.Labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: h.MasterCRVersion,
					Kind:       h.MasterCRKind,
					Name:       h.MasterCR,
					UID:        h.MasterCRUID,
					Controller: &controller,
				},
			},
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:      parentName,
						Namespace: &namespace,
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{hostname},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathPrefix,
								Value: getPathPointer("/"),
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(h.ServiceName),
									Kind: &kindService,
									Port: getPortPointer(gatewayv1.PortNumber(h.Port)),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (b *IngressTemplateBuilder) buildHTTPRouteAnnotations(gatewayServiceName string) map[string]string {
	annotations := make(map[string]string)
	annotations["app.kubernetes.io/managed-by"] = "facade-operator"
	annotations["netcracker.cloud/start.stage"] = "1"

	if gatewayServiceName == facade.PublicGatewayService {
		annotations["netcracker.cloud/tenant.service.tenant.id"] = "GENERAL"
		annotations["netcracker.cloud/tenant.service.show.name"] = "Public Gateway"
		annotations["netcracker.cloud/tenant.service.show.description"] = "Api Gateway to access public API"
	}

	return annotations
}

func getGatewaySystemNamespace() string {
	namespace := os.Getenv("GATEWAY_SYSTEM_NAMESPACE")
	if namespace == "" {
		return defaultGatewaySystemNamespace
	}
	return namespace
}

func getGatewaySystemName() string {
	name := os.Getenv("GATEWAY_SYSTEM_NAME")
	if name == "" {
		return defaultGatewaySystemName
	}
	return name
}

func (b *IngressTemplateBuilder) getHTTPRouteParentName() string {
	if os.Getenv("PEER_NAMESPACE") != "" {
		return edgeRouterName
	}
	return getGatewaySystemName()
}

func (b *IngressTemplateBuilder) getHTTPRouteParentNamespace() string {
	if os.Getenv("PEER_NAMESPACE") != "" {
		controllerNamespace := os.Getenv("CONTROLLER_NAMESPACE")
		if controllerNamespace != "" {
			return controllerNamespace
		}
	}
	return getGatewaySystemNamespace()
}

func getPortPointer(port gatewayv1.PortNumber) *gatewayv1.PortNumber {
	return &port
}

func getPathPointer(path string) *string {
	return &path
}
