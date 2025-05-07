package utils

import (
	"os"
	"strings"
)

type ServiceType string

var (
	serviceType ServiceType
)

const ClusterIp ServiceType = "CLUSTER_IP"
const HeadLess ServiceType = "HEADLESS"

func init() {
	ReloadServiceType()
}

func GetServiceType() ServiceType {
	return serviceType
}

func ReloadServiceType() {
	env := os.Getenv("K8S_SERVICE_TYPE")
	if strings.EqualFold(env, "CLUSTER_IP") || !strings.EqualFold(env, "HEADLESS") {
		serviceType = ClusterIp
	} else {
		serviceType = HeadLess
	}
}
