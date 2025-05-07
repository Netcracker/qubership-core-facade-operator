package services

import (
	"context"
	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/api/facade"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/api/facade/v1"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/api/facade/v1alpha"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CommonCRClient interface {
	FindByNames(ctx context.Context, req ctrl.Request, names []string) ([]facade.MeshGateway, error)
	FindByFields(ctx context.Context, req ctrl.Request, fields client.MatchingFields) ([]facade.MeshGateway, error)
	GetAll(ctx context.Context, req ctrl.Request) ([]facade.MeshGateway, error)
	IsCRExistByName(ctx context.Context, req ctrl.Request, name string) (bool, error)
	GetByLastAppliedCr(ctx context.Context, req ctrl.Request, lastCr *utils.LastAppliedCr) (facade.MeshGateway, error)
}

type commonCRClientImpl struct {
	client client.Client
	logger logging.Logger
}

func NewCommonCRClient(client client.Client) *commonCRClientImpl {
	return &commonCRClientImpl{
		client: client,
		logger: logging.GetLogger("CommonCRClient"),
	}
}

func (c *commonCRClientImpl) FindByNames(ctx context.Context, req ctrl.Request, names []string) ([]facade.MeshGateway, error) {
	result := make([]facade.MeshGateway, 0)
	for _, name := range names {
		nameSpacedRequest := types.NamespacedName{
			Namespace: req.Namespace,
			Name:      name,
		}
		facadeCR, err := c.getByNameSpacedRequest(ctx, req, nameSpacedRequest, &facadeV1Alpha.FacadeService{})
		if err != nil {
			return nil, err
		}
		if facadeCR != nil {
			result = append(result, facadeCR.(*facadeV1Alpha.FacadeService))
		}

		meshCR, err := c.getByNameSpacedRequest(ctx, req, nameSpacedRequest, &facadeV1.Gateway{})
		if err != nil {
			return nil, err
		}
		if meshCR != nil {
			result = append(result, meshCR.(*facadeV1.Gateway))
		}
	}
	return result, nil
}

func (c *commonCRClientImpl) FindByFields(ctx context.Context, req ctrl.Request, fields client.MatchingFields) ([]facade.MeshGateway, error) {
	result := make([]facade.MeshGateway, 0)

	facadeServiceList := &facadeV1Alpha.FacadeServiceList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
		fields,
	}
	err := c.client.List(ctx, facadeServiceList, opts...)
	if err != nil {
		return nil, err
	}
	result = c.appendFacade(result, facadeServiceList)

	meshGatewayList := &facadeV1.GatewayList{}
	err = c.client.List(ctx, meshGatewayList, opts...)
	if err != nil {
		return nil, err
	}
	result = c.appendMeshGateway(result, meshGatewayList)

	return result, nil
}

func (c *commonCRClientImpl) GetAll(ctx context.Context, req ctrl.Request) ([]facade.MeshGateway, error) {
	allCRs := make([]facade.MeshGateway, 0)
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}

	facadeServiceList := &facadeV1Alpha.FacadeServiceList{}
	err := c.client.List(ctx, facadeServiceList, opts...)
	if err != nil || facadeServiceList == nil {
		return nil, errs.NewError(customerrors.UnexpectedKubernetesError, "Failed to get facade service list", err)
	}
	for _, item := range facadeServiceList.Items {
		allCRs = append(allCRs, &item)
	}

	meshGatewayList := &facadeV1.GatewayList{}
	err = c.client.List(ctx, meshGatewayList, opts...)
	if err != nil || meshGatewayList == nil {
		return nil, errs.NewError(customerrors.UnexpectedKubernetesError, "Failed to get mesh gateway list", err)
	}
	for _, item := range meshGatewayList.Items {
		allCRs = append(allCRs, &item)
	}

	return allCRs, nil
}

func (c *commonCRClientImpl) IsCRExistByName(ctx context.Context, req ctrl.Request, name string) (bool, error) {
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}

	found, err := c.isCRExist(ctx, req, nameSpacedRequest, &facadeV1Alpha.FacadeService{})
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	found, err = c.isCRExist(ctx, req, nameSpacedRequest, &facadeV1.Gateway{})
	if err != nil {
		return false, err
	}
	return found, nil
}

func (c *commonCRClientImpl) isCRExist(ctx context.Context, req ctrl.Request, nameSpacedRequest types.NamespacedName, object client.Object) (bool, error) {
	foundObject, err := c.getByNameSpacedRequest(ctx, req, nameSpacedRequest, object)
	if err != nil {
		return false, err
	}

	return foundObject != nil, nil
}

func (c *commonCRClientImpl) GetByLastAppliedCr(ctx context.Context, req ctrl.Request, lastCr *utils.LastAppliedCr) (facade.MeshGateway, error) {
	c.logger.InfoC(ctx, "[%v] Try to find CR by last applied cr. %+v", req.NamespacedName, lastCr)
	if lastCr == nil {
		c.logger.InfoC(ctx, "[%v] Can not found CR by nil last applied cr", req.NamespacedName)
		return nil, nil
	}
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      lastCr.Name,
	}
	object, err := lastCr.ResolveType()
	if err != nil {
		c.logger.ErrorC(ctx, "[%v] Can not resolve CR type. %s", req.NamespacedName, err.Error())
		return nil, err
	}
	cr, err := c.getByNameSpacedRequest(ctx, req, nameSpacedRequest, object.(client.Object))
	if err != nil {
		return nil, err
	}
	if cr == nil {
		c.logger.InfoC(ctx, "[%v] CR not found by last applied cr. %+v", req.NamespacedName, lastCr)
		return nil, nil
	}

	return cr.(facade.MeshGateway), nil
}

func (c *commonCRClientImpl) getByNameSpacedRequest(ctx context.Context, req ctrl.Request, nameSpacedRequest types.NamespacedName, object client.Object) (client.Object, error) {
	err := c.client.Get(ctx, nameSpacedRequest, object, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			c.logger.InfoC(ctx, "[%v] %v with name %v not found", req.NamespacedName, object.GetObjectKind(), nameSpacedRequest.Name)
			return nil, nil
		} else {
			return nil, errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get %v with name %v", object.GetObjectKind(), nameSpacedRequest.Name), err)
		}
	}

	return object, nil
}

func (c *commonCRClientImpl) appendFacade(resultSlice []facade.MeshGateway, facadeList *facadeV1Alpha.FacadeServiceList) []facade.MeshGateway {
	if len(facadeList.Items) == 0 {
		return resultSlice
	}
	for _, item := range facadeList.Items {
		resultSlice = append(resultSlice, &item)
	}

	return resultSlice
}

func (c *commonCRClientImpl) appendMeshGateway(resultSlice []facade.MeshGateway, meshGatewayList *facadeV1.GatewayList) []facade.MeshGateway {
	if len(meshGatewayList.Items) == 0 {
		return resultSlice
	}
	for _, item := range meshGatewayList.Items {
		resultSlice = append(resultSlice, &item)
	}

	return resultSlice
}
