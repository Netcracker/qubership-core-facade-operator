package templates

import (
	"os"
	"testing"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestBuildCustomHTTPRoute(t *testing.T) {
	testBuildHTTPRoute(t, "test-gw")
}

func TestBuildPublicHTTPRoute(t *testing.T) {
	testBuildHTTPRoute(t, facade.PublicGatewayService)
}

func TestBuildPrivateHTTPRoute(t *testing.T) {
	testBuildHTTPRoute(t, facade.PrivateGatewayService)
}

func testBuildHTTPRoute(t *testing.T, gwServiceName string) {
	os.Setenv("GW_INGRESS_ANNOTATIONS", "annotation.name1: 'annotation-val1'\nannotation.name2: 'annotation-val2'")
	defer os.Unsetenv("GW_INGRESS_ANNOTATIONS")

	builder := NewIngressTemplateBuilder(true, false, "")

	httpRouteTemplate, err := builder.BuildHTTPRouteTemplate(facade.IngressSpec{
		Hostname:    "test-host.qubership.org",
		IsGrpc:      false,
		GatewayPort: 8080,
	}, buildFacadeServiceForHTTPRoute(gwServiceName), gwServiceName)
	assert.Nil(t, err)

	validateK8sHTTPRoute(t, httpRouteTemplate, gwServiceName, false)
	validateHTTPRouteParentRefs(t, httpRouteTemplate, false)

	// Test GRPC HTTPRoute
	httpRouteTemplate, err = builder.BuildHTTPRouteTemplate(facade.IngressSpec{
		Hostname:    "test-host-grpc.qubership.org",
		IsGrpc:      true,
		GatewayPort: 10050,
	}, buildFacadeServiceForHTTPRoute(gwServiceName), gwServiceName)
	assert.Nil(t, err)

	validateK8sHTTPRoute(t, httpRouteTemplate, gwServiceName, true)
	validateBackendTrafficPolicy(t, httpRouteTemplate, gwServiceName, true)

	// Test x509/mTLS with ClientTrafficPolicy
	validateClientTrafficPolicy(t, httpRouteTemplate, gwServiceName, false)

	// Test BGD2 scenario with PEER_NAMESPACE
	os.Setenv("PEER_NAMESPACE", "test-peer-namespace")
	os.Setenv("CONTROLLER_NAMESPACE", "test-controller-namespace")
	defer os.Unsetenv("PEER_NAMESPACE")
	defer os.Unsetenv("CONTROLLER_NAMESPACE")

	builder = NewIngressTemplateBuilder(true, false, "")

	httpRouteTemplate, err = builder.BuildHTTPRouteTemplate(facade.IngressSpec{
		Hostname:    "test-host-grpc.qubership.org",
		IsGrpc:      true,
		GatewayPort: 10050,
	}, buildFacadeServiceForHTTPRoute(gwServiceName), gwServiceName)
	assert.Nil(t, err)
	validateHTTPRouteParentRefs(t, httpRouteTemplate, true)

	// Test satellite scenario with different x509 secret namespace
	os.Unsetenv("PEER_NAMESPACE")
	os.Unsetenv("CONTROLLER_NAMESPACE")

	builder = NewIngressTemplateBuilder(true, true, "baseline-namespace")

	httpRouteTemplate, err = builder.BuildHTTPRouteTemplate(facade.IngressSpec{
		Hostname:    "test-host.qubership.org",
		IsGrpc:      false,
		GatewayPort: 8080,
	}, buildFacadeServiceForHTTPRoute(gwServiceName), gwServiceName)
	assert.Nil(t, err)
	validateClientTrafficPolicy(t, httpRouteTemplate, gwServiceName, true)
}

func buildFacadeServiceForHTTPRoute(gwServiceName string) *facadeV1Alpha.FacadeService {
	return &facadeV1Alpha.FacadeService{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gw-srv",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "test-part-of",
			},
		},
		Spec: facade.FacadeServiceSpec{
			Replicas: 1,
			Gateway:  gwServiceName,
			Port:     8080,
			GatewayPorts: []facade.GatewayPorts{
				{Name: "web", Port: 8080}, {Name: "tls", Port: 8443}, {Name: "custom", Port: 10050},
			},
			GatewayType: facade.Ingress,
			Ingresses: []facade.IngressSpec{{
				Hostname:    "test-host.qubership.org",
				IsGrpc:      false,
				GatewayPort: 8080,
			}, {
				Hostname:    "test-host-grpc.qubership.org",
				IsGrpc:      true,
				GatewayPort: 10050,
			}},
		},
	}
}

func validateHTTPRouteName(t *testing.T, httpRouteName, gwServiceName string, isGrpc bool) {
	if gwServiceName == facade.PublicGatewayService {
		if isGrpc {
			assert.Equal(t, "public-gateway-grpc", httpRouteName)
		} else {
			assert.Equal(t, "public-gateway", httpRouteName)
		}
	} else if gwServiceName == facade.PrivateGatewayService {
		if isGrpc {
			assert.Equal(t, "private-gateway-grpc", httpRouteName)
		} else {
			assert.Equal(t, "private-gateway", httpRouteName)
		}
	} else {
		if isGrpc {
			assert.Equal(t, gwServiceName+"-custom-grpc", httpRouteName)
		} else {
			assert.Equal(t, gwServiceName+"-web", httpRouteName)
		}
	}
}

func validateHTTPRouteAnnotations(t *testing.T, annotations map[string]string, gwServiceName string) {
	assert.Equal(t, "facade-operator", annotations["app.kubernetes.io/managed-by"])
	assert.Equal(t, "1", annotations["netcracker.cloud/start.stage"])

	if gwServiceName == facade.PublicGatewayService {
		assert.Equal(t, "GENERAL", annotations["netcracker.cloud/tenant.service.tenant.id"])
		assert.Equal(t, "Public Gateway", annotations["netcracker.cloud/tenant.service.show.name"])
		assert.Equal(t, "Api Gateway to access public API", annotations["netcracker.cloud/tenant.service.show.description"])
		assert.Equal(t, "annotation-val1", annotations["annotation.name1"])
		assert.Equal(t, "annotation-val2", annotations["annotation.name2"])
	} else if gwServiceName == facade.PrivateGatewayService {
		assert.Equal(t, "annotation-val1", annotations["annotation.name1"])
		assert.Equal(t, "annotation-val2", annotations["annotation.name2"])
	} else {
		_, exists := annotations["annotation.name1"]
		assert.False(t, exists)
		_, exists = annotations["annotation.name2"]
		assert.False(t, exists)
	}

	// Note: GRPC annotations don't exist in HTTPRoute - they are handled by BackendTrafficPolicy
	_, exists := annotations["nginx.ingress.kubernetes.io/ssl-redirect"]
	assert.False(t, exists)
	_, exists = annotations["nginx.ingress.kubernetes.io/backend-protocol"]
	assert.False(t, exists)
}

func validateK8sHTTPRoute(t *testing.T, httpRouteTemplate HTTPRoute, gwServiceName string, isGrpc bool) {
	k8sHTTPRoute := httpRouteTemplate.BuildK8sHTTPRoute()

	validateHTTPRouteName(t, k8sHTTPRoute.GetName(), gwServiceName, isGrpc)
	assert.Equal(t, "test-ns", k8sHTTPRoute.GetNamespace())

	validateHTTPRouteAnnotations(t, k8sHTTPRoute.GetAnnotations(), gwServiceName)

	// Validate labels
	labels := k8sHTTPRoute.GetLabels()
	assert.Equal(t, "operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "facade-operator", labels["app.kubernetes.io/managed-by-operator"])
	assert.Equal(t, "test-part-of", labels["app.kubernetes.io/part-of"])

	// Validate hostnames
	assert.Equal(t, 1, len(k8sHTTPRoute.Spec.Hostnames))
	if isGrpc {
		assert.Equal(t, gatewayv1.Hostname("test-host-grpc.qubership.org"), k8sHTTPRoute.Spec.Hostnames[0])
	} else {
		assert.Equal(t, gatewayv1.Hostname("test-host.qubership.org"), k8sHTTPRoute.Spec.Hostnames[0])
	}

	// Validate rules
	assert.Equal(t, 1, len(k8sHTTPRoute.Spec.Rules))
	rule := k8sHTTPRoute.Spec.Rules[0]

	// Validate matches
	assert.Equal(t, 1, len(rule.Matches))
	match := rule.Matches[0]
	assert.NotNil(t, match.Path)
	assert.Equal(t, gatewayv1.PathMatchPathPrefix, *match.Path.Type)
	assert.Equal(t, "/", *match.Path.Value)

	// Validate backend refs
	assert.Equal(t, 1, len(rule.BackendRefs))
	backendRef := rule.BackendRefs[0]
	assert.Equal(t, gatewayv1.ObjectName(gwServiceName), backendRef.Name)
	assert.NotNil(t, backendRef.Kind)
	assert.Equal(t, gatewayv1.Kind("Service"), *backendRef.Kind)
	assert.NotNil(t, backendRef.Port)
	if isGrpc {
		assert.Equal(t, gatewayv1.PortNumber(10050), *backendRef.Port)
	} else {
		assert.Equal(t, gatewayv1.PortNumber(8080), *backendRef.Port)
	}
}

func validateHTTPRouteParentRefs(t *testing.T, httpRouteTemplate HTTPRoute, isBGD2 bool) {
	k8sHTTPRoute := httpRouteTemplate.BuildK8sHTTPRoute()

	assert.Equal(t, 1, len(k8sHTTPRoute.Spec.ParentRefs))
	parentRef := k8sHTTPRoute.Spec.ParentRefs[0]

	if isBGD2 {
		// BGD2 scenario: edge-router in controller namespace
		assert.Equal(t, gatewayv1.ObjectName("edge-router"), parentRef.Name)
		assert.NotNil(t, parentRef.Namespace)
		assert.Equal(t, gatewayv1.Namespace("test-controller-namespace"), *parentRef.Namespace)
	} else {
		// Default scenario: default-external-gateway in gateway-system
		assert.Equal(t, gatewayv1.ObjectName("default-external-gateway"), parentRef.Name)
		assert.NotNil(t, parentRef.Namespace)
		assert.Equal(t, gatewayv1.Namespace("gateway-system"), *parentRef.Namespace)
	}
}

func validateBackendTrafficPolicy(t *testing.T, httpRouteTemplate HTTPRoute, gwServiceName string, isGrpc bool) {
	if !isGrpc {
		assert.Nil(t, httpRouteTemplate.BackendTrafficPolicy)
		return
	}

	// GRPC should have BackendTrafficPolicy
	assert.NotNil(t, httpRouteTemplate.BackendTrafficPolicy)
	policy := httpRouteTemplate.BackendTrafficPolicy

	// Validate metadata
	if gwServiceName == facade.PublicGatewayService {
		assert.Equal(t, "public-gateway-grpc", policy.GetName())
	} else if gwServiceName == facade.PrivateGatewayService {
		assert.Equal(t, "private-gateway-grpc", policy.GetName())
	} else {
		assert.Equal(t, gwServiceName+"-custom-grpc", policy.GetName())
	}
	assert.Equal(t, "test-ns", policy.GetNamespace())
	assert.Equal(t, "BackendTrafficPolicy", policy.GetKind())
	assert.Equal(t, "gateway.envoyproxy.io/v1alpha1", policy.GetAPIVersion())

	spec, found, err := unstructured.NestedMap(policy.Object, "spec")
	assert.NoError(t, err)
	assert.True(t, found)

	targetRefs, found, err := unstructured.NestedSlice(spec, "targetRefs")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 1, len(targetRefs))

	targetRef := targetRefs[0].(map[string]interface{})
	assert.Equal(t, "gateway.networking.k8s.io", targetRef["group"])
	assert.Equal(t, "HTTPRoute", targetRef["kind"])
	if gwServiceName == facade.PublicGatewayService {
		assert.Equal(t, "public-gateway-grpc", targetRef["name"])
	} else if gwServiceName == facade.PrivateGatewayService {
		assert.Equal(t, "private-gateway-grpc", targetRef["name"])
	} else {
		assert.Equal(t, gwServiceName+"-custom-grpc", targetRef["name"])
	}

	useClientProtocol, found, err := unstructured.NestedBool(spec, "useClientProtocol")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.True(t, useClientProtocol)
}

func validateClientTrafficPolicy(t *testing.T, httpRouteTemplate HTTPRoute, gwServiceName string, isSatellite bool) {
	assert.NotNil(t, httpRouteTemplate.ClientTrafficPolicy)
	policy := httpRouteTemplate.ClientTrafficPolicy

	assert.Equal(t, "test-ns", policy.GetNamespace())
	assert.Equal(t, "ClientTrafficPolicy", policy.GetKind())
	assert.Equal(t, "gateway.envoyproxy.io/v1alpha1", policy.GetAPIVersion())

	spec, found, err := unstructured.NestedMap(policy.Object, "spec")
	assert.NoError(t, err)
	assert.True(t, found)

	targetRefs, found, err := unstructured.NestedSlice(spec, "targetRefs")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 1, len(targetRefs))

	targetRef := targetRefs[0].(map[string]interface{})
	assert.Equal(t, "gateway.networking.k8s.io", targetRef["group"])
	assert.Equal(t, "HTTPRoute", targetRef["kind"])

	tls, found, err := unstructured.NestedMap(spec, "tls")
	assert.NoError(t, err)
	assert.True(t, found)

	clientValidation, found, err := unstructured.NestedMap(tls, "clientValidation")
	assert.NoError(t, err)
	assert.True(t, found)

	optional, found, err := unstructured.NestedBool(clientValidation, "optional")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.True(t, optional)

	caCertRefs, found, err := unstructured.NestedSlice(clientValidation, "caCertificateRefs")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 1, len(caCertRefs))

	caCertRef := caCertRefs[0].(map[string]interface{})
	assert.Equal(t, "x509", caCertRef["name"])

	if isSatellite {
		assert.Equal(t, "baseline-namespace", caCertRef["namespace"])
	} else {
		assert.Equal(t, "test-ns", caCertRef["namespace"])
	}

	headers, found, err := unstructured.NestedMap(spec, "headers")
	assert.NoError(t, err)
	assert.True(t, found)

	enableEnvoyHeaders, found, err := unstructured.NestedBool(headers, "enableEnvoyHeaders")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.True(t, enableEnvoyHeaders)
}

func TestGatewaySystemEnvVariables(t *testing.T) {
	assert.Equal(t, "gateway-system", getGatewaySystemNamespace())
	assert.Equal(t, "default-external-gateway", getGatewaySystemName())

	os.Setenv("GATEWAY_SYSTEM_NAMESPACE", "custom-gateway-ns")
	os.Setenv("GATEWAY_SYSTEM_NAME", "custom-gateway-name")
	defer os.Unsetenv("GATEWAY_SYSTEM_NAMESPACE")
	defer os.Unsetenv("GATEWAY_SYSTEM_NAME")

	assert.Equal(t, "custom-gateway-ns", getGatewaySystemNamespace())
	assert.Equal(t, "custom-gateway-name", getGatewaySystemName())
}
