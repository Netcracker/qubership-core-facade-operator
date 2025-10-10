package templates

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildCustomIngress(t *testing.T) {
	testBuildIngress(t, "test-gw")
}

func TestBuildPublicIngress(t *testing.T) {
	testBuildIngress(t, facade.PublicGatewayService)
}

func TestBuildPrivateIngress(t *testing.T) {
	testBuildIngress(t, facade.PrivateGatewayService)
}

func testBuildIngress(t *testing.T, gwServiceName string) {
	os.Setenv("GW_INGRESS_ANNOTATIONS", "annotation.name1: 'annotation-val1'\nannotation.name2: 'annotation-val2'")
	defer os.Unsetenv("GW_INGRESS_ANNOTATIONS")

	builder := NewIngressTemplateBuilder(true, false, "")

	ingressTemplate, err := builder.BuildIngressTemplate(facade.IngressSpec{
		Hostname:    "test-host.netcracker.com",
		IsGrpc:      false,
		GatewayPort: 8080,
	}, buildFacadeService(gwServiceName), gwServiceName)
	assert.Nil(t, err)

	validateK8sIngress(t, ingressTemplate, gwServiceName, false)
	validateK8sBetaIngress(t, ingressTemplate, gwServiceName, false)
	validateOpenshiftRoute(t, ingressTemplate, gwServiceName)

	os.Setenv("INGRESS_CLASS", "test.ingress.class")
	defer os.Unsetenv("INGRESS_CLASS")

	builder = NewIngressTemplateBuilder(true, false, "")

	ingressTemplate, err = builder.BuildIngressTemplate(facade.IngressSpec{
		Hostname:    "test-host-grpc.netcracker.com",
		IsGrpc:      true,
		GatewayPort: 10050,
	}, buildFacadeService(gwServiceName), gwServiceName)
	validateK8sIngress(t, ingressTemplate, gwServiceName, true)

	os.Setenv("PEER_NAMESPACE", "test-peer-namespace")
	defer os.Unsetenv("PEER_NAMESPACE")

	builder = NewIngressTemplateBuilder(true, false, "")

	ingressTemplate, err = builder.BuildIngressTemplate(facade.IngressSpec{
		Hostname:    "test-host-grpc.netcracker.com",
		IsGrpc:      true,
		GatewayPort: 10050,
	}, buildFacadeService(gwServiceName), gwServiceName)
	assert.Nil(t, err)
	validateK8sBetaIngress(t, ingressTemplate, gwServiceName, true)
}

func buildFacadeService(gwServiceName string) *facadeV1Alpha.FacadeService {
	return &facadeV1Alpha.FacadeService{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw-srv", Namespace: "test-ns"},
		Spec: facade.FacadeServiceSpec{
			Replicas: 1,
			Gateway:  gwServiceName,
			Port:     8080,
			GatewayPorts: []facade.GatewayPorts{
				{Name: "web", Port: 8080}, {Name: "tls", Port: 8443}, {Name: "custom", Port: 10050},
			},
			GatewayType: facade.Ingress,
			Ingresses: []facade.IngressSpec{{
				Hostname:    "test-host.netcracker.com",
				IsGrpc:      false,
				GatewayPort: 8080,
			}, {
				Hostname:    "test-host-grpc.netcracker.com",
				IsGrpc:      true,
				GatewayPort: 10050,
			}},
		},
	}
}

func validateIngressName(t *testing.T, ingressName, gwServiceName string, isGrpc bool) {
	if gwServiceName == facade.PublicGatewayService {
		if isGrpc {
			assert.Equal(t, "public-gateway-grpc", ingressName)
		} else {
			assert.Equal(t, "public-gateway", ingressName)
		}
	} else if gwServiceName == facade.PrivateGatewayService {
		if isGrpc {
			assert.Equal(t, "private-gateway-grpc", ingressName)
		} else {
			assert.Equal(t, "private-gateway", ingressName)
		}
	} else {
		if isGrpc {
			assert.Equal(t, gwServiceName+"-custom-grpc", ingressName)
		} else {
			assert.Equal(t, gwServiceName+"-web", ingressName)
		}
	}
}

func validateAnnotations(t *testing.T, annotations map[string]string, gwServiceName string, isGrpc bool) {
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
	if isGrpc {
		assert.Equal(t, "true", annotations["nginx.ingress.kubernetes.io/ssl-redirect"])
		assert.Equal(t, "GRPC", annotations["nginx.ingress.kubernetes.io/backend-protocol"])
	}
}

func validateK8sIngress(t *testing.T, ingressTemplate Ingress, gwServiceName string, isGrpc bool) {
	k8sIngress := ingressTemplate.BuildK8sIngress()

	validateIngressName(t, k8sIngress.GetName(), gwServiceName, isGrpc)
	assert.Equal(t, "test-ns", k8sIngress.GetNamespace())

	validateAnnotations(t, k8sIngress.GetAnnotations(), gwServiceName, isGrpc)

	assert.Equal(t, 1, len(k8sIngress.Spec.Rules))
	rule := k8sIngress.Spec.Rules[0]
	if isGrpc {
		assert.Equal(t, "test-host-grpc.netcracker.com", rule.Host)
	} else {
		assert.Equal(t, "test-host.netcracker.com", rule.Host)
	}
	assert.Equal(t, 1, len(rule.IngressRuleValue.HTTP.Paths))
	path := rule.IngressRuleValue.HTTP.Paths[0]
	assert.Equal(t, "/", path.Path)
	assert.Equal(t, networkingv1.PathTypePrefix, *path.PathType)
	assert.Equal(t, gwServiceName, path.Backend.Service.Name)
	if os.Getenv("PEER_NAMESPACE") == "" {
		if os.Getenv("INGRESS_CLASS") == "" {
			assert.Nil(t, k8sIngress.Spec.IngressClassName)
		} else {
			assert.Equal(t, "test.ingress.class", *k8sIngress.Spec.IngressClassName)
		}
	} else {
		assert.Equal(t, facade.IngressClassName, *k8sIngress.Spec.IngressClassName)
	}
	if isGrpc {
		assert.Equal(t, int32(10050), path.Backend.Service.Port.Number)
	} else {
		assert.Equal(t, int32(8080), path.Backend.Service.Port.Number)
	}
}

func validateK8sBetaIngress(t *testing.T, ingressTemplate Ingress, gwServiceName string, isGrpc bool) {
	k8sIngress := ingressTemplate.BuildK8sBetaIngress()

	validateIngressName(t, k8sIngress.GetName(), gwServiceName, isGrpc)
	assert.Equal(t, "test-ns", k8sIngress.GetNamespace())

	validateAnnotations(t, k8sIngress.GetAnnotations(), gwServiceName, isGrpc)

	assert.Equal(t, 1, len(k8sIngress.Spec.Rules))
	rule := k8sIngress.Spec.Rules[0]
	if isGrpc {
		assert.Equal(t, "test-host-grpc.netcracker.com", rule.Host)
	} else {
		assert.Equal(t, "test-host.netcracker.com", rule.Host)
	}
	assert.Equal(t, 1, len(rule.IngressRuleValue.HTTP.Paths))
	path := rule.IngressRuleValue.HTTP.Paths[0]
	assert.Equal(t, "/", path.Path)
	assert.Equal(t, v1beta1.PathTypePrefix, *path.PathType)
	assert.Equal(t, gwServiceName, path.Backend.ServiceName)
	if os.Getenv("PEER_NAMESPACE") == "" {
		if os.Getenv("INGRESS_CLASS") == "" {
			assert.Nil(t, k8sIngress.Spec.IngressClassName)
		} else {
			assert.Equal(t, "test.ingress.class", *k8sIngress.Spec.IngressClassName)
		}
	} else {
		assert.Equal(t, facade.IngressClassName, *k8sIngress.Spec.IngressClassName)
	}
	if isGrpc {
		assert.Equal(t, int32(10050), path.Backend.ServicePort.IntVal)
	} else {
		assert.Equal(t, int32(8080), path.Backend.ServicePort.IntVal)
	}
}

func validateOpenshiftRoute(t *testing.T, ingressTemplate Ingress, gwServiceName string) {
	route := ingressTemplate.BuildOpenshiftRoute()

	if gwServiceName == facade.PublicGatewayService {
		assert.Equal(t, "public-gateway", route.GetName())
	} else if gwServiceName == facade.PrivateGatewayService {
		assert.Equal(t, "private-gateway", route.GetName())
	} else {
		assert.Equal(t, gwServiceName+"-web", route.GetName())
	}
	assert.Equal(t, "test-ns", route.GetNamespace())

	annotations := route.GetAnnotations()
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

	assert.Equal(t, "test-host.netcracker.com", route.Spec.Host)
	assert.Equal(t, "Service", route.Spec.To.Kind)
	assert.Equal(t, gwServiceName, route.Spec.To.Name)
	assert.Equal(t, intstr.Int, route.Spec.Port.TargetPort.Type)
	assert.Equal(t, int32(8080), route.Spec.Port.TargetPort.IntVal)
}
