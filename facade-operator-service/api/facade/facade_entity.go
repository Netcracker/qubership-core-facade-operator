package facade

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type MeshGateway interface {
	metav1.Object
	metav1.Type
	GetGatewayType() GatewayType
	GetSpec() FacadeServiceSpec
	Priority() byte
}

type GatewayType string

const (
	EgressGateway          = "egress-gateway"
	PublicGatewayService   = "public-gateway-service"
	PrivateGatewayService  = "private-gateway-service"
	InternalGatewayService = "internal-gateway-service"
	IngressClassName       = "bg.mesh.netcracker.com"

	Egress  GatewayType = "egress"
	Ingress GatewayType = "ingress"
	Mesh    GatewayType = "mesh"
)

// +kubebuilder:object:root=true
type FacadeServiceSpec struct {
	Env                 FacadeServiceEnv `json:"env,omitempty"`
	Replicas            interface{}      `json:"replicas,omitempty"`
	Gateway             string           `json:"gateway,omitempty"`
	Port                int32            `json:"port,omitempty"`
	GatewayPorts        []GatewayPorts   `json:"gatewayPorts,omitempty"`
	MasterConfiguration bool             `json:"masterConfiguration,omitempty"`
	GatewayType         GatewayType      `json:"gatewayType,omitempty"`
	AllowVirtualHosts   *bool            `json:"allowVirtualHosts,omitempty"`
	Ingresses           []IngressSpec    `json:"ingresses,omitempty"`
	Hpa                 HPA              `json:"hpa,omitempty"`
}

func (s *FacadeServiceSpec) GetGatewayType() GatewayType {
	if s.GatewayType == "" || s.GatewayType == "null" {
		return Mesh
	}
	return s.GatewayType
}

type IngressSpec struct {
	Hostname    string `json:"hostname,omitempty"`
	IsGrpc      bool   `json:"isGrpc,omitempty"`
	GatewayPort int32  `json:"gatewayPort,omitempty"`
}

// +kubebuilder:object:root=true
type FacadeServiceEnv struct {
	FacadeGatewayCpuLimit      interface{} `json:"facadeGatewayCpuLimit,omitempty"`
	FacadeGatewayCpuRequest    interface{} `json:"facadeGatewayCpuRequest,omitempty"`
	FacadeGatewayMemoryLimit   string      `json:"facadeGatewayMemoryLimit,omitempty"`
	FacadeGatewayMemoryRequest string      `json:"facadeGatewayMemoryRequest,omitempty"`
	FacadeGatewayConcurrency   interface{} `json:"facadeGatewayConcurrency,omitempty"`
}

type GatewayPorts struct {
	Name     string `json:"name,omitempty"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol,omitempty"`
}

type HPA struct {
	MinReplicas           any         `json:"minReplicas,omitempty"`
	MaxReplicas           any         `json:"maxReplicas,omitempty"`
	AverageCpuUtilization any         `json:"averageCpuUtilization,omitempty"`
	ScaleUpBehavior       HPABehavior `json:"scaleUpBehavior,omitempty"`
	ScaleDownBehavior     HPABehavior `json:"scaleDownBehavior,omitempty"`
}

type HPABehavior struct {
	StabilizationWindowSeconds any           `json:"stabilizationWindowSeconds,omitempty"`
	SelectPolicy               string        `json:"selectPolicy,omitempty"`
	Policies                   []HPAPolicies `json:"policies,omitempty"`
}

type HPAPolicies struct {
	Type          string `json:"type,omitempty"`
	Value         any    `json:"value,omitempty"`
	PeriodSeconds any    `json:"periodSeconds,omitempty"`
}
