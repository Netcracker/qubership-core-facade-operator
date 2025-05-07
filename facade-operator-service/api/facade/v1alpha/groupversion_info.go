// Package v1alpha contains API Schema definitions for the qubership.org v1alpha API group
// +kubebuilder:object:generate=true
// +groupName=nqubership.org
package v1alpha

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	groupVersion = schema.GroupVersion{Group: "qubership.org", Version: "v1alpha"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: groupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
