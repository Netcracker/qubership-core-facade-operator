package helper

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GatewayKind       = "Gateway"
	FacadeServiceKind = "FacadeService"
)

type ApiGroupVersionProvider interface {
	GetApiGroups(kind string) []schema.GroupVersion
}

type DefaultApiGroupProvider struct {
}

func (p *DefaultApiGroupProvider) GetApiGroups(kind string) []schema.GroupVersion {
	switch kind {
	case GatewayKind:
		return []schema.GroupVersion{
			{Group: "core.qubership.org", Version: "v1"},
		}
	case FacadeServiceKind:
		return []schema.GroupVersion{
			{Group: "qubership.org", Version: "v1alpha"},
		}
	default:
		panic("cannot resolve api GroupVersion for kind: " + kind)
	}
}
