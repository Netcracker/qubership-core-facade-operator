package templates

import (
	"os"
	"testing"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestServicePorts_shouldReturnOnePort_whenOnlyPortFilled(t *testing.T) {
	port := int32(8080)

	facadeService := &FacadeService{
		Name:         "name",
		Namespace:    "namespace",
		NameSelector: "gatewayName",
		Port:         port,
		GatewayPorts: nil,
	}

	kubService := facadeService.GetService()
	assert.Equal(t, 1, len(kubService.Spec.Ports))
	assert.Equal(t, port, kubService.Spec.Ports[0].Port)
}

func TestServicePorts_shouldReturnTwoPorts_whenGatewayPortsFilled(t *testing.T) {
	gatewayPorts := []facade.GatewayPorts{
		{
			Name:     "web1",
			Port:     int32(1234),
			Protocol: "TCP",
		},
		{
			Name: "web1",
			Port: int32(4321),
		},
	}

	facadeService := &FacadeService{
		Name:         "name",
		Namespace:    "namespace",
		NameSelector: "gatewayName",
		Port:         8080,
		GatewayPorts: gatewayPorts,
	}

	kubService := facadeService.GetService()
	actualPorts := kubService.Spec.Ports
	assert.Equal(t, 2, len(actualPorts))

	assert.Equal(t, gatewayPorts[0].Name, actualPorts[0].Name)
	assert.Equal(t, gatewayPorts[0].Port, actualPorts[0].Port)
	assert.Equal(t, corev1.Protocol(gatewayPorts[0].Protocol), actualPorts[0].Protocol)

	assert.Equal(t, gatewayPorts[1].Name, actualPorts[1].Name)
	assert.Equal(t, gatewayPorts[1].Port, actualPorts[1].Port)
	assert.Equal(t, corev1.Protocol("TCP"), actualPorts[1].Protocol)
}

func TestServiceDefaultLabels(t *testing.T) {
	fsName := "name"
	facadeService := &FacadeService{
		Name:         fsName,
		Namespace:    "namespace",
		NameSelector: "gatewayName",
		Port:         1234,
		GatewayPorts: nil,
	}

	kubService := facadeService.GetService()
	assert.Equal(t, kubService.Labels["app.kubernetes.io/name"], fsName)
	assert.Equal(t, kubService.Labels["app.kubernetes.io/part-of"], utils.Unknown)
}

func TestGetSharedService_selectorIsApp_whenCoreMeshType(t *testing.T) {
	defer func() {
		os.Unsetenv("SERVICE_MESH_TYPE")
		utils.ReloadServiceMeshType()
	}()
	os.Unsetenv("SERVICE_MESH_TYPE")
	utils.ReloadServiceMeshType()

	fs := &FacadeService{
		Name:         "egress-gateway",
		Namespace:    "test-ns",
		NameSelector: "egress-gateway-gateway",
		Port:         8080,
	}
	svc := fs.GetSharedService()
	assert.Equal(t, map[string]string{"app": "egress-gateway-gateway"}, svc.Spec.Selector)
	assert.Nil(t, svc.OwnerReferences)
	assert.Empty(t, svc.Spec.ClusterIP)
}

func TestGetSharedService_selectorIsGatewayName_whenIstioMeshType(t *testing.T) {
	defer func() {
		os.Unsetenv("SERVICE_MESH_TYPE")
		utils.ReloadServiceMeshType()
	}()
	os.Setenv("SERVICE_MESH_TYPE", "Istio")
	utils.ReloadServiceMeshType()

	fs := &FacadeService{
		Name:         "egress-gateway",
		Namespace:    "test-ns",
		NameSelector: "egress-gateway-gateway",
		Port:         8080,
	}
	svc := fs.GetSharedService()
	assert.Equal(t, map[string]string{"gateway.networking.k8s.io/gateway-name": "egress-gateway"}, svc.Spec.Selector)
	assert.Nil(t, svc.OwnerReferences)
	assert.Empty(t, svc.Spec.ClusterIP)
}

func TestGetSharedService_clusterIPEmptyEvenWhenHeadless(t *testing.T) {
	defer func() {
		os.Unsetenv("K8S_SERVICE_TYPE")
		os.Unsetenv("SERVICE_MESH_TYPE")
		utils.ReloadServiceType()
		utils.ReloadServiceMeshType()
	}()
	os.Setenv("K8S_SERVICE_TYPE", "HEADLESS")
	os.Unsetenv("SERVICE_MESH_TYPE")
	utils.ReloadServiceType()
	utils.ReloadServiceMeshType()

	fs := &FacadeService{
		Name:         "egress-gateway",
		Namespace:    "test-ns",
		NameSelector: "egress-gateway-gateway",
		Port:         8080,
	}
	shared := fs.GetSharedService()
	assert.Empty(t, shared.Spec.ClusterIP)

	// GetService still returns "None" for HEADLESS
	regular := fs.GetService()
	assert.Equal(t, "None", regular.Spec.ClusterIP)
}

func TestGetSharedService_noOwnerReferences(t *testing.T) {
	fs := &FacadeService{
		Name:            "egress-gateway",
		Namespace:       "test-ns",
		NameSelector:    "egress-gateway-gateway",
		Port:            8080,
		MasterCR:        "core-egress-gateway",
		MasterCRVersion: "netcracker.com/v1alpha",
		MasterCRKind:    "FacadeService",
	}
	svc := fs.GetSharedService()
	assert.Nil(t, svc.OwnerReferences)
}

func TestGetSharedService_gatewayPortsHonoured(t *testing.T) {
	fs := &FacadeService{
		Name:         "egress-gateway",
		Namespace:    "test-ns",
		NameSelector: "egress-gateway-gateway",
		Port:         8080,
		GatewayPorts: []facade.GatewayPorts{
			{Name: "http", Port: 8080, Protocol: "TCP"},
			{Name: "grpc", Port: 9090, Protocol: "TCP"},
		},
	}
	svc := fs.GetSharedService()
	assert.Equal(t, 2, len(svc.Spec.Ports))
	assert.Equal(t, int32(8080), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(9090), svc.Spec.Ports[1].Port)
}

func TestServiceCustomLabels(t *testing.T) {
	fsLabelName := "test-name"
	fsLabelPartOf := "test-cloud-core"
	facadeService := &FacadeService{
		Name:      "name",
		Namespace: "namespace",
		Labels: map[string]string{
			"app.kubernetes.io/name":    fsLabelName,
			"app.kubernetes.io/part-of": fsLabelPartOf,
		},
		NameSelector: "gatewayName",
		Port:         1234,
		GatewayPorts: nil,
		MasterCR:     "masterCR",
	}

	kubService := facadeService.GetService()
	assert.Equal(t, kubService.Labels["app.kubernetes.io/name"], fsLabelName)
	assert.Equal(t, kubService.Labels["app.kubernetes.io/part-of"], fsLabelPartOf)
}
