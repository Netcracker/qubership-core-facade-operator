package services

import (
	"context"
	"os"
	"testing"

	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	mock_services "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestHTTPRouteClientImpl_Apply(t *testing.T) {
	os.Setenv("PAAS_PLATFORM", "kubernetes")
	os.Setenv("PAAS_VERSION", "v1.23.0")
	utils.ReloadPlatform()
	defer func() {
		os.Unsetenv("PAAS_PLATFORM")
		os.Unsetenv("PAAS_VERSION")
		utils.ReloadPlatform()
	}()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	k8sClient := mock_client.NewMockClient(ctrl)
	commonCRClient := mock_services.NewMockCommonCRClient(ctrl)
	httpRouteClient := NewHTTPRouteClient(k8sClient, templates.NewIngressTemplateBuilder(false, false, ""), commonCRClient)

	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-gw"}}
	httpRouteTemplate := templates.HTTPRoute{
		Name:            "test-httproute",
		Namespace:       "test-ns",
		Annotations:     map[string]string{"annotation1": "val1"},
		Hostname:        "test-host.qubership.org",
		ServiceName:     "test-gw",
		Port:            8080,
		ParentName:      "default-external-gateway",
		ParentNamespace: "gateway-system",
	}
	httpRouteReq := types.NamespacedName{Namespace: "test-ns", Name: "test-httproute"}
	httpRoute := httpRouteTemplate.BuildK8sHTTPRoute()

	k8sClient.EXPECT().Get(context.Background(), httpRouteReq, &gatewayv1.HTTPRoute{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), httpRoute, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(nil)
	err := httpRouteClient.Apply(context.Background(), req, httpRouteTemplate)
	assert.Nil(t, err)

	k8sClient.EXPECT().Get(context.Background(), httpRouteReq, &gatewayv1.HTTPRoute{}, &client.GetOptions{}).Return(getUnknownError())
	err = httpRouteClient.Apply(context.Background(), req, httpRouteTemplate)
	assert.NotNil(t, err)
}

func TestHTTPRouteClientImpl_DeleteOrphaned(t *testing.T) {
	os.Setenv("PAAS_PLATFORM", "kubernetes")
	os.Setenv("PAAS_VERSION", "v1.23.0")
	utils.ReloadPlatform()
	defer func() {
		os.Unsetenv("PAAS_PLATFORM")
		os.Unsetenv("PAAS_VERSION")
		utils.ReloadPlatform()
	}()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	k8sClient := mock_client.NewMockClient(ctrl)
	commonCRClient := mock_services.NewMockCommonCRClient(ctrl)
	httpRouteClient := NewHTTPRouteClient(k8sClient, templates.NewIngressTemplateBuilder(false, false, ""), commonCRClient)

	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-gw"}}
	httpRouteReq := types.NamespacedName{Namespace: "test-ns", Name: "test-gw-web"}
	httpRouteTemplate1 := templates.HTTPRoute{
		Name:            "test-gw-web",
		Namespace:       "test-ns",
		Annotations:     map[string]string{"app.kubernetes.io/managed-by": "facade-operator"},
		Hostname:        "test-host.qubership.org",
		ServiceName:     "test-gw",
		Port:            8080,
		ParentName:      "default-external-gateway",
		ParentNamespace: "gateway-system",
	}
	httpRouteTemplate2 := templates.HTTPRoute{
		Name:            "test-gw2-web",
		Namespace:       "test-ns",
		Annotations:     map[string]string{"app.kubernetes.io/managed-by": "facade-operator"},
		Hostname:        "test-host2.qubership.org",
		ServiceName:     "test-gw2",
		Port:            8080,
		ParentName:      "default-external-gateway",
		ParentNamespace: "gateway-system",
	}
	httpRoute1 := httpRouteTemplate1.BuildK8sHTTPRoute()
	httpRoute2 := httpRouteTemplate2.BuildK8sHTTPRoute()

	httpRoutes := &gatewayv1.HTTPRouteList{
		Items: []gatewayv1.HTTPRoute{*httpRoute1, *httpRoute2},
	}
	cr := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw2", Namespace: "test-ns"},
		Spec: facade.FacadeServiceSpec{
			Gateway:     "test-gw2",
			Port:        8080,
			GatewayType: facade.Ingress,
			Ingresses: []facade.IngressSpec{{
				Hostname:    "test-host2.qubership.org",
				IsGrpc:      false,
				GatewayPort: 8080,
			}},
		},
	}
	crs := []facade.MeshGateway{cr}

	k8sClient.EXPECT().List(context.Background(), &gatewayv1.HTTPRouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *gatewayv1.HTTPRouteList, _ ...client.ListOption) error {
			*list = *httpRoutes
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), httpRouteReq, &gatewayv1.HTTPRoute{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *gatewayv1.HTTPRoute, _ ...client.GetOption) error {
		*obj = *httpRoute1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), httpRoute1).Return(nil)
	err := httpRouteClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)
}
