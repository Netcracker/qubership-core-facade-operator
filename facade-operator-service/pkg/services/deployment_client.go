package services

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	v1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentClient interface {
	Get(ctx context.Context, req ctrl.Request, name string) (*v1.Deployment, error)
	DeleteMasterCRLabel(ctx context.Context, req ctrl.Request, deploymentName string) error
	SetLastAppliedCR(ctx context.Context, req ctrl.Request, deploymentName string, lastAppliedCr *utils.LastAppliedCr) error
	Apply(ctx context.Context, req ctrl.Request, newDeployment *v1.Deployment) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
	GetMeshRouterDeployments(ctx context.Context, req ctrl.Request) (*v1.DeploymentList, error)
	IsFacadeGateway(ctx context.Context, req ctrl.Request, name string) (bool, error)
	GetMasterCR(ctx context.Context, req ctrl.Request, name string) (string, error)
}
type deploymentClientImpl struct {
	client client.Client
	logger logging.Logger
}

func NewDeploymentClientImpl(client client.Client) *deploymentClientImpl {
	return &deploymentClientImpl{
		client: client,
		logger: logging.GetLogger("DeploymentClient"),
	}
}

func (r *deploymentClientImpl) Get(ctx context.Context, req ctrl.Request, name string) (*v1.Deployment, error) {
	r.logger.InfoC(ctx, "[%v] Get deployment %v", req.NamespacedName, name)
	foundDeployment := &v1.Deployment{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}
	err := r.client.Get(ctx, nameSpacedRequest, foundDeployment, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.InfoC(ctx, "[%v] Deployment %v not found", req.NamespacedName, name)
			return nil, nil
		}
		return nil, errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get deployment: %v", name), err)
	}

	return foundDeployment, nil
}

func (r *deploymentClientImpl) SetLastAppliedCR(ctx context.Context, req ctrl.Request, deploymentName string, lastAppliedCr *utils.LastAppliedCr) error {
	foundDeployment, err := r.Get(ctx, req, deploymentName)
	if err != nil || foundDeployment == nil {
		return err
	}
	if lastAppliedCr == nil {
		return nil
	}
	r.logger.InfoC(ctx, "[%v] Mark deleted last applied CR on Deployment %v", req.NamespacedName, deploymentName)

	newLastAppliedCRAnnotation, err := utils.JsonMarshal(lastAppliedCr)
	if err != nil {
		r.logger.ErrorC(ctx, "[%v] Can not marshal last applied cr '%+v'", req.NamespacedName, lastAppliedCr)
		return err
	}

	foundDeployment.Annotations[utils.LastAppliedCRAnnotation] = newLastAppliedCRAnnotation
	return r.update(ctx, req, foundDeployment, foundDeployment)
}

func (r *deploymentClientImpl) DeleteMasterCRLabel(ctx context.Context, req ctrl.Request, deploymentName string) error {
	foundDeployment, err := r.Get(ctx, req, deploymentName)
	if err != nil || foundDeployment == nil {
		return err
	}

	masterCRlabel := foundDeployment.Labels[utils.MasterCR]
	masterCRSpecLabel := foundDeployment.Spec.Template.ObjectMeta.Labels[utils.MasterCR]
	if masterCRlabel == "" && masterCRSpecLabel == "" {
		r.logger.InfoC(ctx, "[%v] Deployment %v have no master CR label", req.NamespacedName, deploymentName)
		return nil
	}

	r.logger.InfoC(ctx, "[%v] Delete master CR label on Deployment %v", req.NamespacedName, deploymentName)
	delete(foundDeployment.ObjectMeta.Labels, utils.MasterCR)
	delete(foundDeployment.Spec.Template.Labels, utils.MasterCR)
	return r.update(ctx, req, foundDeployment, foundDeployment)
}

func (r *deploymentClientImpl) Apply(ctx context.Context, req ctrl.Request, newDeployment *v1.Deployment) error {
	r.logger.InfoC(ctx, "[%v] Apply new deployment %v", req.NamespacedName, newDeployment.Name)
	foundDeployment := &v1.Deployment{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      newDeployment.GetName(),
	}
	err := r.client.Get(ctx, nameSpacedRequest, foundDeployment, &client.GetOptions{})
	found := true
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.InfoC(ctx, "[%v] Deployment %v not found", req.NamespacedName, newDeployment.Name)
			found = false
		} else {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get deployment: %v", newDeployment.GetName()), err)
		}
	}

	if found {
		newDeployment.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(newDeployment.ObjectMeta.OwnerReferences, foundDeployment.ObjectMeta.OwnerReferences)
		return r.update(ctx, req, foundDeployment, newDeployment)
	} else {
		r.logger.InfoC(ctx, "[%v] Create deployment %v", req.NamespacedName, newDeployment.GetName())
		options := &client.CreateOptions{
			FieldManager: "facadeOperator",
		}
		err = r.client.Create(ctx, newDeployment, options)
		if err != nil {
			if k8sErrors.IsAlreadyExists(err) {
				r.logger.Debugf("[%v] Deployment %v already created. Error: %v", req.NamespacedName, newDeployment.GetName(), err)
				return &customerrors.ExpectedError{Message: err.Error()}
			}
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to create deployment: %v", newDeployment.GetName()), err)
		}
	}

	return nil
}

func (r *deploymentClientImpl) update(ctx context.Context, req ctrl.Request, foundDeployment *v1.Deployment, newDeployment *v1.Deployment) error {
	r.logger.InfoC(ctx, "[%v] Update deployment %v", req.NamespacedName, newDeployment.GetName())
	newDeployment.SetResourceVersion(foundDeployment.GetResourceVersion())
	options := &client.UpdateOptions{
		FieldManager: "facadeOperator",
	}
	err := r.client.Update(ctx, newDeployment, options)
	if err != nil {
		if k8sErrors.IsConflict(err) {
			r.logger.Debugf("[%v] Deployment %v already updated. Error: %v", req.NamespacedName, newDeployment.GetName(), err)
			return &customerrors.ExpectedError{Message: err.Error()}
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to update deployment: %v", newDeployment.GetName()), err)
	}

	return nil
}

func (r *deploymentClientImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	r.logger.InfoC(ctx, "[%v] Start delete deployment %v", req.NamespacedName, name)

	foundDeployment, err := r.Get(ctx, req, name)
	if err != nil {
		return err
	}
	if foundDeployment == nil {
		r.logger.InfoC(ctx, "[%v] Deployment %v already deleted", req.NamespacedName, name)
		return nil
	}

	err = r.client.Delete(ctx, foundDeployment)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.Debugf("[%v] Deployment %v already deleted. Error: %v", req.NamespacedName, name, err)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete deployment: %v", name), err)
	}
	r.logger.InfoC(ctx, "[%v] Deployment %v deleted", req.NamespacedName, name)

	return nil
}

func (r *deploymentClientImpl) GetMeshRouterDeployments(ctx context.Context, req ctrl.Request) (*v1.DeploymentList, error) {
	foundDeployments := &v1.DeploymentList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{
			utils.MeshRouter: "true",
		}),
	}

	err := r.client.List(ctx, foundDeployments, opts...)
	if err != nil {
		return nil, errs.NewError(customerrors.UnexpectedKubernetesError, "Failed to get facade gateway deployments", err)
	}

	return foundDeployments, nil
}

func (r *deploymentClientImpl) IsFacadeGateway(ctx context.Context, req ctrl.Request, name string) (bool, error) {
	foundDeployment, err := r.Get(ctx, req, name)
	if err != nil || foundDeployment == nil {
		r.logger.InfoC(ctx, "[%v] Facade gateway %v not found", req.NamespacedName, name)
		return false, err
	}

	isComposite := r.isLabel(foundDeployment, utils.MeshRouter)
	if isComposite {
		r.logger.InfoC(ctx, "[%v] %v is composite gateway", req.NamespacedName, name)
		return false, nil
	}

	isFacade := r.isLabel(foundDeployment, utils.FacadeGateway)
	if !isFacade {
		r.logger.InfoC(ctx, "[%v] %v is not facade gateway", req.NamespacedName, name)
		return false, nil
	}

	return true, nil
}

func (r *deploymentClientImpl) isLabel(foundDeployment *v1.Deployment, labelName string) bool {
	specLabel := foundDeployment.Spec.Template.ObjectMeta.Labels[labelName]
	label := foundDeployment.Labels[labelName]
	return specLabel == "true" || label == "true"
}

func (r *deploymentClientImpl) GetMasterCR(ctx context.Context, req ctrl.Request, name string) (string, error) {
	foundDeployment, err := r.Get(ctx, req, name)
	if err != nil || foundDeployment == nil {
		return "", err
	}

	label := foundDeployment.Labels[utils.MasterCR]
	if label != "" {
		return label, nil
	}
	return foundDeployment.Spec.Template.ObjectMeta.Labels[utils.MasterCR], nil
}
