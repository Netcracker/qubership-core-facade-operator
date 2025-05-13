package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	gatewayV1Kind       = resolveCRKind(&facadeV1.Gateway{})
	gatewayV1ApiVersion = fmt.Sprintf("%s/%s", facadeV1.SchemeBuilder.GroupVersion.Group, facadeV1.SchemeBuilder.GroupVersion.Version)
	gatewayV1CRType     = fmt.Sprintf("%s-%s", gatewayV1ApiVersion, gatewayV1Kind)

	facadeV1AlphaKind       = resolveCRKind(&facadeV1Alpha.FacadeService{})
	facadeV1AlphaApiVersion = fmt.Sprintf("%s/%s", facadeV1Alpha.SchemeBuilder.GroupVersion.Group, facadeV1Alpha.SchemeBuilder.GroupVersion.Version)
	facadeV1AlphaCRType     = fmt.Sprintf("%s-%s", facadeV1AlphaApiVersion, facadeV1AlphaKind)
)

func resolveCRKind(cr facade.MeshGateway) string {
	valueOf := reflect.ValueOf(cr)
	return reflect.Indirect(valueOf).Type().Name()
}

type LastAppliedCr struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Deleted    bool   `json:"deleted"`
}

func (cr *LastAppliedCr) ResolveType() (facade.MeshGateway, error) {
	ApiKind := cr.ApiVersion + "-" + cr.Kind
	switch ApiKind {
	case gatewayV1CRType:
		return &facadeV1.Gateway{TypeMeta: metav1.TypeMeta{Kind: gatewayV1Kind, APIVersion: gatewayV1ApiVersion}}, nil
	case facadeV1AlphaCRType:
		return &facadeV1Alpha.FacadeService{TypeMeta: metav1.TypeMeta{Kind: facadeV1AlphaKind, APIVersion: facadeV1AlphaApiVersion}}, nil
	default:
		return nil, errs.NewError(customerrors.UnknownErrorCode, fmt.Sprintf("Can not resolve type for apiVersion '%s' and kind '%s'", cr.ApiVersion, cr.Kind), nil)
	}
}

func JsonMarshal[T any](object T) (string, error) {
	b, err := json.Marshal(object)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func JsonUnmarshal[T any](line string) (*T, error) {
	if line == "" {
		return nil, nil
	}
	data := new(T)
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return nil, err
	}

	return data, nil
}

func GetInt32EnvValueOrDefault(name string, defaultValue int32) int32 {
	value := os.Getenv(name)
	if value == "null" || value == "" {
		return defaultValue
	}
	result, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int32(result)
}

func ConvertToInt32(value any) (int32, error) {
	switch value.(type) {
	case int:
		return int32(value.(int)), nil
	case int32:
		return value.(int32), nil
	case int64:
		return int32(value.(int64)), nil
	case string:
		result, err := strconv.ParseInt(value.(string), 10, 32)
		if err != nil {
			return 0, errs.NewError(customerrors.InitParamsValidationError, fmt.Sprintf("Can not convert string value '%s' to int32", value), nil)
		}
		return int32(result), nil
	default:
		return 0, errs.NewError(customerrors.InitParamsValidationError, fmt.Sprintf("Can not convert '%T' type to 'int32'", value), nil)
	}
}

func GetPointer[T any](v T) *T {
	return &v
}

func MergeIntoMap[K comparable, V any](targetMap, valuesToMerge map[K]V) map[K]V {
	for key, val := range valuesToMerge {
		targetMap[key] = val
	}
	return targetMap
}

func MergeOwnerReferences(array1, array2 []metav1.OwnerReference) []metav1.OwnerReference {
	result := make([]metav1.OwnerReference, 0)
	result = append(result, array1...)
	alreadyAdded := false

	for _, item1 := range array2 {
		alreadyAdded = false
		for _, item2 := range result {
			if item1.UID == item2.UID {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			result = append(result, item1)
		}
	}

	return result
}

func GetValueOrDefault(value string, defaultValue string) string {
	if value == "null" || value == "" {
		return defaultValue
	}
	return value
}

func ResolveGatewayServiceName(crName string, cr facade.MeshGateway) string {
	if crName == facade.PublicGatewayService ||
		crName == facade.PrivateGatewayService ||
		crName == facade.InternalGatewayService {
		return crName
	}
	if cr != nil &&
		cr.GetGatewayType() == facade.Mesh &&
		cr.GetSpec().Gateway != "" &&
		cr.GetSpec().Gateway != "null" {
		return cr.GetSpec().Gateway
	}
	return crName
}

func GetResourceRequirements(ctx context.Context, cr facade.MeshGateway) corev1.ResourceRequirements {
	memoryLimit := GetValueOrDefault(cr.GetSpec().Env.FacadeGatewayMemoryLimit, DefaultFacadeGatewayMemoryLimit)
	if cr.GetName() == facade.EgressGateway {
		defaultFacadeGatewayMemoryLimitInt, _ := strconv.Atoi(strings.ReplaceAll(DefaultFacadeGatewayMemoryLimit, "Mi", ""))
		if defaultFacadeGatewayMemoryLimitInt < MinimumEgressGatewayMemoryLimitInt {
			memoryLimit = GetValueOrDefault(cr.GetSpec().Env.FacadeGatewayMemoryLimit, strconv.Itoa(MinimumEgressGatewayMemoryLimitInt)+"Mi")
		}
	}
	cpuLimit := GetValueOrDefault(fmt.Sprintf("%v", cr.GetSpec().Env.FacadeGatewayCpuLimit), DefaultFacadeGatewayCpuLimit)
	cpuRequest := GetValueOrDefault(fmt.Sprintf("%v", cr.GetSpec().Env.FacadeGatewayCpuRequest), DefaultFacadeGatewayCpuRequest)

	memoryLimitQuantity, err := resource.ParseQuantity(memoryLimit)
	if err != nil {
		logger.ErrorC(ctx, "Error during converting env variable for FacadeGatewayMemoryLimit: %v", err)
		memoryLimitQuantity = resource.MustParse(DefaultFacadeGatewayMemoryLimit)
	}

	cpuLimitQuantity, err := resource.ParseQuantity(cpuLimit)
	if err != nil {
		logger.ErrorC(ctx, "Error during converting env variable for FacadeGatewayCpuLimit: %v", err)
		cpuLimitQuantity = resource.MustParse(DefaultFacadeGatewayCpuLimit)
	}

	cpuRequestQuantity, err := resource.ParseQuantity(cpuRequest)
	if err != nil {
		logger.ErrorC(ctx, "Error during converting env variable for FacadeGatewayCpuRequest: %v", err)
		cpuRequestQuantity = resource.MustParse(DefaultFacadeGatewayCpuRequest)
	}

	return corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    cpuRequestQuantity,
			corev1.ResourceMemory: memoryLimitQuantity,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    cpuLimitQuantity,
			corev1.ResourceMemory: memoryLimitQuantity,
		},
	}
}

func GetBoolEnvValueOrDefault(name string, defaultValue bool) bool {
	value := os.Getenv(name)
	if value == "null" || value == "" {
		return defaultValue
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return result
}

func StringToIntValueOrDefault(ctx context.Context, value string, defaultValue int) int {
	if value == "null" || value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		logger.ErrorC(ctx, "Error during converting env variable: %v", err)
		return defaultValue
	}
	return result
}

func GetRequiredParameterString(name string) string {
	parameter := configloader.GetOrDefault(name, nil)
	if parameter == nil {
		panic(fmt.Errorf("parameter %s is required but could not be found and no default value was provided", name))
	}
	if s, ok := parameter.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", parameter)
}

type NamedResourceLock struct {
	lockedNames *sync.Map
}

func NewNamedResourceLock() *NamedResourceLock {
	return &NamedResourceLock{lockedNames: &sync.Map{}}
}

func (namedResLock *NamedResourceLock) Lock(name string) {
	_, alreadyExisted := namedResLock.lockedNames.LoadOrStore(name, name)
	for alreadyExisted {
		time.Sleep(100 * time.Millisecond)
		_, alreadyExisted = namedResLock.lockedNames.LoadOrStore(name, name)
	}
}

func (namedResLock *NamedResourceLock) Unlock(name string) {
	namedResLock.lockedNames.Delete(name)
}

type CloudTopology struct {
	TopologyKey       string  `json:"topologyKey"`
	MaxSkew           *int32  `json:"maxSkew"`
	WhenUnsatisfiable *string `json:"whenUnsatisfiable"`
}

var cloudTopologies []CloudTopology

func GetCloudTopologies(ctx context.Context) []CloudTopology {
	if CloudTopologiesJsonBase64 == "" {
		return nil
	}
	if cloudTopologies != nil {
		return cloudTopologies
	}
	jsonBytes, err := base64.StdEncoding.DecodeString(CloudTopologiesJsonBase64)
	if err != nil {
		logger.ErrorC(ctx, "Error during decoding env variable CLOUD_TOPOLOGIES: %v", err)
		return nil
	}

	var localCloudTopologies []CloudTopology
	err = json.Unmarshal(jsonBytes, &localCloudTopologies)
	if err != nil {
		logger.ErrorC(ctx, "Error during unmarshaling env variable CLOUD_TOPOLOGIES: %v", err)
		return nil
	}

	err = checkMandatoryParameters(localCloudTopologies)
	if err != nil {
		logger.ErrorC(ctx, "Error during checking CLOUD_TOPOLOGIES: %v", err)
		return nil
	}

	setDefaults(localCloudTopologies)
	cloudTopologies = localCloudTopologies
	return cloudTopologies
}

func checkMandatoryParameters(cloudTopologies []CloudTopology) error {
	for _, v := range cloudTopologies {
		if v.TopologyKey == "" {
			return fmt.Errorf("parameter CLOUD_TOPOLOGIES does't contain mandatory field 'topologyKey'")
		}
	}
	return nil
}

func setDefaults(cloudTopologies []CloudTopology) {
	defaultMaxSkew := int32(1)
	defaultWhenUnsatisfiable := string(corev1.ScheduleAnyway)
	for i, v := range cloudTopologies {
		if v.MaxSkew == nil {
			cloudTopologies[i].MaxSkew = &defaultMaxSkew
		}
		if v.WhenUnsatisfiable == nil {
			cloudTopologies[i].WhenUnsatisfiable = &defaultWhenUnsatisfiable
		}
	}
}
