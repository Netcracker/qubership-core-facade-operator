package services

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericClient[T client.Object] interface {
	Apply(ctx context.Context, req ctrl.Request, existingResReceiver, newObject client.Object, merge func(existingResReceiver, newObject T)) error
	Delete(ctx context.Context, req ctrl.Request, name string, existingResReceiver client.Object) error
	GetClient() client.Client
}

type GenericClientImpl[T client.Object] struct {
	resourceType string
	client       client.Client
	logger       logging.Logger
}

func NewGenericClient[T client.Object](client client.Client, resourceType string) *GenericClientImpl[T] {
	return &GenericClientImpl[T]{
		resourceType: resourceType,
		client:       client,
		logger:       logging.GetLogger(resourceType + "Client"),
	}
}

func (r *GenericClientImpl[T]) GetClient() client.Client {
	return r.client
}

func (r *GenericClientImpl[T]) Apply(ctx context.Context, req ctrl.Request, existingResReceiver, newObject client.Object,
	merge func(existingResReceiver, newObject T)) error {
	nameSpacedRequest := types.NamespacedName{
		Namespace: newObject.GetNamespace(),
		Name:      newObject.GetName(),
	}
	found := true
	err := r.client.Get(ctx, nameSpacedRequest, existingResReceiver, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			found = false
		} else {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get %s %v", r.resourceType, newObject.GetName()), err)
		}
	}

	if found {
		r.logger.InfoC(ctx, "[%v] Update %s %v", req.NamespacedName, r.resourceType, existingResReceiver.GetName())

		merge(existingResReceiver.(T), newObject.(T))

		options := &client.UpdateOptions{
			FieldManager: "facadeOperator",
		}
		err = r.client.Update(ctx, newObject, options)
		if err != nil {
			if k8sErrors.IsConflict(err) {
				r.logger.Debugf("[%v] %s %v already updated. Error: %v", req.NamespacedName, r.resourceType, newObject.GetName(), err)
				return &customerrors.ExpectedError{Message: err.Error()}
			}
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to update %s %v", r.resourceType, newObject.GetName()), err)
		}
	} else {
		r.logger.InfoC(ctx, "[%v] Create %s %v", req.NamespacedName, r.resourceType, newObject.GetName())
		options := &client.CreateOptions{
			FieldManager: "facadeOperator",
		}
		err = r.client.Create(ctx, newObject, options)
		if err != nil {
			if k8sErrors.IsAlreadyExists(err) {
				r.logger.Debugf("[%v] %s %v already created. Error: %v", req.NamespacedName, r.resourceType, newObject.GetName(), err)
				return &customerrors.ExpectedError{Message: err.Error()}
			}
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to create %s %v", r.resourceType, newObject.GetName()), err)
		}
	}

	return nil
}

func (r *GenericClientImpl[T]) Delete(ctx context.Context, req ctrl.Request, name string, existingResReceiver client.Object) error {
	r.logger.InfoC(ctx, "[%v] Start delete %s %v", req.NamespacedName, r.resourceType, name)

	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}
	err := r.client.Get(ctx, nameSpacedRequest, existingResReceiver, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.InfoC(ctx, "[%v] %s %v not found", req.NamespacedName, r.resourceType, name)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get %s %v", r.resourceType, name), err)
	}

	err = r.client.Delete(ctx, existingResReceiver)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.Debugf("[%v] %s %v already deleted. Error: %v", req.NamespacedName, r.resourceType, name, err)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete %s %v", r.resourceType, name), err)
	}
	r.logger.InfoC(ctx, "[%v] %s %v deleted", req.NamespacedName, r.resourceType, name)

	return nil
}
