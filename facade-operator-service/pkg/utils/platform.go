package utils

import (
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"os"
	"strings"
)

type Mode string

var (
	mode        Mode
	paasVersion SemVer

	logger = logging.GetLogger("utils/platform")
)

const Kubernetes Mode = "kubernetes"
const Openshift Mode = "openshift"

func init() {
	ReloadPlatform()
}

func GetVersion() SemVer {
	return paasVersion
}

func GetPlatform() Mode {
	return mode
}

func ReloadPlatform() {
	platform := os.Getenv("PAAS_PLATFORM")
	if strings.EqualFold(platform, "kubernetes") || !strings.EqualFold(platform, "openshift") {
		mode = Kubernetes
	} else {
		mode = Openshift
	}

	var err error
	paasVersionEnv := os.Getenv("PAAS_VERSION")
	paasVersion, err = NewSemVer(paasVersionEnv)
	if err != nil {
		logger.Errorf("Could not read semver from PAAS_VERSION env variable value '%s':\n %v", paasVersionEnv, err)
	}
}
