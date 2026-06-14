package services

import (
	"context"
	"fmt"

	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceClient interface {
	Apply(ctx context.Context, req ctrl.Request, service *corev1.Service) error
	Delete(ctx context.Context, req ctrl.Request, name string, gatewayName ...string) error
}

type ServiceClientImpl struct {
	client            client.Client
	deploymentsClient DeploymentClient
	logger            logging.Logger
}

func NewServiceClient(client client.Client, deploymentsClient DeploymentClient) *ServiceClientImpl {
	return &ServiceClientImpl{
		client:            client,
		deploymentsClient: deploymentsClient,
		logger:            logging.GetLogger("ServiceClient"),
	}
}

func (r *ServiceClientImpl) Apply(ctx context.Context, req ctrl.Request, service *corev1.Service) error {
	foundService := &corev1.Service{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	}
	found := true
	err := r.client.Get(ctx, nameSpacedRequest, foundService, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			found = false
		} else {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get service %v", service.Name), err)
		}
	}

	if found {
		if r.isWrongType(foundService) {
			r.logger.InfoC(ctx, "[%v] Service '%s' will be recreated. Expected type '%s'. Current clusterIp '%s'", req.NamespacedName, foundService.Name, utils.GetServiceType(), foundService.Spec.ClusterIP)
			if err = r.Delete(ctx, req, foundService.Name); err != nil {
				return err
			}
			return r.create(ctx, req, service)
		} else {
			service.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(service.ObjectMeta.OwnerReferences, foundService.ObjectMeta.OwnerReferences)
			return r.update(ctx, req, service, foundService)
		}
	} else {
		return r.create(ctx, req, service)
	}
}

func (r *ServiceClientImpl) update(ctx context.Context, req ctrl.Request, service, foundService *corev1.Service) error {
	r.logger.InfoC(ctx, "[%v] Update service %v", req.NamespacedName, foundService.Name)
	service.Spec.ClusterIP = foundService.Spec.ClusterIP
	service.ObjectMeta.ResourceVersion = foundService.ObjectMeta.ResourceVersion

	options := &client.UpdateOptions{
		FieldManager: "facadeOperator",
	}
	if err := r.client.Update(ctx, service, options); err != nil {
		if k8sErrors.IsConflict(err) {
			r.logger.Debugf("[%v] Service %v already updated. Error: %v", req.NamespacedName, service.Name, err)
			return &customerrors.ExpectedError{Message: err.Error()}
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to update service %v", service.Name), err)
	}
	return nil
}

func (r *ServiceClientImpl) create(ctx context.Context, req ctrl.Request, service *corev1.Service) error {
	r.logger.InfoC(ctx, "[%v] Create service %v", req.NamespacedName, service.Name)
	options := &client.CreateOptions{
		FieldManager: "facadeOperator",
	}
	if err := r.client.Create(ctx, service, options); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.logger.Debugf("[%v] Service %v already created. Error: %v", req.NamespacedName, service.Name, err)
			return &customerrors.ExpectedError{Message: err.Error()}
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to create service %v", service.Name), err)
	}
	return nil
}

func (r *ServiceClientImpl) isWrongType(foundService *corev1.Service) bool {
	clusterIp := foundService.Spec.ClusterIP
	serviceType := utils.GetServiceType()
	return serviceType == utils.HeadLess && clusterIp != "None" ||
		serviceType == utils.ClusterIp && (clusterIp == "" || clusterIp == "None")
}

func (r *ServiceClientImpl) Delete(ctx context.Context, req ctrl.Request, name string, gatewayName ...string) error {
	r.logger.InfoC(ctx, "[%v] Start delete service %v", req.NamespacedName, name)

	foundService := &corev1.Service{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}
	err := r.client.Get(ctx, nameSpacedRequest, foundService, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.InfoC(ctx, "[%v] Service %v not found", req.NamespacedName, name)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get service %v", name), err)
	}

	// If expected gateway name is provided, check for early match
	if len(gatewayName) > 0 && gatewayName[0] != "" {
		if foundService.Spec.Selector["app"] == gatewayName[0] {
			r.logger.InfoC(ctx, "[%v] Service '%s' selector matches expected gateway '%s'", req.NamespacedName, name, gatewayName[0])
		} else {
			// Selector doesn't match provided gateway name, fall back to full validation via deployment
			r.logger.InfoC(ctx, "[%v] Service '%s' selector does not match expected gateway '%s', validating via deployment", req.NamespacedName, name, gatewayName[0])
			isManaged, err := r.selectsManagedGateway(ctx, req, foundService)
			if err != nil {
				return err
			}
			if !isManaged {
				r.logger.InfoC(ctx, "[%v] Service '%s' does not select a managed gateway deployment, skipping delete", req.NamespacedName, name)
				return nil
			}
		}
	} else {
		// No expected gateway provided, validate against deployment labels
		isManaged, err := r.selectsManagedGateway(ctx, req, foundService)
		if err != nil {
			return err
		}
		if !isManaged {
			r.logger.InfoC(ctx, "[%v] Service '%s' does not select a managed gateway deployment, skipping delete", req.NamespacedName, name)
			return nil
		}
	}

	rv := foundService.ResourceVersion
	err = r.client.Delete(ctx, foundService, &client.DeleteOptions{
		Preconditions: &metav1.Preconditions{
			ResourceVersion: &rv,
		},
	})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.Debugf("[%v] Service %v already deleted. Error: %v", req.NamespacedName, name, err)
			return nil
		}
		if k8sErrors.IsConflict(err) {
			r.logger.Debugf("[%v] Service %v conflict on delete. Error: %v", req.NamespacedName, name, err)
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete service %v", foundService.Name), err)
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete service %v", name), err)
	}
	r.logger.InfoC(ctx, "[%v] Service %v deleted", req.NamespacedName, name)

	return nil
}

func (r *ServiceClientImpl) selectsManagedGateway(ctx context.Context, req ctrl.Request, svc *corev1.Service) (bool, error) {
	gatewayName := svc.Spec.Selector["app"]
	if gatewayName == "" {
		return false, nil
	}

	deployment, err := r.deploymentsClient.Get(ctx, req, gatewayName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	if deployment == nil {
		return true, nil
	}

	if deployment.GetLabels()[utils.FacadeGateway] == "true" {
		return true, nil
	}
	return false, nil
}
