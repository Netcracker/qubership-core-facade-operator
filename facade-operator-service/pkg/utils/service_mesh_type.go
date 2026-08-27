package utils

import (
	"os"
	"strings"
)

var serviceMeshType string

func init() {
	ReloadServiceMeshType()
}

func IsIstioMeshType() bool {
	return strings.EqualFold(serviceMeshType, "Istio")
}

func ReloadServiceMeshType() {
	serviceMeshType = os.Getenv("SERVICE_MESH_TYPE")
}
