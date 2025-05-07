package predicates

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	IgnoreUpdateStatusPredicate = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return true
			}
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}
)

type ObjectNamePredicate struct {
	Name string
}

func (p ObjectNamePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld != nil {
		return e.ObjectOld.GetName() == p.Name
	}
	if e.ObjectNew != nil {
		return e.ObjectNew.GetName() == p.Name
	}

	return false
}

func (p ObjectNamePredicate) Create(e event.CreateEvent) bool {
	return e.Object.GetName() == p.Name
}

func (p ObjectNamePredicate) Delete(e event.DeleteEvent) bool {
	return e.Object.GetName() == p.Name
}

func (p ObjectNamePredicate) Generic(e event.GenericEvent) bool {
	return e.Object.GetName() == p.Name
}
