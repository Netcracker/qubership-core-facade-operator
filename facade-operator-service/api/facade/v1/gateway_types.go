package v1

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Phase string

const (
	UpdatedPhase              Phase = "Updated"
	BackingOffPhase           Phase = "BackingOff"
	InvalidConfigurationPhase Phase = "InvalidConfiguration"
	WaitingForDependencyPhase Phase = "WaitingForDependency"
	UpdatingPhase             Phase = "Updating"
	UnknownPhase              Phase = "Unknown"
)

// GatewayStatus defines the observed state of MeshGateway
type GatewayStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	Phase              Phase `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Gateway is the Schema for the MeshGateways API
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   facade.FacadeServiceSpec `json:"spec,omitempty"`
	Status GatewayStatus            `json:"status,omitempty"`
}

func (s *Gateway) GetGatewayType() facade.GatewayType {
	if s.GetName() == facade.EgressGateway {
		return facade.Egress
	}
	s.GetName()
	return s.Spec.GetGatewayType()
}

func (s *Gateway) GetSpec() facade.FacadeServiceSpec {
	return s.Spec
}

func (s *Gateway) GetAPIVersion() string {
	return s.APIVersion
}

func (s *Gateway) SetAPIVersion(version string) {
	s.APIVersion = version
}

func (s *Gateway) GetKind() string {
	return s.Kind
}

func (s *Gateway) SetKind(kind string) {
	s.Kind = kind
}

func (s *Gateway) Priority() byte {
	return 1
}

//+kubebuilder:object:root=true

// GatewayList contains a list of MeshGateway
type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gateway{}, &GatewayList{})
}
