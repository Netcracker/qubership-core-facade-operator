package services

import (
	"context"
	"encoding/json"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusUpdater interface {
	SetUpdating(ctx context.Context, resource facade.MeshGateway) error

	SetUpdated(ctx context.Context, resource facade.MeshGateway) error

	SetFail(ctx context.Context, resource facade.MeshGateway) error
}

type StatusUpdaterImpl struct {
	client client.Client
	log    logging.Logger
}

func NewStatusUpdater(client client.Client) StatusUpdater {
	return &StatusUpdaterImpl{
		client: client,
		log:    logging.GetLogger("StatusUpdater"),
	}
}

func (updater StatusUpdaterImpl) SetUpdating(ctx context.Context, resource facade.MeshGateway) error {
	meshGateway, ok := resource.(*facadeV1.Gateway)
	if !ok {
		return nil
	}
	return updater.patchStatus(ctx, meshGateway, facadeV1.UpdatingPhase)
}

func (updater StatusUpdaterImpl) SetUpdated(ctx context.Context, resource facade.MeshGateway) error {
	meshGateway, ok := resource.(*facadeV1.Gateway)
	if !ok {
		return nil
	}
	return updater.patchStatus(ctx, meshGateway, facadeV1.UpdatedPhase)
}

func (updater StatusUpdaterImpl) SetFail(ctx context.Context, resource facade.MeshGateway) error {
	meshGateway, ok := resource.(*facadeV1.Gateway)
	if !ok {
		return nil
	}
	return updater.patchStatus(ctx, meshGateway, facadeV1.BackingOffPhase)
}

func (updater StatusUpdaterImpl) patchStatus(ctx context.Context, resource *facadeV1.Gateway, phase facadeV1.Phase) error {
	if resource == nil {
		updater.log.InfoC(ctx, "Can not patch status on nil resource")
		return nil
	}
	status := resource.Status
	status.Phase = phase
	status.ObservedGeneration = resource.ObjectMeta.Generation
	resource.Status = status
	return updater.patch(ctx, resource)
}

func (updater StatusUpdaterImpl) patch(ctx context.Context, resource *facadeV1.Gateway) error {
	resourceBuf, err := json.Marshal(resource)
	if err != nil {
		return errs.NewError(customerrors.UnknownErrorCode, "Unknown error while marshal CR", err)
	}
	object := &unstructured.Unstructured{Object: map[string]interface{}{}}
	if err = json.Unmarshal(resourceBuf, object); err != nil {
		return errs.NewError(customerrors.UnknownErrorCode, "Unknown error while unmarshal CR", err)
	}

	mergePatch, err := json.Marshal(map[string]interface{}{
		"status": object.Object["status"],
	})
	if err != nil {
		return errs.NewError(customerrors.UnknownErrorCode, "Unknown error while marshal CR status", err)
	}

	patch := client.RawPatch(types.MergePatchType, mergePatch)

	if err := updater.client.Status().Patch(ctx, resource, patch); err != nil {
		return errs.NewError(customerrors.UnexpectedKubernetesError, "Unknown error while patching CR", err)
	}

	return nil
}
