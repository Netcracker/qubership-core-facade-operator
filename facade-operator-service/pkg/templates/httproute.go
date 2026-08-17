package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var httpRouteLogger = logging.GetLogger("templates/httproute")

const (
	defaultGatewaySystemNamespace = "gateway-system"
	defaultGatewaySystemName      = "default-external-gateway"
	edgeRouterName                = "edge-router"

	envHTTPRouteIdleTimeout   = "HTTP_ROUTE_REQUEST_IDLE_TIMEOUT"
	envHTTPRouteCustomFilters = "HTTP_ROUTE_CUSTOM_FILTERS"

	annotationProxyReadTimeout = "nginx.ingress.kubernetes.io/proxy-read-timeout"
	annotationProxySendTimeout = "nginx.ingress.kubernetes.io/proxy-send-timeout"
)

// gatewayAPIDuration mirrors the validation pattern of gateway-api v1.Duration (GEP-2257),
// which is the format accepted by Envoy Gateway timeout fields.
var gatewayAPIDuration = regexp.MustCompile(`^([0-9]{1,5}(h|m|s|ms)){1,4}$`)

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
	CustomFilters         []gatewayv1.HTTPRouteFilter
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
		Labels:              b.buildIngressLabels(cr.GetLabels()[utils.KubernetesPartOf]),
		Annotations:         b.buildHTTPRouteAnnotations(gatewayServiceName),
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
		CustomFilters:       b.httpRouteCustomFilters,
		X509SecretNamespace: x509SecretNamespace,
	}

	httpRoute.BackendTrafficPolicy, err = b.buildBackendTrafficPolicy(httpRouteName, cr, ingressSpec.IsGrpc)
	if err != nil {
		return HTTPRoute{}, err
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
	kindGateway := gatewayv1.Kind("Gateway")
	groupGateway := gatewayv1.Group(gatewayv1.GroupName)
	groupCore := gatewayv1.Group("")
	namespace := gatewayv1.Namespace(h.ParentNamespace)
	parentName := gatewayv1.ObjectName(h.ParentName)
	weight := int32(1)

	rule := gatewayv1.HTTPRouteRule{
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
						Group: &groupCore,
						Name:  gatewayv1.ObjectName(h.ServiceName),
						Kind:  &kindService,
						Port:  getPortPointer(gatewayv1.PortNumber(h.Port)),
					},
					Weight: &weight,
				},
			},
		},
	}
	if len(h.CustomFilters) > 0 {
		rule.Filters = h.CustomFilters
	}

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
						Group:     &groupGateway,
						Kind:      &kindGateway,
						Name:      parentName,
						Namespace: &namespace,
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{hostname},
			Rules:     []gatewayv1.HTTPRouteRule{rule},
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

func buildHTTPRouteCustomFilters() []gatewayv1.HTTPRouteFilter {
	envVal := strings.TrimSpace(os.Getenv(envHTTPRouteCustomFilters))
	if envVal == "" || envVal == "[]" || envVal == "null" {
		return nil
	}
	var filters []gatewayv1.HTTPRouteFilter
	if err := json.Unmarshal([]byte(envVal), &filters); err != nil {
		httpRouteLogger.Errorf("Failed to unmarshal %s: %v", envHTTPRouteCustomFilters, err)
		return nil
	}
	if len(filters) == 0 {
		return nil
	}

	validFilters := make([]gatewayv1.HTTPRouteFilter, 0, len(filters))
	for _, filter := range filters {
		if err := validateHTTPRouteFilter(filter); err != nil {
			httpRouteLogger.Errorf("Skipping invalid HTTPRoute filter from %s: %v", envHTTPRouteCustomFilters, err)
			continue
		}
		validFilters = append(validFilters, filter)
	}
	if len(validFilters) == 0 {
		return nil
	}
	return validFilters
}

var httpRouteFilterValidators = map[gatewayv1.HTTPRouteFilterType]struct {
	field   string
	present func(gatewayv1.HTTPRouteFilter) bool
}{
	gatewayv1.HTTPRouteFilterRequestHeaderModifier:  {"requestHeaderModifier", func(f gatewayv1.HTTPRouteFilter) bool { return f.RequestHeaderModifier != nil }},
	gatewayv1.HTTPRouteFilterResponseHeaderModifier: {"responseHeaderModifier", func(f gatewayv1.HTTPRouteFilter) bool { return f.ResponseHeaderModifier != nil }},
	gatewayv1.HTTPRouteFilterRequestRedirect:        {"requestRedirect", func(f gatewayv1.HTTPRouteFilter) bool { return f.RequestRedirect != nil }},
	gatewayv1.HTTPRouteFilterURLRewrite:             {"urlRewrite", func(f gatewayv1.HTTPRouteFilter) bool { return f.URLRewrite != nil }},
	gatewayv1.HTTPRouteFilterRequestMirror:          {"requestMirror", func(f gatewayv1.HTTPRouteFilter) bool { return f.RequestMirror != nil }},
	gatewayv1.HTTPRouteFilterCORS:                   {"cors", func(f gatewayv1.HTTPRouteFilter) bool { return f.CORS != nil }},
	gatewayv1.HTTPRouteFilterExternalAuth:           {"externalAuth", func(f gatewayv1.HTTPRouteFilter) bool { return f.ExternalAuth != nil }},
	gatewayv1.HTTPRouteFilterExtensionRef:           {"extensionRef", func(f gatewayv1.HTTPRouteFilter) bool { return f.ExtensionRef != nil }},
}

func validateHTTPRouteFilter(filter gatewayv1.HTTPRouteFilter) error {
	validator, ok := httpRouteFilterValidators[filter.Type]
	if !ok {
		return fmt.Errorf("unsupported filter type %q", filter.Type)
	}
	if !validator.present(filter) {
		return fmt.Errorf("type %q requires %s", filter.Type, validator.field)
	}
	return nil
}

func (b *IngressTemplateBuilder) resolveStreamIdleTimeout() (string, error) {
	idleTimeout := strings.TrimSpace(os.Getenv(envHTTPRouteIdleTimeout))
	if idleTimeout != "" {
		if !gatewayAPIDuration.MatchString(idleTimeout) {
			return "", fmt.Errorf("%s value %q is not a valid Gateway API duration, expected pattern %s, for example 1800s",
				envHTTPRouteIdleTimeout, idleTimeout, gatewayAPIDuration.String())
		}
		return idleTimeout, nil
	}

	return deriveIdleTimeoutFromLegacyAnnotations(b.gwIngressAnnotations)
}

func deriveIdleTimeoutFromLegacyAnnotations(annotations map[string]string) (string, error) {
	if len(annotations) == 0 {
		return "", nil
	}
	maxSec := -1
	for _, annotation := range []string{annotationProxyReadTimeout, annotationProxySendTimeout} {
		value := strings.TrimSpace(annotations[annotation])
		if value == "" {
			continue
		}
		sec, err := strconv.Atoi(value)
		if err != nil {
			httpRouteLogger.Warnf("Ignoring annotation %s value %q: expected a number of seconds", annotation, value)
			continue
		}
		if sec > maxSec {
			maxSec = sec
		}
	}
	if maxSec < 0 {
		return "", nil
	}

	idleTimeout := strconv.Itoa(maxSec) + "s"
	if !gatewayAPIDuration.MatchString(idleTimeout) {
		return "", fmt.Errorf("timeout %s derived from legacy annotations is not a valid Gateway API duration", idleTimeout)
	}
	return idleTimeout, nil
}

func (b *IngressTemplateBuilder) buildBackendTrafficPolicy(httpRouteName string, cr facade.MeshGateway, isGrpc bool) (*unstructured.Unstructured, error) {
	streamIdleTimeout, err := b.resolveStreamIdleTimeout()
	if err != nil {
		httpRouteLogger.Errorf("Failed to resolve stream idle timeout for HTTPRoute %s: %v", httpRouteName, err)
		return nil, err
	}
	if !isGrpc && streamIdleTimeout == "" {
		return nil, nil
	}

	spec := map[string]interface{}{
		"mergeType": "StrategicMerge",
		"targetRefs": []interface{}{
			map[string]interface{}{
				"group": "gateway.networking.k8s.io",
				"kind":  "HTTPRoute",
				"name":  httpRouteName,
			},
		},
	}
	if isGrpc {
		spec["useClientProtocol"] = true
	}
	if streamIdleTimeout != "" {
		spec["timeout"] = map[string]interface{}{
			"http": map[string]interface{}{
				"streamIdleTimeout": streamIdleTimeout,
			},
		}
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": utils.ApiVersionV1AlphaV1,
			"kind":       "BackendTrafficPolicy",
			"metadata": map[string]interface{}{
				"name":      httpRouteName,
				"namespace": cr.GetNamespace(),
				"labels":    b.buildIngressLabels(cr.GetLabels()[utils.KubernetesPartOf]),
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
			"spec": spec,
		},
	}, nil
}

func (b *IngressTemplateBuilder) buildClientTrafficPolicy(httpRouteName string, cr facade.MeshGateway, x509SecretNamespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": utils.ApiVersionV1AlphaV1,
			"kind":       "ClientTrafficPolicy",
			"metadata": map[string]interface{}{
				"name":      httpRouteName,
				"namespace": cr.GetNamespace(),
				"labels":    b.buildIngressLabels(cr.GetLabels()[utils.KubernetesPartOf]),
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
}
