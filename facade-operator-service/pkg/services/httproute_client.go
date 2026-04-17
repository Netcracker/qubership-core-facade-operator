package services

import (
	"context"
	"fmt"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type HTTPRouteClient interface {
	Apply(ctx context.Context, req ctrl.Request, httpRoute templates.HTTPRoute) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
	DeleteOrphaned(ctx context.Context, req ctrl.Request) error
}

type HTTPRouteClientImpl struct {
	GenericClient[*gatewayv1.HTTPRoute]
	ingressBuilder *templates.IngressTemplateBuilder
	commonCRClient CommonCRClient
	logger         logging.Logger
}

func NewHTTPRouteClient(client client.Client, ingressBuilder *templates.IngressTemplateBuilder, commonCRClient CommonCRClient) *HTTPRouteClientImpl {
	return &HTTPRouteClientImpl{
		GenericClient:   NewGenericClient[*gatewayv1.HTTPRoute](client, "HTTPRoute"),
		ingressBuilder:  ingressBuilder,
		commonCRClient:  commonCRClient,
		logger:          logging.GetLogger("gatewayV1.HTTPRoute client"),
	}
}

func (h *HTTPRouteClientImpl) Apply(ctx context.Context, req ctrl.Request, httpRoute templates.HTTPRoute) error {
	return h.GenericClient.Apply(ctx, req, &gatewayv1.HTTPRoute{}, httpRoute.BuildK8sHTTPRoute(), h.mergeHTTPRoutes)
}

func (h *HTTPRouteClientImpl) mergeHTTPRoutes(existingResReceiver, newObject *gatewayv1.HTTPRoute) {
	newObject.ObjectMeta.ResourceVersion = existingResReceiver.ResourceVersion
	newObject.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(newObject.ObjectMeta.OwnerReferences, existingResReceiver.ObjectMeta.OwnerReferences)
}

func (h *HTTPRouteClientImpl) DeleteOrphaned(ctx context.Context, req ctrl.Request) error {
	httpRoutes := &gatewayv1.HTTPRouteList{}
	err := h.GetClient().List(ctx, httpRoutes, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace))
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, "Failed to get httproutes managed by facade-operator", err)
	}

	h.logger.DebugC(ctx, "Found %d httproutes managed by facade-operator", len(httpRoutes.Items))
	if len(httpRoutes.Items) == 0 {
		return nil
	}

	httpRouteNamesSet, err := collectHTTPRouteNamesFromFacadeServices(ctx, req, h.ingressBuilder, h.commonCRClient)
	if err != nil {
		return err
	}

	for _, httpRoute := range httpRoutes.Items {
		if !httpRouteNamesSet[httpRoute.Name] {
			if err = h.Delete(ctx, req, httpRoute.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *HTTPRouteClientImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	return h.GenericClient.Delete(ctx, req, name, &gatewayv1.HTTPRoute{})
}

func collectHTTPRouteNamesFromFacadeServices(ctx context.Context, req ctrl.Request, ingressBuilder *templates.IngressTemplateBuilder, commonCRClient CommonCRClient) (map[string]bool, error) {
	crs, err := commonCRClient.FindByFields(ctx, req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)})
	if err != nil || len(crs) == 0 {
		return nil, err
	}

	httpRouteNamesSet := make(map[string]bool)
	for _, cr := range crs {
		for _, ingressSpec := range cr.GetSpec().Ingresses {
			httpRouteName, _, err := ingressBuilder.BuildNameAndPort(ingressSpec, cr, utils.ResolveGatewayServiceName(cr.GetName(), cr))
			if err != nil {
				return nil, errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("failed to build HTTPRoute name for CR %s", cr.GetName()), err)
			}
			httpRouteNamesSet[httpRouteName] = true
		}
	}
	return httpRouteNamesSet, nil
}
