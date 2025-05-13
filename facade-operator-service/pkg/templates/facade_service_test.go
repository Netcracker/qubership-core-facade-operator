package templates

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"testing"

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
