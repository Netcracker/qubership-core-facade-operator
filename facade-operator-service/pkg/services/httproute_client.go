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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	ingressBuilder      *templates.IngressTemplateBuilder
	commonCRClient      CommonCRClient
	logger              logging.Logger
	policyClient        client.Client
	backendPolicyClient GenericClient[*unstructured.Unstructured]
	clientPolicyClient  GenericClient[*unstructured.Unstructured]
}

func NewHTTPRouteClient(client client.Client, ingressBuilder *templates.IngressTemplateBuilder, commonCRClient CommonCRClient) *HTTPRouteClientImpl {
	return &HTTPRouteClientImpl{
		GenericClient:       NewGenericClient[*gatewayv1.HTTPRoute](client, "HTTPRoute"),
		ingressBuilder:      ingressBuilder,
		commonCRClient:      commonCRClient,
		logger:              logging.GetLogger("gatewayV1.HTTPRoute client"),
		policyClient:        client,
		backendPolicyClient: NewGenericClient[*unstructured.Unstructured](client, "BackendTrafficPolicy"),
		clientPolicyClient:  NewGenericClient[*unstructured.Unstructured](client, "ClientTrafficPolicy"),
	}
}

func (h *HTTPRouteClientImpl) Apply(ctx context.Context, req ctrl.Request, httpRoute templates.HTTPRoute) error {
	if err := h.GenericClient.Apply(ctx, req, &gatewayv1.HTTPRoute{}, httpRoute.BuildK8sHTTPRoute(), h.mergeHTTPRoutes); err != nil {
		return err
	}

	if httpRoute.BackendTrafficPolicy != nil {
		h.logger.InfoC(ctx, "[%v] Applying BackendTrafficPolicy for HTTPRoute %s", req.NamespacedName, httpRoute.Name)
		if err := h.backendPolicyClient.Apply(ctx, req, &unstructured.Unstructured{}, httpRoute.BackendTrafficPolicy, h.mergeUnstructuredPolicies); err != nil {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("failed to apply BackendTrafficPolicy for HTTPRoute %s", httpRoute.Name), err)
		}
	}

	if httpRoute.ClientTrafficPolicy != nil {
		h.logger.InfoC(ctx, "[%v] Applying ClientTrafficPolicy for HTTPRoute %s", req.NamespacedName, httpRoute.Name)
		if err := h.clientPolicyClient.Apply(ctx, req, &unstructured.Unstructured{}, httpRoute.ClientTrafficPolicy, h.mergeUnstructuredPolicies); err != nil {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("failed to apply ClientTrafficPolicy for HTTPRoute %s", httpRoute.Name), err)
		}
	}

	return nil
}

func (h *HTTPRouteClientImpl) mergeHTTPRoutes(existingResReceiver, newObject *gatewayv1.HTTPRoute) {
	newObject.ObjectMeta.ResourceVersion = existingResReceiver.ResourceVersion
	newObject.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(newObject.ObjectMeta.OwnerReferences, existingResReceiver.ObjectMeta.OwnerReferences)
}

func (h *HTTPRouteClientImpl) mergeUnstructuredPolicies(existingResReceiver, newObject *unstructured.Unstructured) {
	newObject.SetResourceVersion(existingResReceiver.GetResourceVersion())
	newObject.SetOwnerReferences(utils.MergeOwnerReferences(newObject.GetOwnerReferences(), existingResReceiver.GetOwnerReferences()))
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

	if err = h.deleteOrphanedPolicies(ctx, req, httpRouteNamesSet, "BackendTrafficPolicy"); err != nil {
		return err
	}
	if err = h.deleteOrphanedPolicies(ctx, req, httpRouteNamesSet, "ClientTrafficPolicy"); err != nil {
		return err
	}

	return nil
}

func (h *HTTPRouteClientImpl) Delete(ctx context.Context, req ctrl.Request, name string) error {
	if err := h.GenericClient.Delete(ctx, req, name, &gatewayv1.HTTPRoute{}); err != nil {
		return err
	}

	h.logger.InfoC(ctx, "[%v] Deleting BackendTrafficPolicy %s if exists", req.NamespacedName, name)
	backendPolicy := &unstructured.Unstructured{}
	backendPolicy.SetAPIVersion("gateway.envoyproxy.io/v1alpha1")
	backendPolicy.SetKind("BackendTrafficPolicy")
	if err := h.backendPolicyClient.Delete(ctx, req, name, backendPolicy); err != nil && !k8sErrors.IsNotFound(err) {
		h.logger.WarnC(ctx, "[%v] Failed to delete BackendTrafficPolicy %s: %v", req.NamespacedName, name, err)
	}

	// Delete associated ClientTrafficPolicy (same name as HTTPRoute)
	h.logger.InfoC(ctx, "[%v] Deleting ClientTrafficPolicy %s if exists", req.NamespacedName, name)
	clientPolicy := &unstructured.Unstructured{}
	clientPolicy.SetAPIVersion("gateway.envoyproxy.io/v1alpha1")
	clientPolicy.SetKind("ClientTrafficPolicy")
	if err := h.clientPolicyClient.Delete(ctx, req, name, clientPolicy); err != nil && !k8sErrors.IsNotFound(err) {
		h.logger.WarnC(ctx, "[%v] Failed to delete ClientTrafficPolicy %s: %v", req.NamespacedName, name, err)
	}

	return nil
}

func (h *HTTPRouteClientImpl) deleteOrphanedPolicies(ctx context.Context, req ctrl.Request, validNames map[string]bool, policyKind string) error {
	policyList := &unstructured.UnstructuredList{}
	policyList.SetAPIVersion("gateway.envoyproxy.io/v1alpha1")
	policyList.SetKind(policyKind + "List")

	err := h.policyClient.List(ctx, policyList, client.InNamespace(req.Namespace))
	if err != nil {
		if k8sErrors.IsNotFound(err) || k8sErrors.IsMethodNotSupported(err) {
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to list %s in namespace %s", policyKind, req.Namespace), err)
	}

	for _, policy := range policyList.Items {
		labels := policy.GetLabels()
		if labels["app.kubernetes.io/managed-by-operator"] == "facade-operator" {
			if !validNames[policy.GetName()] {
				h.logger.InfoC(ctx, "[%v] Deleting orphaned %s: %s", req.NamespacedName, policyKind, policy.GetName())
				if err := h.policyClient.Delete(ctx, &policy); err != nil && !k8sErrors.IsNotFound(err) {
					return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete %s %s", policyKind, policy.GetName()), err)
				}
			}
		}
	}

	return nil
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
