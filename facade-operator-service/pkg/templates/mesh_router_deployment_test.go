package templates

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadOnlyContainerEnabled(t *testing.T) {
	os.Setenv("LOG_LEVEL", "info")
	configloader.Init(configloader.EnvPropertySource())
	facadeDeployment := &RouterDeployment{
		ReadOnlyContainerEnabled: true,
	}

	deployment := facadeDeployment.GetDeployment()
	securityContext := deployment.Spec.Template.Spec.Containers[0].SecurityContext.ReadOnlyRootFilesystem
	assert.True(t, *securityContext)

	mounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts
	assert.NotNil(t, mounts)
	assert.Equal(t, "config", mounts[0].Name)
	assert.Equal(t, "/envoy/config", mounts[0].MountPath)

	volumes := deployment.Spec.Template.Spec.Volumes
	assert.NotNil(t, volumes)
	assert.Equal(t, "config", volumes[0].Name)
	assert.NotNil(t, volumes[0].VolumeSource.EmptyDir)
}

func TestDeploymentPorts_hostedByLabel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "info")
	configloader.Init(configloader.EnvPropertySource())
	facadeDeployment := &RouterDeployment{
		HostedBy: "value",
	}

	kubDeployment := facadeDeployment.GetDeployment()
	labelValue := kubDeployment.Labels[utils.HostedByLabel]
	assert.Equal(t, "", labelValue)

	facadeDeployment = &RouterDeployment{
		MasterCR: "masterCr",
		HostedBy: "value",
	}

	kubDeployment = facadeDeployment.GetDeployment()
	labelValue = kubDeployment.Labels[utils.HostedByLabel]
	assert.Equal(t, "value", labelValue)
}

func TestDeploymentPorts_shouldReturnDefaultPort_whenGatewayPortsNotFilled(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	configloader.Init(configloader.EnvPropertySource())
	facadeDeployment := &RouterDeployment{
		GatewayPorts: nil,
	}

	kubDeployment := facadeDeployment.GetDeployment()
	actualPorts := kubDeployment.Spec.Template.Spec.Containers[0].Ports
	assert.Equal(t, 2, len(actualPorts))
	assert.Equal(t, int32(9901), actualPorts[0].ContainerPort)
	assert.Equal(t, int32(8080), actualPorts[1].ContainerPort)
}

func TestDeploymentPorts_shouldReturnCorrectPorts_whenGatewayPortsFilled(t *testing.T) {
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

	facadeDeployment := &RouterDeployment{
		GatewayPorts: gatewayPorts,
	}

	kubDeployment := facadeDeployment.GetDeployment()
	actualPorts := kubDeployment.Spec.Template.Spec.Containers[0].Ports
	assert.Equal(t, 3, len(actualPorts))

	assert.Equal(t, int32(9901), actualPorts[0].ContainerPort)
	assert.Equal(t, gatewayPorts[0].Port, actualPorts[1].ContainerPort)
	assert.Equal(t, gatewayPorts[1].Port, actualPorts[2].ContainerPort)
}

func TestDeploymentPorts_shouldNotDuplicateAdminPort(t *testing.T) {
	gatewayPorts := []facade.GatewayPorts{
		{
			Name:     "web1",
			Port:     int32(9901),
			Protocol: "TCP",
		},
		{
			Name: "web1",
			Port: int32(4321),
		},
	}

	facadeDeployment := &RouterDeployment{
		GatewayPorts: gatewayPorts,
	}

	kubDeployment := facadeDeployment.GetDeployment()
	actualPorts := kubDeployment.Spec.Template.Spec.Containers[0].Ports
	assert.Equal(t, 2, len(actualPorts))

	assert.Equal(t, int32(9901), actualPorts[0].ContainerPort)
	assert.Equal(t, gatewayPorts[1].Port, actualPorts[1].ContainerPort)
}

func TestDeploymentDefaultLabels(t *testing.T) {
	fsLabelName := "name"
	facadeDeployment := &RouterDeployment{
		GatewayName: fsLabelName,
	}

	kubDeployment := facadeDeployment.GetDeployment()
	actualLabels := kubDeployment.ObjectMeta.Labels
	assert.Equal(t, actualLabels["name"], fsLabelName)
	assert.Equal(t, actualLabels["app.kubernetes.io/name"], fsLabelName)
}

func TestDeploymentCustomLabels(t *testing.T) {
	fsLabelName := "name"
	fsLabelKubIOName := "test-name"
	fsLabelPartOf := "test-cloud-core"
	facadeDeployment := &RouterDeployment{
		GatewayName: fsLabelName,
		CrLabels: map[string]string{
			"app.kubernetes.io/name":    fsLabelKubIOName,
			"app.kubernetes.io/part-of": fsLabelPartOf,
		},
		MasterCR: "masterCR",
	}

	kubDeployment := facadeDeployment.GetDeployment()
	actualLabels := kubDeployment.ObjectMeta.Labels
	assert.Equal(t, actualLabels["name"], fsLabelName)
	assert.Equal(t, actualLabels["app.kubernetes.io/name"], fsLabelKubIOName)
	assert.Equal(t, actualLabels["app.kubernetes.io/part-of"], fsLabelPartOf)
}
