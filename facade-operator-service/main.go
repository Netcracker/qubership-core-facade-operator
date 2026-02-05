package main

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/lib"
	fiberSec "github.com/netcracker/qubership-core-lib-go-fiber-server-utils/v2/security"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

func init() {
	serviceloader.Register(1, &security.DummyToken{})
	serviceloader.Register(1, &fiberSec.DummyFiberServerSecurityMiddleware{})

	// uses default values:
	//   WithRatio(0.9)
	//   WithProvider(memlimit.FromCgroup)
	// and no logger
	memlimit, _ := memlimit.SetGoMemLimitWithOpts()

	logger := logging.GetLogger("main")
	if memlimit > 0 {
		logger.Info("GOMEMLIMIT set to %d bytes (0.9 of cgroup's memory limit)", memlimit)
	} else {
		logger.Info("GOMEMLIMIT not set, using default value")
	}
}

func main() {
	lib.RunService()
}
