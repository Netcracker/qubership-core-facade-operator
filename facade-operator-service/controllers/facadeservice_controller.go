package controllers

import (
	"context"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/predicates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/services"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/context-propagation/baseproviders/xrequestid"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type FacadeServiceReconciler struct {
	base *FacadeCommonReconciler
}

func NewFacadeServiceReconciler(base *FacadeCommonReconciler) *FacadeServiceReconciler {
	return &FacadeServiceReconciler{base: base}
}

//+kubebuilder:rbac:groups=netcracker.com,resources=facadeservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netcracker.com,resources=facadeservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netcracker.com,resources=facadeservices/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *FacadeServiceReconciler) SetupFacadeServiceManager(mgr ctrl.Manager, maxConcurrentReconciles int, client client.Client, deploymentsClient services.DeploymentClient, commonCRClient services.CommonCRClient) error {
	recoverPanic := true
	options := controller.Options{
		MaxConcurrentReconciles: maxConcurrentReconciles,
		RecoverPanic:            &recoverPanic,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&facadeV1Alpha.FacadeService{}).
		WithEventFilter(predicates.IgnoreUpdateStatusPredicate).
		WithOptions(options).
		Complete(r)
}

func (r *FacadeServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctxWithNewRequestId := context.WithValue(ctx, xrequestid.X_REQUEST_ID_COTEXT_NAME, xrequestid.NewXRequestIdContextObject(""))
	r.base.logger.InfoC(ctxWithNewRequestId, "Start processing kind=FacadeService apiVersion=netcracker.com/v1alpha")
	cr, err := r.getCR(ctxWithNewRequestId, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	return r.base.Reconcile(ctxWithNewRequestId, req, cr)
}

func (r *FacadeServiceReconciler) getCR(ctx context.Context, req ctrl.Request) (facade.MeshGateway, error) {
	cr := &facadeV1Alpha.FacadeService{}
	err := r.base.client.Get(ctx, req.NamespacedName, cr, &client.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			r.base.logger.InfoC(ctx, "[%v] mesh gateway CR not found", req.NamespacedName)
			return nil, nil
		}
		return nil, errs.NewError(customerrors.UnexpectedKubernetesError, "Failed to get facade CR", err)
	}

	return cr, nil
}
