package main

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/lib"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/restclient"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/utils"
)

func init() {
	serviceloader.Register(1, &security.DummyToken{})
	serviceloader.Register(1, utils.NewResourceGroupAnnotationsMapper("qubership.cloud"))
	serviceloader.Register(1, restclient.NewSimpleRestClient())
}

func main() {
	lib.RunService()
}
