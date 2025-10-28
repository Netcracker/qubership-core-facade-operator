package controllers

import (
	"context"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	meshGateway1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1"
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

type GatewayReconciler struct {
	base *FacadeCommonReconciler
}

func NewGatewayReconciler(base *FacadeCommonReconciler) *GatewayReconciler {
	return &GatewayReconciler{base: base}
}

//+kubebuilder:rbac:groups=core.netcracker.com,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.netcracker.com,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.netcracker.com,resources=gateways/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupMeshGatewayManager(mgr ctrl.Manager, maxConcurrentReconciles int, client client.Client, deploymentsClient services.DeploymentClient, commonCRClient services.CommonCRClient) error {
	recoverPanic := true
	options := controller.Options{
		MaxConcurrentReconciles: maxConcurrentReconciles,
		RecoverPanic:            &recoverPanic,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&meshGateway1.Gateway{}).
		WithEventFilter(predicates.IgnoreUpdateStatusPredicate).
		WithOptions(options).
		Complete(r)
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctxWithNewRequestId := context.WithValue(ctx, xrequestid.X_REQUEST_ID_COTEXT_NAME, xrequestid.NewXRequestIdContextObject(""))
	r.base.logger.InfoC(ctxWithNewRequestId, "Start processing kind=Gateway apiVersion=core.netcracker.com/v1")
	cr, err := r.getCR(ctxWithNewRequestId, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	return r.base.Reconcile(ctxWithNewRequestId, req, cr)
}

func (r *GatewayReconciler) getCR(ctx context.Context, req ctrl.Request) (facade.MeshGateway, error) {
	cr := &meshGateway1.Gateway{}
	err := r.base.client.Get(ctx, req.NamespacedName, cr, &client.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			r.base.logger.InfoC(ctx, "[%v] mesh gateway CR not found", req.NamespacedName)
			return nil, nil
		}
		return nil, errs.NewError(customerrors.UnexpectedKubernetesError, "Failed to get mesh gateway CR", err)
	}

	return cr, nil
}
