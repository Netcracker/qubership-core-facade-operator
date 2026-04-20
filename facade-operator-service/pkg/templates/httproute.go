package templates

import (
	"maps"
	"os"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	defaultGatewaySystemNamespace = "gateway-system"
	defaultGatewaySystemName      = "default-external-gateway"
	edgeRouterName                = "edge-router"
)

type HTTPRoute struct {
	Name                  string
	Namespace             string
	Labels                map[string]string
	Annotations           map[string]string
	Hostname              string
	ServiceName           string
	Port                  int32
	IsGrpc                bool
	ParentName            string
	ParentNamespace       string
	MasterCR              string
	MasterCRVersion       string
	MasterCRKind          string
	MasterCRUID           types.UID
	BackendTrafficPolicy  *unstructured.Unstructured
	ClientTrafficPolicy   *unstructured.Unstructured
	NeedsBackendTLSPolicy bool
	X509SecretNamespace   string
}

func (b *IngressTemplateBuilder) BuildHTTPRouteTemplate(ingressSpec facade.IngressSpec, cr facade.MeshGateway, gatewayServiceName string) (HTTPRoute, error) {
	httpRouteName, gwPort, err := b.BuildNameAndPort(ingressSpec, cr, gatewayServiceName)
	if err != nil {
		return HTTPRoute{}, err
	}

	x509SecretNamespace := cr.GetNamespace()
	if b.isSatellite {
		x509SecretNamespace = b.baselineNamespace
	}

	httpRoute := HTTPRoute{
		Name:                httpRouteName,
		Namespace:           cr.GetNamespace(),
		Labels:              b.buildIngressLabels(cr.GetLabels()["app.kubernetes.io/part-of"]),
		Annotations:         b.buildHTTPRouteAnnotations(gatewayServiceName, cr.GetNamespace(), ingressSpec.IsGrpc),
		Hostname:            ingressSpec.Hostname,
		ServiceName:         gatewayServiceName,
		Port:                gwPort,
		IsGrpc:              ingressSpec.IsGrpc,
		ParentName:          b.getHTTPRouteParentName(),
		ParentNamespace:     b.getHTTPRouteParentNamespace(),
		MasterCR:            cr.GetName(),
		MasterCRVersion:     cr.GetAPIVersion(),
		MasterCRKind:        cr.GetKind(),
		MasterCRUID:         cr.GetUID(),
		X509SecretNamespace: x509SecretNamespace,
	}

	if ingressSpec.IsGrpc {
		httpRoute.BackendTrafficPolicy = b.buildBackendTrafficPolicy(httpRouteName, cr)
	}

	if b.x509Enable {
		httpRoute.ClientTrafficPolicy = b.buildClientTrafficPolicy(httpRouteName, cr, x509SecretNamespace)
	}

	return httpRoute, nil
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

func (b *IngressTemplateBuilder) buildHTTPRouteAnnotations(gatewayServiceName, namespace string, isGrpc bool) map[string]string {
	annotations := make(map[string]string)
	annotations["app.kubernetes.io/managed-by"] = "facade-operator"
	annotations["netcracker.cloud/start.stage"] = "1"

	if gatewayServiceName == facade.PublicGatewayService {
		annotations["netcracker.cloud/tenant.service.tenant.id"] = "GENERAL"
		annotations["netcracker.cloud/tenant.service.show.name"] = "Public Gateway"
		annotations["netcracker.cloud/tenant.service.show.description"] = "Api Gateway to access public API"
		maps.Copy(annotations, b.gwIngressAnnotations)
	} else if gatewayServiceName == facade.PrivateGatewayService {
		maps.Copy(annotations, b.gwIngressAnnotations)
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

func (b *IngressTemplateBuilder) buildBackendTrafficPolicy(httpRouteName string, cr facade.MeshGateway) *unstructured.Unstructured {
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.envoyproxy.io/v1alpha1",
			"kind":       "BackendTrafficPolicy",
			"metadata": map[string]interface{}{
				"name":      httpRouteName,
				"namespace": cr.GetNamespace(),
				"labels":    b.buildIngressLabels(cr.GetLabels()["app.kubernetes.io/part-of"]),
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"apiVersion": cr.GetAPIVersion(),
						"kind":       cr.GetKind(),
						"name":       cr.GetName(),
						"uid":        string(cr.GetUID()),
						"controller": false,
					},
				},
			},
			"spec": map[string]interface{}{
				"targetRefs": []interface{}{
					map[string]interface{}{
						"group": "gateway.networking.k8s.io",
						"kind":  "HTTPRoute",
						"name":  httpRouteName,
					},
				},
				"useClientProtocol": true,
			},
		},
	}

	return policy
}

func (b *IngressTemplateBuilder) buildClientTrafficPolicy(httpRouteName string, cr facade.MeshGateway, x509SecretNamespace string) *unstructured.Unstructured {
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.envoyproxy.io/v1alpha1",
			"kind":       "ClientTrafficPolicy",
			"metadata": map[string]interface{}{
				"name":      httpRouteName,
				"namespace": cr.GetNamespace(),
				"labels":    b.buildIngressLabels(cr.GetLabels()["app.kubernetes.io/part-of"]),
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"apiVersion": cr.GetAPIVersion(),
						"kind":       cr.GetKind(),
						"name":       cr.GetName(),
						"uid":        string(cr.GetUID()),
						"controller": false,
					},
				},
			},
			"spec": map[string]interface{}{
				"targetRefs": []interface{}{
					map[string]interface{}{
						"group": "gateway.networking.k8s.io",
						"kind":  "HTTPRoute",
						"name":  httpRouteName,
					},
				},
				"tls": map[string]interface{}{
					"clientValidation": map[string]interface{}{
						// optional_no_ca - analog of nginx.ingress.kubernetes.io/auth-tls-verify-client: optional_no_ca
						"optional": true,
						"caCertificateRefs": []interface{}{
							map[string]interface{}{
								"group":     "",
								"kind":      "Secret",
								"name":      "x509",
								"namespace": x509SecretNamespace,
							},
						},
					},
				},
				"headers": map[string]interface{}{
					"enableEnvoyHeaders": true,
				},
			},
		},
	}

	return policy
}
