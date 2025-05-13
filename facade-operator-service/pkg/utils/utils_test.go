package utils

import (
	"context"
	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetInt32EnvValueOrDefault(t *testing.T) {
	defer os.Unsetenv("test_env")
	defaultValue := int32(100500)

	result := GetInt32EnvValueOrDefault("test_env", defaultValue)
	assert.Equal(t, defaultValue, result)

	os.Setenv("test_env", "1")
	result = GetInt32EnvValueOrDefault("test_env", defaultValue)
	assert.Equal(t, int32(1), result)

	os.Setenv("test_env", "null")
	result = GetInt32EnvValueOrDefault("test_env", defaultValue)
	assert.Equal(t, defaultValue, result)

	os.Setenv("test_env", "NotInt32")
	result = GetInt32EnvValueOrDefault("test_env", defaultValue)
	assert.Equal(t, defaultValue, result)
}

func TestConvertToInt32(t *testing.T) {
	//success cases
	result, err := ConvertToInt32(int(1))
	assert.Nil(t, err)
	assert.Equal(t, int32(1), result)

	result, err = ConvertToInt32(int32(1))
	assert.Nil(t, err)
	assert.Equal(t, int32(1), result)

	result, err = ConvertToInt32(int64(1))
	assert.Nil(t, err)
	assert.Equal(t, int32(1), result)

	result, err = ConvertToInt32("1")
	assert.Nil(t, err)
	assert.Equal(t, int32(1), result)

	//error cases
	result, err = ConvertToInt32("1aaaaaa")
	assert.NotNil(t, err)
	assert.Equal(t, int32(0), result)
	assert.Equal(t, customerrors.InitParamsValidationError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, "Can not convert string value '1aaaaaa' to int32", err.(*errs.ErrCodeError).Detail)

	result, err = ConvertToInt32(LastAppliedCr{})
	assert.NotNil(t, err)
	assert.Equal(t, int32(0), result)
	assert.Equal(t, customerrors.InitParamsValidationError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, "Can not convert 'utils.LastAppliedCr' type to 'int32'", err.(*errs.ErrCodeError).Detail)
}

func TestJsonMarshalAndUnmarshal(t *testing.T) {
	lastAppliedCr := LastAppliedCr{
		ApiVersion: "testVersion",
		Kind:       "testKind",
		Name:       "testName",
	}
	lastAppliedResultMarshaled, err := JsonMarshal(lastAppliedCr)
	assert.Nil(t, err)
	assert.NotNil(t, lastAppliedResultMarshaled)

	lastAppliedResultUnmarshaled, err := JsonUnmarshal[LastAppliedCr](lastAppliedResultMarshaled)
	assert.Nil(t, err)
	assert.NotNil(t, lastAppliedResultUnmarshaled)
	assert.Equal(t, lastAppliedCr.ApiVersion, lastAppliedResultUnmarshaled.ApiVersion)
	assert.Equal(t, lastAppliedCr.Kind, lastAppliedResultUnmarshaled.Kind)
	assert.Equal(t, lastAppliedCr.Name, lastAppliedResultUnmarshaled.Name)
}

func TestGetValueOrDefault(t *testing.T) {
	value := "test1"
	defaultValue := "test2"

	result := GetValueOrDefault(value, defaultValue)
	assert.Equal(t, value, result)

	result = GetValueOrDefault("", defaultValue)
	assert.Equal(t, defaultValue, result)
}

func TestGetResourceRequirements(t *testing.T) {
	facadeService := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "1",
		},
		Spec: facade.FacadeServiceSpec{
			MasterConfiguration: false,
			Replicas:            int32(1),
			Port:                8080,
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:    "1m",
				FacadeGatewayCpuRequest:  "2m",
				FacadeGatewayMemoryLimit: "3Mi",
			},
			Gateway: "deployment2",
		},
	}
	resource := GetResourceRequirements(context.Background(), facadeService)
	assert.Equal(t, facadeService.Spec.Env.FacadeGatewayMemoryLimit, resource.Limits.Memory().String())
	assert.Equal(t, facadeService.Spec.Env.FacadeGatewayMemoryLimit, resource.Requests.Memory().String())
	assert.Equal(t, facadeService.Spec.Env.FacadeGatewayCpuLimit, resource.Limits.Cpu().String())
	assert.Equal(t, facadeService.Spec.Env.FacadeGatewayCpuRequest, resource.Requests.Cpu().String())
}

func TestGetResourceRequirements_CpuIntAndEmptyStr(t *testing.T) {
	defaultFacadeGatewayCpuRequest := "500m"
	reflect.ValueOf(&DefaultFacadeGatewayCpuRequest).Elem().SetString(defaultFacadeGatewayCpuRequest)
	facadeService := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "1",
		},
		Spec: facade.FacadeServiceSpec{
			MasterConfiguration: false,
			Replicas:            int32(1),
			Port:                8080,
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:    1,
				FacadeGatewayCpuRequest:  "",
				FacadeGatewayMemoryLimit: "3Mi",
			},
			Gateway: "deployment2",
		},
	}
	resource := GetResourceRequirements(context.Background(), facadeService)
	assert.Equal(t, fmt.Sprintf("%v", facadeService.Spec.Env.FacadeGatewayCpuLimit), resource.Limits.Cpu().String())
	assert.Equal(t, defaultFacadeGatewayCpuRequest, resource.Requests.Cpu().String())
}

func TestGetResourceRequirementsEgressGatewayMinimumDefaultMemory1(t *testing.T) {
	reflect.ValueOf(&DefaultFacadeGatewayMemoryLimit).Elem().SetString("32Mi")
	minimumEgressGatewayMemoryLimit := strconv.Itoa(MinimumEgressGatewayMemoryLimitInt) + "Mi"
	facadeService := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "egress-gateway",
		},
		Spec: facade.FacadeServiceSpec{
			MasterConfiguration: false,
			Replicas:            int32(1),
			Port:                8080,
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:   "1m",
				FacadeGatewayCpuRequest: "2m",
			},
		},
	}
	resource := GetResourceRequirements(context.Background(), facadeService)
	assert.Equal(t, minimumEgressGatewayMemoryLimit, resource.Limits.Memory().String())
	assert.Equal(t, minimumEgressGatewayMemoryLimit, resource.Requests.Memory().String())
}

func TestGetResourceRequirementsEgressGatewayMinimumDefaultMemory2(t *testing.T) {
	var defaultFacadeGatewayMemoryLimit = "128Mi"
	reflect.ValueOf(&DefaultFacadeGatewayMemoryLimit).Elem().SetString(defaultFacadeGatewayMemoryLimit)
	facadeService := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "egress-gateway",
		},
		Spec: facade.FacadeServiceSpec{
			MasterConfiguration: false,
			Replicas:            int32(1),
			Port:                8080,
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:   "1m",
				FacadeGatewayCpuRequest: "2m",
			},
		},
	}
	resource := GetResourceRequirements(context.Background(), facadeService)
	assert.Equal(t, defaultFacadeGatewayMemoryLimit, resource.Limits.Memory().String())
	assert.Equal(t, defaultFacadeGatewayMemoryLimit, resource.Requests.Memory().String())
}

func TestGetResourceRequirementsEgressGatewayMinimumDefaultMemory3(t *testing.T) {
	var defaultFacadeGatewayMemoryLimit = "128Mi"
	reflect.ValueOf(&DefaultFacadeGatewayMemoryLimit).Elem().SetString(defaultFacadeGatewayMemoryLimit)
	facadeService := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "egress-gateway",
		},
		Spec: facade.FacadeServiceSpec{
			MasterConfiguration: false,
			Replicas:            int32(1),
			Port:                8080,
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:    "1m",
				FacadeGatewayCpuRequest:  "2m",
				FacadeGatewayMemoryLimit: "3Mi",
			},
		},
	}
	resource := GetResourceRequirements(context.Background(), facadeService)
	assert.Equal(t, facadeService.Spec.Env.FacadeGatewayMemoryLimit, resource.Limits.Memory().String())
	assert.Equal(t, facadeService.Spec.Env.FacadeGatewayMemoryLimit, resource.Requests.Memory().String())
}

func TestGetBoolEnvValueOrDefault(t *testing.T) {
	envName := "TEST"
	defaultValue := false

	os.Setenv(envName, "12345")
	defer os.Unsetenv(envName)

	value := GetBoolEnvValueOrDefault(envName, defaultValue)
	assert.Equal(t, defaultValue, value)

	os.Setenv(envName, "")
	value = GetBoolEnvValueOrDefault(envName, defaultValue)
	assert.Equal(t, defaultValue, value)

	os.Setenv(envName, "true")
	value = GetBoolEnvValueOrDefault(envName, defaultValue)
	assert.Equal(t, true, value)
}

func TestStringToIntValueOrDefault(t *testing.T) {
	value := StringToIntValueOrDefault(context.Background(), "", 15)
	assert.Equal(t, 15, value)

	value = StringToIntValueOrDefault(context.Background(), "null", 15)
	assert.Equal(t, 15, value)

	value = StringToIntValueOrDefault(context.Background(), "6", 15)
	assert.Equal(t, 6, value)

	value = StringToIntValueOrDefault(context.Background(), "6s", 15)
	assert.Equal(t, 15, value)
}

func TestGetCloudTopologies(t *testing.T) {
	//[{"maxSkew":5,"topologyKey":"topology.kubernetes.io/zone","whenUnsatisfiable":"DoNotSchedule"},{"maxSkew":10,"topologyKey":"kubernetes.io/hostname","whenUnsatisfiable":"ScheduleAnyway"}]
	CloudTopologiesJsonBase64 = "W3sibWF4U2tldyI6NSwidG9wb2xvZ3lLZXkiOiJ0b3BvbG9neS5rdWJlcm5ldGVzLmlvL3pvbmUiLCJ3aGVuVW5zYXRpc2ZpYWJsZSI6IkRvTm90U2NoZWR1bGUifSx7Im1heFNrZXciOjEwLCJ0b3BvbG9neUtleSI6Imt1YmVybmV0ZXMuaW8vaG9zdG5hbWUiLCJ3aGVuVW5zYXRpc2ZpYWJsZSI6IlNjaGVkdWxlQW55d2F5In1d"

	expectedTopology1 := CloudTopology{
		MaxSkew:           GetPointer(int32(5)),
		TopologyKey:       "topology.kubernetes.io/zone",
		WhenUnsatisfiable: GetPointer("DoNotSchedule"),
	}
	expectedTopology2 := CloudTopology{
		MaxSkew:           GetPointer(int32(10)),
		TopologyKey:       "kubernetes.io/hostname",
		WhenUnsatisfiable: GetPointer("ScheduleAnyway"),
	}

	actualTopologies := GetCloudTopologies(context.Background())

	assert.Contains(t, actualTopologies, expectedTopology1)
	assert.Contains(t, actualTopologies, expectedTopology2)
	cloudTopologies = nil

	//[{"topologyKey":"topology.kubernetes.io/zone","whenUnsatisfiable":"DoNotSchedule"},{"topologyKey":"kubernetes.io/hostname"}]
	CloudTopologiesJsonBase64 = "W3sidG9wb2xvZ3lLZXkiOiJ0b3BvbG9neS5rdWJlcm5ldGVzLmlvL3pvbmUiLCJ3aGVuVW5zYXRpc2ZpYWJsZSI6IkRvTm90U2NoZWR1bGUifSx7InRvcG9sb2d5S2V5Ijoia3ViZXJuZXRlcy5pby9ob3N0bmFtZSJ9XQ=="
	expectedTopology1 = CloudTopology{
		MaxSkew:           GetPointer(int32(1)),
		TopologyKey:       "topology.kubernetes.io/zone",
		WhenUnsatisfiable: GetPointer("DoNotSchedule"),
	}
	expectedTopology2 = CloudTopology{
		MaxSkew:           GetPointer(int32(1)),
		TopologyKey:       "kubernetes.io/hostname",
		WhenUnsatisfiable: GetPointer("ScheduleAnyway"),
	}

	actualTopologies = GetCloudTopologies(context.Background())

	assert.Contains(t, actualTopologies, expectedTopology1)
	assert.Contains(t, actualTopologies, expectedTopology2)
	cloudTopologies = nil

	//[{"topologyKey":"topology.kubernetes.io/zone","whenUnsatisfiable":"DoNotSchedule"}]
	CloudTopologiesJsonBase64 = "W3sidG9wb2xvZ3lLZXkiOiJ0b3BvbG9neS5rdWJlcm5ldGVzLmlvL3pvbmUiLCJ3aGVuVW5zYXRpc2ZpYWJsZSI6IkRvTm90U2NoZWR1bGUifV0="
	expectedTopology1 = CloudTopology{
		MaxSkew:           GetPointer(int32(1)),
		TopologyKey:       "topology.kubernetes.io/zone",
		WhenUnsatisfiable: GetPointer("DoNotSchedule"),
	}

	actualTopologies = GetCloudTopologies(context.Background())

	assert.Contains(t, actualTopologies, expectedTopology1)
	cloudTopologies = nil

	//empty
	CloudTopologiesJsonBase64 = ""

	actualTopologies = GetCloudTopologies(context.Background())
	assert.Nil(t, actualTopologies)

	//Mandatory field topologyKey is absent [{"maxSkew":5,"whenUnsatisfiable":"DoNotSchedule"}]
	CloudTopologiesJsonBase64 = "W3sibWF4U2tldyI6NSwid2hlblVuc2F0aXNmaWFibGUiOiJEb05vdFNjaGVkdWxlIn1d"

	actualTopologies = GetCloudTopologies(context.Background())
	assert.Nil(t, actualTopologies)
	//Check twice
	actualTopologies = GetCloudTopologies(context.Background())
	assert.Nil(t, actualTopologies)
}

func TestMergeOwnerReferences(t *testing.T) {
	r1 := metav1.OwnerReference{
		Name: "Ref1",
		UID:  types.UID("1111"),
	}
	r2 := metav1.OwnerReference{
		Name: "Ref2",
		UID:  types.UID("2222"),
	}
	r3 := metav1.OwnerReference{
		Name: "Ref3",
		UID:  types.UID("3333"),
	}

	array1 := []metav1.OwnerReference{r1, r2}
	array2 := []metav1.OwnerReference{r2, r3}
	array3 := []metav1.OwnerReference{r1, r2, r3}
	array4 := []metav1.OwnerReference{}

	mergedReferences := MergeOwnerReferences(array1, array2)
	assert.Equal(t, array3, mergedReferences)

	mergedReferences = MergeOwnerReferences(array1, array4)
	assert.Equal(t, array1, mergedReferences)

	mergedReferences = MergeOwnerReferences(array4, array2)
	assert.Equal(t, array2, mergedReferences)
}
