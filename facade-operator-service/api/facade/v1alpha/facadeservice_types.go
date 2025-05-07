package v1alpha

import (
	"github.com/netcracker/qubership-core-facade-operator/api/facade"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FacadeServiceStatus defines the observed state of FacadeService
type FacadeServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FacadeService is the Schema for the facadeservices API
type FacadeService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   facade.FacadeServiceSpec `json:"spec,omitempty"`
	Status FacadeServiceStatus      `json:"status,omitempty"`
}

func (s *FacadeService) GetGatewayType() facade.GatewayType {
	if s.GetName() == facade.EgressGateway {
		return facade.Egress
	}
	return s.Spec.GetGatewayType()
}

func (s *FacadeService) GetSpec() facade.FacadeServiceSpec {
	return s.Spec
}

func (s *FacadeService) GetAPIVersion() string {
	return s.APIVersion
}

func (s *FacadeService) SetAPIVersion(version string) {
	s.APIVersion = version
}

func (s *FacadeService) GetKind() string {
	return s.Kind
}

func (s *FacadeService) SetKind(kind string) {
	s.Kind = kind
}

func (s *FacadeService) Priority() byte {
	return 0
}

//+kubebuilder:object:root=true

// FacadeServiceList contains a list of FacadeService
type FacadeServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FacadeService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FacadeService{}, &FacadeServiceList{})
}
