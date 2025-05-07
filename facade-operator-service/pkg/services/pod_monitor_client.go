package services

import (
	"context"
	"fmt"
	monitoringV1 "github.com/netcracker/qubership-core-facade-operator/api/monitoring/v1"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodMonitorClient interface {
	Create(ctx context.Context, req ctrl.Request, podMonitor *monitoringV1.PodMonitor) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
}

type PodMonitorClientImpl struct {
	client client.Client
	logger logging.Logger
}

func NewPodMonitorClient(client client.Client) *PodMonitorClientImpl {
	return &PodMonitorClientImpl{
		client: client,
		logger: logging.GetLogger("PodMonitorClient"),
	}
}

func (r *PodMonitorClientImpl) Create(ctx context.Context, req ctrl.Request, podMonitor *monitoringV1.PodMonitor) error {
	r.logger.InfoC(ctx, "[%v] Start create PodMonitor %v", req.NamespacedName, podMonitor.Name)

	foundPodMonitor := &monitoringV1.PodMonitor{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: podMonitor.Namespace,
		Name:      podMonitor.Name,
	}

	found := true
	err := r.client.Get(ctx, nameSpacedRequest, foundPodMonitor)
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			// go to update branch. found = true
		} else if k8sErrors.IsNotFound(err) {
			found = false
		} else {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get PodMonitor %v", podMonitor.Name), err)
		}
	}
	if found {
		podMonitor.SetResourceVersion(foundPodMonitor.GetResourceVersion())
		updOptions := &client.UpdateOptions{
			FieldManager: "facadeOperator",
		}
		podMonitor.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(podMonitor.ObjectMeta.OwnerReferences, foundPodMonitor.ObjectMeta.OwnerReferences)
		err = r.client.Update(ctx, podMonitor, updOptions)
		if err != nil {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to update PodMonitor %v", podMonitor.Name), err)
		}
	} else {
		options := &client.CreateOptions{
			FieldManager: "facadeOperator",
		}
		err = r.client.Create(ctx, podMonitor, options)
		if err != nil {
			if k8sErrors.IsAlreadyExists(err) {
				r.logger.Debugf("[%v] PodMonitor %v already exist. Error: %v", req.NamespacedName, podMonitor.Name, err)
				return nil
			}
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to create PodMonitor %v", podMonitor.Name), err)
		}
	}

	return nil
}

func (r *PodMonitorClientImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	r.logger.InfoC(ctx, "[%v] Start delete PodMonitor %v", req.NamespacedName, name)

	foundPodMonitor := &monitoringV1.PodMonitor{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}
	err := r.client.Get(ctx, nameSpacedRequest, foundPodMonitor, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.InfoC(ctx, "[%v] PodMonitor %v not found", req.NamespacedName, name)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get PodMonitor %v", name), err)
	}

	err = r.client.Delete(ctx, foundPodMonitor)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.Debugf("[%v] PodMonitor %v already deleted. Error: %v", req.NamespacedName, name, err)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete PodMonitor %v", name), err)
	}
	r.logger.InfoC(ctx, "[%v] PodMonitor %v deleted", req.NamespacedName, name)

	return nil
}
