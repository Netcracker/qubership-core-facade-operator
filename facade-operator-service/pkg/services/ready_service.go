package services

import (
	"context"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

type ReadyService interface {
	CheckDeploymentReady(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) (ctrl.Result, error)
	IsUpdatingPhase(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) bool
}

type ReadyServiceImpl struct {
	deploymentsClient DeploymentClient
	statusUpdater     StatusUpdater
	log               logging.Logger
}

func NewReadyService(deploymentsClient DeploymentClient, statusUpdater StatusUpdater) ReadyService {
	return &ReadyServiceImpl{
		deploymentsClient: deploymentsClient,
		statusUpdater:     statusUpdater,
		log:               logging.GetLogger("ReadyService"),
	}
}

func (r *ReadyServiceImpl) IsUpdatingPhase(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) bool {
	meshGateway, ok := cr.(*facadeV1.Gateway)
	if !ok {
		return false
	}
	crGeneration := meshGateway.ObjectMeta.Generation
	statusGeneration := meshGateway.Status.ObservedGeneration
	if crGeneration != statusGeneration {
		r.log.InfoC(ctx, "[%v] CR status generation '%d', object generation '%d'", req.NamespacedName, statusGeneration, crGeneration)
		return false
	}
	phase := meshGateway.Status.Phase
	r.log.InfoC(ctx, "[%v] Current phase %v", req.NamespacedName, phase)
	return facadeV1.UpdatingPhase == phase
}

func (r *ReadyServiceImpl) CheckDeploymentReady(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) (ctrl.Result, error) {
	meshGateway, ok := cr.(*facadeV1.Gateway)
	if !ok {
		return ctrl.Result{}, nil
	}
	r.log.InfoC(ctx, "[%v] Wait ready status", req.NamespacedName)
	ready, err := r.isDeploymentReadyInternal(ctx, req, meshGateway)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !ready {
		r.log.InfoC(ctx, "[%v] Deployment still not ready", req.NamespacedName)
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}

	r.log.InfoC(ctx, "[%v] Status ready", req.NamespacedName)
	if statusErr := r.statusUpdater.SetUpdated(ctx, cr); statusErr != nil {
		r.log.ErrorC(ctx, "[%v] Can not update status on CR. Error: %s", req.NamespacedName, statusErr.Error())
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, nil
}

func (r *ReadyServiceImpl) isDeploymentReadyInternal(ctx context.Context, req ctrl.Request, cr *facadeV1.Gateway) (bool, error) {
	deploymentName := cr.ObjectMeta.Name + utils.GatewaySuffix
	if cr.Spec.Gateway != "null" && cr.Spec.Gateway != "" {
		deploymentName = cr.Spec.Gateway
	}

	foundDeployment, err := r.deploymentsClient.Get(ctx, req, deploymentName)
	if err != nil {
		return false, err
	}
	if foundDeployment == nil {
		r.log.WarnC(ctx, "[%v] Deployment '%s' not found. Can not check ready status", req.NamespacedName, deploymentName)
		return false, nil
	}
	r.log.InfoC(ctx, "[%v] Found deployment '%s'", req.NamespacedName, deploymentName)

	replicas := foundDeployment.Status.Replicas
	readyReplicas := foundDeployment.Status.ReadyReplicas
	unavailableReplicas := foundDeployment.Status.UnavailableReplicas

	r.log.InfoC(ctx, "[%v] Deployment replicas: '%d', readyReplicas: '%d', unavailableReplicas: '%d'", req.NamespacedName, replicas, readyReplicas, unavailableReplicas)
	if unavailableReplicas != 0 {
		return false, nil
	}

	return replicas == readyReplicas, nil
}
