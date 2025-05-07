package services

import (
	"context"
	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/api/facade"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	openshiftv1 "github.com/openshift/api/route/v1"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngressClientAggregator interface {
	Apply(ctx context.Context, req ctrl.Request, ingress templates.Ingress) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
	DeleteOrphaned(ctx context.Context, req ctrl.Request) error
}

type AggregatorImpl struct {
	ingressBuilder    *templates.IngressTemplateBuilder
	ingressClient     IngressClient
	ingressBetaClient IngressBetaClient
	routeClient       RouteClient
}

func NewIngressClientAggregator(client client.Client, ingressBuilder *templates.IngressTemplateBuilder, commonCRClient CommonCRClient) *AggregatorImpl {
	return &AggregatorImpl{
		ingressBuilder:    ingressBuilder,
		ingressClient:     NewIngressClient(client, ingressBuilder, commonCRClient),
		ingressBetaClient: NewIngressBetaClient(client, ingressBuilder, commonCRClient),
		routeClient:       NewRouteClient(client, ingressBuilder, commonCRClient),
	}
}

func (c *AggregatorImpl) Apply(ctx context.Context, req ctrl.Request, ingress templates.Ingress) error {
	if utils.GetPlatform() == utils.Kubernetes {
		if utils.GetVersion().IsNewerThanOrEqual(utils.SemVer{Major: 1, Minor: 22, Patch: 0}) {
			return c.ingressClient.Apply(ctx, req, ingress.BuildK8sIngress())
		} else {
			return c.ingressBetaClient.Apply(ctx, req, ingress.BuildK8sBetaIngress())
		}
	} else {
		return c.routeClient.Apply(ctx, req, ingress.BuildOpenshiftRoute())
	}
}

func (c *AggregatorImpl) DeleteOrphaned(ctx context.Context, req ctrl.Request) error {
	if utils.GetPlatform() == utils.Kubernetes {
		if utils.GetVersion().IsNewerThanOrEqual(utils.SemVer{Major: 1, Minor: 22, Patch: 0}) {
			return c.ingressClient.DeleteOrphaned(ctx, req)
		} else {
			return c.ingressBetaClient.DeleteOrphaned(ctx, req)
		}
	} else {
		return c.routeClient.DeleteOrphaned(ctx, req)
	}
}

func (c *AggregatorImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	if utils.GetPlatform() == utils.Kubernetes {
		if utils.GetVersion().IsNewerThanOrEqual(utils.SemVer{Major: 1, Minor: 22, Patch: 0}) {
			return c.ingressClient.Delete(ctx, req, name)
		} else {
			return c.ingressBetaClient.Delete(ctx, req, name)
		}
	} else {
		return c.routeClient.Delete(ctx, req, name)
	}
}

type IngressClient interface {
	Apply(ctx context.Context, req ctrl.Request, ingress *v1.Ingress) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
	DeleteOrphaned(ctx context.Context, req ctrl.Request) error
}

type IngressClientImpl struct {
	GenericClient[*v1.Ingress]
	ingressBuilder *templates.IngressTemplateBuilder
	commonCRClient CommonCRClient
	logger         logging.Logger
}

func NewIngressClient(client client.Client, ingressBuilder *templates.IngressTemplateBuilder, commonCRClient CommonCRClient) *IngressClientImpl {
	return &IngressClientImpl{NewGenericClient[*v1.Ingress](client, "Ingress"), ingressBuilder, commonCRClient, logging.GetLogger("v1.Ingress client")}
}

func (i *IngressClientImpl) Apply(ctx context.Context, req ctrl.Request, ingress *v1.Ingress) error {
	return i.GenericClient.Apply(ctx, req, &v1.Ingress{}, ingress, i.mergeIngresses)
}

func (i *IngressClientImpl) mergeIngresses(existingResReceiver, newObject *v1.Ingress) {
	newObject.ObjectMeta.ResourceVersion = existingResReceiver.ResourceVersion
	newObject.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(newObject.ObjectMeta.OwnerReferences, existingResReceiver.ObjectMeta.OwnerReferences)
}

func (i *IngressClientImpl) DeleteOrphaned(ctx context.Context, req ctrl.Request) error {
	// load all ingresses managed by facade-operator
	ingresses := &v1.IngressList{}
	err := i.GetClient().List(ctx, ingresses, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace))
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get ingresses managed by facade-operator"), err)
	}

	i.logger.DebugC(ctx, "Found %d ingresses managed by facade-operator", len(ingresses.Items))

	if len(ingresses.Items) == 0 {
		return nil
	}

	ingressNamesSet, err := collectIngressNamesFromFacadeServices(ctx, req, i.GetClient(), i.ingressBuilder, i.commonCRClient)
	if err != nil {
		return err
	}

	// delete orphaned ingresses
	for _, ingress := range ingresses.Items {
		if !ingressNamesSet[ingress.Name] { // no CR refers to this ingress, so we need to delete it
			if err = i.Delete(ctx, req, ingress.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *IngressClientImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	return i.GenericClient.Delete(ctx, req, name, &v1.Ingress{})
}

type IngressBetaClient interface {
	Apply(ctx context.Context, req ctrl.Request, ingress *v1beta1.Ingress) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
	DeleteOrphaned(ctx context.Context, req ctrl.Request) error
}

type IngressBetaClientImpl struct {
	GenericClient[*v1beta1.Ingress]
	ingressBuilder *templates.IngressTemplateBuilder
	commonCRClient CommonCRClient
	logger         logging.Logger
}

func NewIngressBetaClient(client client.Client, ingressBuilder *templates.IngressTemplateBuilder, commonCRClient CommonCRClient) *IngressBetaClientImpl {
	return &IngressBetaClientImpl{NewGenericClient[*v1beta1.Ingress](client, "Ingress"), ingressBuilder, commonCRClient, logging.GetLogger("v1beta1.Ingress client")}
}

func (i *IngressBetaClientImpl) Apply(ctx context.Context, req ctrl.Request, ingress *v1beta1.Ingress) error {
	return i.GenericClient.Apply(ctx, req, &v1beta1.Ingress{}, ingress, i.mergeIngresses)
}

func (i *IngressBetaClientImpl) mergeIngresses(existingResReceiver, newObject *v1beta1.Ingress) {
	newObject.ObjectMeta.ResourceVersion = existingResReceiver.ResourceVersion
	newObject.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(newObject.ObjectMeta.OwnerReferences, existingResReceiver.ObjectMeta.OwnerReferences)
}

func (i *IngressBetaClientImpl) DeleteOrphaned(ctx context.Context, req ctrl.Request) error {
	// load all ingresses managed by facade-operator
	ingresses := &v1beta1.IngressList{}
	err := i.GetClient().List(ctx, ingresses, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace))
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get ingresses managed by facade-operator"), err)
	}

	i.logger.DebugC(ctx, "Found %d ingresses managed by facade-operator", len(ingresses.Items))

	if len(ingresses.Items) == 0 {
		return nil
	}

	ingressNamesSet, err := collectIngressNamesFromFacadeServices(ctx, req, i.GetClient(), i.ingressBuilder, i.commonCRClient)
	if err != nil {
		return err
	}

	// delete orphaned ingresses
	for _, ingress := range ingresses.Items {
		if !ingressNamesSet[ingress.Name] { // no CR refers to this ingress, so we need to delete it
			if err = i.Delete(ctx, req, ingress.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *IngressBetaClientImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	return i.GenericClient.Delete(ctx, req, name, &v1beta1.Ingress{})
}

type RouteClient interface {
	Apply(ctx context.Context, req ctrl.Request, ingress *openshiftv1.Route) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
	DeleteOrphaned(ctx context.Context, req ctrl.Request) error
}

type RouteClientImpl struct {
	GenericClient[*openshiftv1.Route]
	ingressBuilder *templates.IngressTemplateBuilder
	commonCRClient CommonCRClient
	logger         logging.Logger
}

func NewRouteClient(client client.Client, ingressBuilder *templates.IngressTemplateBuilder, commonCRClient CommonCRClient) *RouteClientImpl {
	return &RouteClientImpl{NewGenericClient[*openshiftv1.Route](client, "Route"), ingressBuilder, commonCRClient, logging.GetLogger("openshiftV1.Route client")}
}

func (i *RouteClientImpl) Apply(ctx context.Context, req ctrl.Request, ingress *openshiftv1.Route) error {
	return i.GenericClient.Apply(ctx, req, &openshiftv1.Route{}, ingress, i.mergeIngresses)
}

func (i *RouteClientImpl) mergeIngresses(existingResReceiver, newObject *openshiftv1.Route) {
	newObject.ObjectMeta.ResourceVersion = existingResReceiver.ResourceVersion
	newObject.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(newObject.ObjectMeta.OwnerReferences, existingResReceiver.ObjectMeta.OwnerReferences)
}

func (i *RouteClientImpl) DeleteOrphaned(ctx context.Context, req ctrl.Request) error {
	// load all ingresses managed by facade-operator
	ingresses := &openshiftv1.RouteList{}
	err := i.GetClient().List(ctx, ingresses, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace))
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get ingresses managed by facade-operator"), err)
	}

	i.logger.DebugC(ctx, "Found %d openshift routes managed by facade-operator", len(ingresses.Items))

	if len(ingresses.Items) == 0 {
		return nil
	}

	ingressNamesSet, err := collectIngressNamesFromFacadeServices(ctx, req, i.GetClient(), i.ingressBuilder, i.commonCRClient)
	if err != nil {
		return err
	}

	// delete orphaned ingresses
	for _, ingress := range ingresses.Items {
		if !ingressNamesSet[ingress.Name] { // no CR refers to this ingress, so we need to delete it
			if err = i.Delete(ctx, req, ingress.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *RouteClientImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	return i.GenericClient.Delete(ctx, req, name, &openshiftv1.Route{})
}

func collectIngressNamesFromFacadeServices(ctx context.Context, req ctrl.Request, k8sClient client.Client, ingressBuilder *templates.IngressTemplateBuilder, commonCRClient CommonCRClient) (map[string]bool, error) {
	crs, err := commonCRClient.FindByFields(ctx, req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)})
	if err != nil || len(crs) == 0 {
		return nil, err
	}

	// collect all names of the ingresses declared by existing facadeServices
	ingressNamesSet := make(map[string]bool)
	for _, cr := range crs {
		for _, ingresSpec := range cr.GetSpec().Ingresses {
			ingressName, _, err := ingressBuilder.BuildNameAndPort(ingresSpec, cr, utils.ResolveGatewayServiceName(cr.GetName(), cr))
			if err != nil {
				return nil, err
			}
			ingressNamesSet[ingressName] = true
		}
	}
	return ingressNamesSet, nil
}
