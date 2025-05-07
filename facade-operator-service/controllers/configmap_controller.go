package controllers

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/predicates"
	"github.com/netcracker/qubership-core-facade-operator/pkg/services"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ConfigMapReconciller struct {
	client          client.Client
	logger          logging.Logger
	configMapClient services.ConfigMapClient
}

func NewConfigMapReconciller(client client.Client, configMapClient services.ConfigMapClient) *ConfigMapReconciller {
	return &ConfigMapReconciller{client: client, logger: logging.GetLogger("ConfigMapReconciler"), configMapClient: configMapClient}
}

func (r *ConfigMapReconciller) SetupWithManager(mgr ctrl.Manager) error {
	recoverPanic := true
	options := controller.Options{
		MaxConcurrentReconciles: 1,
		RecoverPanic:            &recoverPanic,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(predicates.ObjectNamePredicate{Name: services.CoreGatewayImageConfigMap})).
		WithOptions(options).
		Complete(r)
}

func (r *ConfigMapReconciller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	result, err := r.reconcile(ctx, req)
	if err == nil {
		return result, err
	}

	switch e := err.(type) {
	case *customerrors.ExpectedError:
		// It is necessary to mute the errors associated with the race
		// For example, when creating multiple CRs at the same time with 1 composite gateway
		return ctrl.Result{Requeue: true}, nil
	case *errs.ErrCodeError:
		r.logger.ErrorC(ctx, "[%v] %v", req.NamespacedName, errs.ToLogFormat(e))
		return result, e
	default:
		return result, errs.NewError(customerrors.UnknownErrorCode, "Unknown error", err)
	}
}

func (r *ConfigMapReconciller) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.InfoC(ctx, "[%v] Start sync images", req.NamespacedName)

	gatewayImage, err := r.configMapClient.GetGatewayImage(ctx, req)
	if err != nil {
		return ctrl.Result{}, errs.NewError(customerrors.GatewayImageError, "Failed to get image", err)
	}

	facadeDeployments := &v1.DeploymentList{}
	opts := r.getListOption(req)
	err = r.client.List(ctx, facadeDeployments, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			r.logger.Debugf("[%v] Facade gateways not found", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errs.NewError(customerrors.UpdateImageUnexpectedKubernetesError, "Failed to get facade gateway deployments", err)
	}

	isConflictError, err := r.updateImages(ctx, req, facadeDeployments, gatewayImage)
	if err != nil {
		return ctrl.Result{}, err
	}

	if isConflictError {
		r.logger.Debugf("[%v] Conflict error found", req.NamespacedName)
		return ctrl.Result{Requeue: true}, &customerrors.ExpectedError{}
	}

	r.logger.InfoC(ctx, "[%v] Done sync images", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *ConfigMapReconciller) updateImages(ctx context.Context, req ctrl.Request, facadeDeployments *v1.DeploymentList, gatewayImage string) (bool, error) {
	isConflictError := false
	for _, facadeDeployment := range facadeDeployments.Items {
		facadeImage := facadeDeployment.Spec.Template.Spec.Containers[0].Image
		if facadeImage == gatewayImage {
			continue
		}

		r.logger.InfoC(ctx, "[%v] Update image %v. Old image: %v. New image: %v", req.NamespacedName, facadeDeployment.GetName(), facadeImage, gatewayImage)
		facadeDeployment.Spec.Template.Spec.Containers[0].Image = gatewayImage
		err := r.client.Update(ctx, &facadeDeployment)
		if err != nil {
			if k8sErrors.IsConflict(err) {
				r.logger.Debugf("[%v] Can not update image on %v. Error: %v", req.NamespacedName, facadeDeployment.Name, err)
				isConflictError = true
				continue
			}
			return false, errs.NewError(customerrors.UpdateImageUnexpectedKubernetesError, fmt.Sprintf("Failed to update Deployment: %v", facadeDeployment.Name), err)
		}
	}

	return isConflictError, nil
}

func (r *ConfigMapReconciller) getListOption(req ctrl.Request) []client.ListOption {
	return []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{
			utils.FacadeGateway: "true",
		}),
	}
}
