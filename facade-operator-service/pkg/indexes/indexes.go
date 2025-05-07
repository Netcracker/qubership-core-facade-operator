package indexes

import (
	"context"
	"fmt"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/api/facade/v1"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/api/facade/v1alpha"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"

	openshiftv1 "github.com/openshift/api/route/v1"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ManagedByFieldName = "metadata.annotations.app.kubernetes.io/managed-by"
const ManagedByAnnotationName = "app.kubernetes.io/managed-by"
const SpecGatewayTypeFieldName = "spec.gatewayType"

func IndexFields(ctx context.Context, indexer cache.Cache) error {
	if utils.GetPlatform() == utils.Kubernetes {
		// ingress managed-by field index

		if utils.GetVersion().IsNewerThanOrEqual(utils.SemVer{Major: 1, Minor: 22}) {
			if err := indexer.IndexField(ctx, &v1.Ingress{}, ManagedByFieldName, func(object client.Object) []string {
				ingress := object.(*v1.Ingress)
				val := ingress.GetAnnotations()[ManagedByAnnotationName]
				return []string{val}
			}); err != nil {
				return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("could not index v1.Ingress field %s", ManagedByFieldName), err)
			}
		} else {
			if err := indexer.IndexField(ctx, &v1beta1.Ingress{}, ManagedByFieldName, func(object client.Object) []string {
				ingress := object.(*v1beta1.Ingress)
				val := ingress.GetAnnotations()[ManagedByAnnotationName]
				return []string{val}
			}); err != nil {
				return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("could not index v1beta1.Ingress field %s", ManagedByFieldName), err)
			}
		}
	} else {
		if err := indexer.IndexField(ctx, &openshiftv1.Route{}, ManagedByFieldName, func(object client.Object) []string {
			ingress := object.(*openshiftv1.Route)
			val := ingress.GetAnnotations()[ManagedByAnnotationName]
			return []string{val}
		}); err != nil {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("could not index openshiftv1.Route field %s", ManagedByFieldName), err)
		}
	}

	// facadeService gatewayType field index
	if err := indexer.IndexField(ctx, &facadeV1Alpha.FacadeService{}, SpecGatewayTypeFieldName, func(object client.Object) []string {
		facadeService := object.(*facadeV1Alpha.FacadeService)
		return []string{string(facadeService.Spec.GatewayType)}
	}); err != nil {
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("could not index FacadeService field %s", SpecGatewayTypeFieldName), err)
	}

	// facadeService gateway field index
	facadeSpecGatewayIndexFunc := func(obj client.Object) []string {
		return []string{obj.(*facadeV1Alpha.FacadeService).Spec.Gateway}
	}
	if err := indexer.IndexField(ctx, &facadeV1Alpha.FacadeService{}, utils.SpecGatewayField, facadeSpecGatewayIndexFunc); err != nil {
		panic(err)
	}

	// meshgateway gatewayType field index
	if err := indexer.IndexField(ctx, &facadeV1.Gateway{}, SpecGatewayTypeFieldName, func(object client.Object) []string {
		meshGateway := object.(*facadeV1.Gateway)
		return []string{string(meshGateway.Spec.GatewayType)}
	}); err != nil {
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("could not index FacadeService field %s", SpecGatewayTypeFieldName), err)
	}

	// meshgateway gateway field index
	meshSpecGatewayIndexFunc := func(obj client.Object) []string {
		return []string{obj.(*facadeV1.Gateway).Spec.Gateway}
	}

	if err := indexer.IndexField(ctx, &facadeV1.Gateway{}, utils.SpecGatewayField, meshSpecGatewayIndexFunc); err != nil {
		panic(err)
	}

	return nil
}
