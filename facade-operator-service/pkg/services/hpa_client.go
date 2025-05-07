package services

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	hpav2 "k8s.io/api/autoscaling/v2"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HPAClient interface {
	Create(ctx context.Context, req ctrl.Request, hpa *hpav2.HorizontalPodAutoscaler) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
}

type HPAClientImpl struct {
	client client.Client
	logger logging.Logger
}

func NewHPAClient(client client.Client) *HPAClientImpl {
	return &HPAClientImpl{
		client: client,
		logger: logging.GetLogger("HPAClient"),
	}
}

func (r *HPAClientImpl) Create(ctx context.Context, req ctrl.Request, hpa *hpav2.HorizontalPodAutoscaler) error {
	r.logger.InfoC(ctx, "[%v] Start create HPA '%s'", req.NamespacedName, hpa.Name)

	foundHPA := &hpav2.HorizontalPodAutoscaler{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: hpa.Namespace,
		Name:      hpa.Name,
	}

	found := true
	err := r.client.Get(ctx, nameSpacedRequest, foundHPA)
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			// go to update branch. found = true
		} else if k8sErrors.IsNotFound(err) {
			found = false
		} else {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get HPA '%s'", hpa.Name), err)
		}
	}
	if found {
		hpa.SetResourceVersion(foundHPA.GetResourceVersion())
		updOptions := &client.UpdateOptions{
			FieldManager: "facadeOperator",
		}
		hpa.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(hpa.ObjectMeta.OwnerReferences, foundHPA.ObjectMeta.OwnerReferences)
		err = r.client.Update(ctx, hpa, updOptions)
		if err != nil {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to update HPA '%s'", hpa.Name), err)
		}
	} else {
		options := &client.CreateOptions{
			FieldManager: "facadeOperator",
		}
		err = r.client.Create(ctx, hpa, options)
		if err != nil {
			if k8sErrors.IsAlreadyExists(err) {
				r.logger.Debugf("[%v] HPA %v already exist. Error: %v", req.NamespacedName, hpa.Name, err)
				return nil
			}
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to create HPA '%s'", hpa.Name), err)
		}
	}

	return nil
}

func (r *HPAClientImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	r.logger.InfoC(ctx, "[%v] Start delete HPA '%s'", req.NamespacedName, name)

	foundHPA := &hpav2.HorizontalPodAutoscaler{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}
	err := r.client.Get(ctx, nameSpacedRequest, foundHPA, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.InfoC(ctx, "[%v] HPA '%s' not found", req.NamespacedName, name)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get HPA '%s'", name), err)
	}

	err = r.client.Delete(ctx, foundHPA)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.Debugf("[%v] HPA '%s' already deleted. Error: %v", req.NamespacedName, name, err)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete HPA '%s'", name), err)
	}
	r.logger.InfoC(ctx, "[%v] HPA '%s' deleted", req.NamespacedName, name)

	return nil
}
