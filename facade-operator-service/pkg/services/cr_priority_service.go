package services

import (
	"context"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	appsV1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type CRPriorityService interface {
	UpdateAvailable(ctx context.Context, req ctrl.Request, gatewayName string, cr facade.MeshGateway) (bool, error)
}

type crPriorityServiceImpl struct {
	deploymentsClient DeploymentClient
	commonCRClient    CommonCRClient
	logger            logging.Logger
}

func NewCRPriorityService(deploymentsClient DeploymentClient, commonCRClient CommonCRClient) CRPriorityService {
	return &crPriorityServiceImpl{
		deploymentsClient: deploymentsClient,
		commonCRClient:    commonCRClient,
		logger:            logging.GetLogger("CRPriorityService"),
	}
}

func (c *crPriorityServiceImpl) UpdateAvailable(ctx context.Context, req ctrl.Request, gatewayName string, cr facade.MeshGateway) (bool, error) {
	deployment, err := c.deploymentsClient.Get(ctx, req, gatewayName)
	if err != nil {
		return false, err
	}
	if deployment == nil {
		return true, nil
	}
	masterCR := c.getMasterCR(deployment)

	lastAppliedCrAnnotation, err := c.getLastAppliedCR(ctx, deployment)
	if err != nil {
		return false, err
	}
	c.logger.InfoC(ctx, "[%v] Found master CR name '%s' and last applied CR '%+v'", req, masterCR, lastAppliedCrAnnotation)

	foundAppliedCr, err := c.commonCRClient.GetByLastAppliedCr(ctx, req, lastAppliedCrAnnotation)
	if err != nil {
		return false, err
	}

	if cr.GetSpec().MasterConfiguration && masterCR == "" {
		return true, nil
	}

	if masterCR == "" {
		return c.checkByKindPriority(ctx, req, cr, foundAppliedCr), nil
	}

	if masterCR == cr.GetName() && c.isSameType(cr, foundAppliedCr) {
		return c.checkByKindPriority(ctx, req, cr, foundAppliedCr), nil
	}

	c.logger.InfoC(ctx, "[%v] Try to find master CR %v for gateway %v", req.NamespacedName, masterCR, gatewayName)
	foundMasterCR, err := c.commonCRClient.IsCRExistByName(ctx, req, masterCR)
	if err != nil {
		return false, err
	}
	if !foundMasterCR {
		c.logger.InfoC(ctx, "[%v] Master CR label %v exists on deployment %v, but CR not found", req.NamespacedName, masterCR, gatewayName)
		return c.checkByKindPriority(ctx, req, cr, foundAppliedCr), nil
	}

	if !cr.GetSpec().MasterConfiguration {
		c.logger.InfoC(ctx, "[%v] Non master configuration CR [%v] can not update deployment [%v] with master CR [%v]",
			req.NamespacedName,
			cr.GetName(),
			gatewayName,
			masterCR)
		return false, nil
	}

	return c.checkMasterCRByKindPriority(ctx, req, cr, foundAppliedCr), nil
}

func (c *crPriorityServiceImpl) isSameType(cr facade.MeshGateway, lastAppliedCr facade.MeshGateway) bool {
	if lastAppliedCr == nil {
		return true
	}
	return lastAppliedCr.GetKind() == cr.GetKind() && lastAppliedCr.GetAPIVersion() == cr.GetAPIVersion()
}

func (c *crPriorityServiceImpl) checkByKindPriority(ctx context.Context, req ctrl.Request, cr facade.MeshGateway, lastAppliedCr facade.MeshGateway) bool {
	if lastAppliedCr != nil && cr.Priority() < lastAppliedCr.Priority() {
		c.logger.InfoC(ctx, "[%v] Current CR has lower priority '%d' than the last applied one '%d'", req.NamespacedName, cr.Priority(), lastAppliedCr.Priority)
		return false
	}

	return true
}

func (c *crPriorityServiceImpl) checkMasterCRByKindPriority(ctx context.Context, req ctrl.Request, cr facade.MeshGateway, lastAppliedCr facade.MeshGateway) bool {
	if lastAppliedCr != nil && cr.Priority() <= lastAppliedCr.Priority() {
		c.logger.InfoC(ctx, "[%v] Another master CR has equals or lower priority '%d' than the last applied one '%d'", req.NamespacedName, cr.Priority(), lastAppliedCr.Priority)
		return false
	}

	return true
}

func (c *crPriorityServiceImpl) getMasterCR(deployment *appsV1.Deployment) string {
	label := deployment.Labels[utils.MasterCR]
	if label != "" {
		return label
	}
	return deployment.Spec.Template.ObjectMeta.Labels[utils.MasterCR]
}

func (c *crPriorityServiceImpl) getLastAppliedCR(ctx context.Context, deployment *appsV1.Deployment) (*utils.LastAppliedCr, error) {
	lastAppliedCRAnnotation := deployment.Annotations[utils.LastAppliedCRAnnotation]
	lastAppliedCr, err := utils.JsonUnmarshal[utils.LastAppliedCr](lastAppliedCRAnnotation)
	if err != nil {
		c.logger.ErrorC(ctx, "Can not unmarshal '%s' annotation with value: %s", utils.LastAppliedCRAnnotation, lastAppliedCRAnnotation)
		return nil, err
	}
	return lastAppliedCr, nil
}
