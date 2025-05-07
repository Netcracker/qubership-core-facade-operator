package predicates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestIgnoreUpdateStatusPredicate(t *testing.T) {
	object1 := &TestPredicateStruct{ObjectMeta: metav1.ObjectMeta{Name: "name", Generation: 1}}
	object2 := &TestPredicateStruct{ObjectMeta: metav1.ObjectMeta{Name: "name", Generation: 2}}

	updateEvent := event.UpdateEvent{ObjectOld: object1}
	assert.True(t, IgnoreUpdateStatusPredicate.Update(updateEvent))

	updateEvent = event.UpdateEvent{ObjectNew: object2}
	assert.True(t, IgnoreUpdateStatusPredicate.Update(updateEvent))

	updateEvent = event.UpdateEvent{ObjectOld: object1, ObjectNew: object1}
	assert.False(t, IgnoreUpdateStatusPredicate.Update(updateEvent))

	updateEvent = event.UpdateEvent{ObjectOld: object2, ObjectNew: object2}
	assert.False(t, IgnoreUpdateStatusPredicate.Update(updateEvent))

	updateEvent = event.UpdateEvent{ObjectOld: object1, ObjectNew: object2}
	assert.True(t, IgnoreUpdateStatusPredicate.Update(updateEvent))
}

func TestObjectNamePredicate_shouldTrue_whenObjectWithSameName(t *testing.T) {
	predicate := ObjectNamePredicate{"test"}
	object := &TestPredicateStruct{ObjectMeta: metav1.ObjectMeta{Name: predicate.Name}}

	updateEvent := event.UpdateEvent{ObjectOld: object}
	assert.True(t, predicate.Update(updateEvent))

	updateEvent = event.UpdateEvent{ObjectNew: object}
	assert.True(t, predicate.Update(updateEvent))

	createEvent := event.CreateEvent{Object: object}
	assert.True(t, predicate.Create(createEvent))

	deleteEvent := event.DeleteEvent{Object: object}
	assert.True(t, predicate.Delete(deleteEvent))

	genericEvent := event.GenericEvent{Object: object}
	assert.True(t, predicate.Generic(genericEvent))
}

func TestObjectNamePredicate_shouldFalse_whenObjectWithDiffName(t *testing.T) {
	predicate := ObjectNamePredicate{"test"}
	object := &TestPredicateStruct{ObjectMeta: metav1.ObjectMeta{Name: predicate.Name + "suffix"}}

	updateEvent := event.UpdateEvent{ObjectOld: object}
	assert.False(t, predicate.Update(updateEvent))

	updateEvent = event.UpdateEvent{ObjectNew: object}
	assert.False(t, predicate.Update(updateEvent))

	createEvent := event.CreateEvent{Object: object}
	assert.False(t, predicate.Create(createEvent))

	deleteEvent := event.DeleteEvent{Object: object}
	assert.False(t, predicate.Delete(deleteEvent))

	genericEvent := event.GenericEvent{Object: object}
	assert.False(t, predicate.Generic(genericEvent))
}

type TestPredicateStruct struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func (in *TestPredicateStruct) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *TestPredicateStruct) DeepCopy() *TestPredicateStruct {
	if in == nil {
		return nil
	}
	out := new(TestPredicateStruct)
	in.DeepCopyInto(out)
	return out
}

func (in *TestPredicateStruct) DeepCopyInto(out *TestPredicateStruct) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
}
