package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodMonitorSpec defines the desired state of PodMonitor
type PodMonitorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	JobLabel            string                `json:"jobLabel,omitempty"`
	NamespaceSelector   *NamespaceSelector    `json:"namespaceSelector,omitempty"`
	PodMetricsEndpoints []PodMetricsEndpoint  `json:"podMetricsEndpoints,omitempty"`
	Selector            *metav1.LabelSelector `json:"selector,omitempty"`
}

// PodMonitorStatus defines the observed state of PodMonitor
type PodMonitorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PodMonitor is the Schema for the PodMonitors API
type PodMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodMonitorSpec   `json:"spec,omitempty"`
	Status PodMonitorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PodMonitorList contains a list of PodMonitor
type PodMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodMonitor `json:"items"`
}

type PodMetricsEndpoint struct {
	Interval string `json:"interval,omitempty"`
	Port     string `json:"port,omitempty"`
	Scheme   string `json:"scheme,omitempty"`
	Path     string `json:"path,omitempty"`
}

type NamespaceSelector struct {
	MatchNames []string `json:"matchNames,omitempty"`
}

func init() {
	SchemeBuilder.Register(&PodMonitor{}, &PodMonitorList{})
}
