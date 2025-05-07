package utils

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGetServiceType(t *testing.T) {
	os.Setenv("K8S_SERVICE_TYPE", "HEADLESS")
	defer os.Unsetenv("K8S_SERVICE_TYPE")
	ReloadServiceType()
	currentServiceType := GetServiceType()
	assert.Equal(t, HeadLess, currentServiceType)

	os.Setenv("K8S_SERVICE_TYPE", "CLUSTER_IP")
	ReloadServiceType()
	currentServiceType = GetServiceType()
	assert.Equal(t, ClusterIp, currentServiceType)

	os.Setenv("K8S_SERVICE_TYPE", "HEADLESS-12345")
	ReloadServiceType()
	currentServiceType = GetServiceType()
	assert.Equal(t, ClusterIp, currentServiceType)

	os.Unsetenv("K8S_SERVICE_TYPE")
	ReloadServiceType()
	currentServiceType = GetServiceType()
	assert.Equal(t, ClusterIp, currentServiceType)
}
