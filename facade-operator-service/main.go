package main

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/lib"
	fiberSec "github.com/netcracker/qubership-core-lib-go-fiber-server-utils/v2/security"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"

	// memlimit sets memory limit = 0.9 of cgroup memory limit
	_ "github.com/netcracker/qubership-core-lib-go/v3/memlimit"
)

func init() {
	serviceloader.Register(1, &security.DummyToken{})
	serviceloader.Register(1, &fiberSec.DummyFiberServerSecurityMiddleware{})
}

func main() {
	lib.RunService()
}
