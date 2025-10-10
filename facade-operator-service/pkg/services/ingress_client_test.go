package services

import (
	"context"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	mock_services "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/services"
	"go.uber.org/mock/gomock"
	"os"
	"testing"

	openshiftv1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIngressClientImpl_DeleteOrphaned(t *testing.T) {
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
	ingressClient := NewIngressClientAggregator(k8sClient, templates.NewIngressTemplateBuilder(false, false, ""), commonCRClient)

	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-gw"}}
	ingressReq := types.NamespacedName{Namespace: "test-ns", Name: "test-gw-web"}
	ingressTemplate1 := templates.Ingress{
		Name:        "test-gw-web",
		Namespace:   "test-ns",
		Annotations: map[string]string{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"},
		Hostname:    "test-host.netcracker.com",
		ServiceName: "test-gw",
		Port:        8080,
	}
	ingressTemplate2 := templates.Ingress{
		Name:        "test-gw2-web",
		Namespace:   "test-ns",
		Annotations: map[string]string{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"},
		Hostname:    "test-host2.netcracker.com",
		ServiceName: "test-gw2",
		Port:        8080,
	}
	ingress1 := ingressTemplate1.BuildK8sIngress()
	ingress2 := ingressTemplate2.BuildK8sIngress()

	ingresses := &v1.IngressList{
		Items: []v1.Ingress{*ingress1, *ingress2},
	}
	cr := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw2", Namespace: "test-ns"},
		Spec: facade.FacadeServiceSpec{
			Gateway:     "test-gw2",
			Port:        8080,
			GatewayType: facade.Ingress,
			Ingresses: []facade.IngressSpec{{
				Hostname:    "test-host2.netcracker.com",
				IsGrpc:      false,
				GatewayPort: 8080,
			}},
		},
	}
	crs := []facade.MeshGateway{cr}

	k8sClient.EXPECT().List(context.Background(), &v1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1.Ingress, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(nil)
	err := ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1.Ingress, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1.Ingress, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)
}

func TestIngressClientImpl_DeleteOrphanedV1Beta1(t *testing.T) {
	os.Setenv("PAAS_PLATFORM", "kubernetes")
	os.Setenv("PAAS_VERSION", "v1.11.0")
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
	ingressClient := NewIngressClientAggregator(k8sClient, templates.NewIngressTemplateBuilder(false, false, ""), commonCRClient)

	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-gw"}}
	ingressReq := types.NamespacedName{Namespace: "test-ns", Name: "test-gw-web"}
	ingressTemplate1 := templates.Ingress{
		Name:        "test-gw-web",
		Namespace:   "test-ns",
		Annotations: map[string]string{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"},
		Hostname:    "test-host.netcracker.com",
		ServiceName: "test-gw",
		Port:        8080,
	}
	ingressTemplate2 := templates.Ingress{
		Name:        "test-gw2-web",
		Namespace:   "test-ns",
		Annotations: map[string]string{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"},
		Hostname:    "test-host2.netcracker.com",
		ServiceName: "test-gw2",
		Port:        8080,
	}
	ingress1 := ingressTemplate1.BuildK8sBetaIngress()
	ingress2 := ingressTemplate2.BuildK8sBetaIngress()

	ingresses := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{*ingress1, *ingress2},
	}
	cr := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw2", Namespace: "test-ns"},
		Spec: facade.FacadeServiceSpec{
			Gateway:     "test-gw2",
			Port:        8080,
			GatewayType: facade.Ingress,
			Ingresses: []facade.IngressSpec{{
				Hostname:    "test-host2.netcracker.com",
				IsGrpc:      false,
				GatewayPort: 8080,
			}},
		},
	}
	crs := []facade.MeshGateway{cr}

	k8sClient.EXPECT().List(context.Background(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1beta1.Ingress, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(nil)
	err := ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1beta1.Ingress, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1beta1.Ingress, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)
}

func TestIngressClientImpl_DeleteOrphanedOpenshift(t *testing.T) {
	os.Setenv("PAAS_PLATFORM", "openshift")
	os.Setenv("PAAS_VERSION", "v3.11.0")
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
	ingressClient := NewIngressClientAggregator(k8sClient, templates.NewIngressTemplateBuilder(false, false, ""), commonCRClient)

	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-gw"}}
	ingressReq := types.NamespacedName{Namespace: "test-ns", Name: "test-gw-web"}
	ingressTemplate1 := templates.Ingress{
		Name:        "test-gw-web",
		Namespace:   "test-ns",
		Annotations: map[string]string{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"},
		Hostname:    "test-host.netcracker.com",
		ServiceName: "test-gw",
		Port:        8080,
	}
	ingressTemplate2 := templates.Ingress{
		Name:        "test-gw2-web",
		Namespace:   "test-ns",
		Annotations: map[string]string{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"},
		Hostname:    "test-host2.netcracker.com",
		ServiceName: "test-gw2",
		Port:        8080,
	}
	ingress1 := ingressTemplate1.BuildOpenshiftRoute()
	ingress2 := ingressTemplate2.BuildOpenshiftRoute()

	ingresses := &openshiftv1.RouteList{
		Items: []openshiftv1.Route{*ingress1, *ingress2},
	}
	cr := &facadeV1Alpha.FacadeService{
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw2", Namespace: "test-ns"},
		Spec: facade.FacadeServiceSpec{
			Gateway:     "test-gw2",
			Port:        8080,
			GatewayType: facade.Ingress,
			Ingresses: []facade.IngressSpec{{
				Hostname:    "test-host2.netcracker.com",
				IsGrpc:      false,
				GatewayPort: 8080,
			}},
		},
	}
	crs := []facade.MeshGateway{cr}

	k8sClient.EXPECT().List(context.Background(), &openshiftv1.RouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *openshiftv1.RouteList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *openshiftv1.Route, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(nil)
	err := ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &openshiftv1.RouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &openshiftv1.RouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &openshiftv1.RouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *openshiftv1.RouteList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &openshiftv1.RouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *openshiftv1.RouteList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &openshiftv1.RouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *openshiftv1.RouteList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)

	k8sClient.EXPECT().List(context.Background(), &openshiftv1.RouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *openshiftv1.RouteList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *openshiftv1.Route, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(getNotFoundError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.Nil(t, err)

	k8sClient.EXPECT().List(context.Background(), &openshiftv1.RouteList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *openshiftv1.RouteList, _ ...client.ListOption) error {
			*list = *ingresses
			return nil
		})
	commonCRClient.EXPECT().FindByFields(context.Background(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(crs, nil)
	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *openshiftv1.Route, _ ...client.GetOption) *gomock.Call {
		*obj = *ingress1
		return nil
	})
	k8sClient.EXPECT().Delete(context.Background(), ingress1).Return(getUnknownError())
	err = ingressClient.DeleteOrphaned(context.Background(), req)
	assert.NotNil(t, err)
}

func TestIngressClientImpl_Apply(t *testing.T) {
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
	ingressClient := NewIngressClientAggregator(k8sClient, templates.NewIngressTemplateBuilder(false, false, ""), commonCRClient)

	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-gw"}}
	ingressTemplate := templates.Ingress{
		Name:        "test-ingress",
		Namespace:   "test-ns",
		Annotations: map[string]string{"annotation1": "val1"},
		Hostname:    "test-host.netcracker.com",
		ServiceName: "test-gw",
		Port:        8080,
	}

	ingressReq := types.NamespacedName{Namespace: "test-ns", Name: "test-ingress"}
	ingress := ingressTemplate.BuildK8sIngress()

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(nil)
	err := ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.Nil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(getConflictError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1.Ingress, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(nil)
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.Nil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1.Ingress, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1.Ingress, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(getConflictError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)
}

func TestIngressClientImpl_ApplyK8sBetaV1(t *testing.T) {
	os.Setenv("PAAS_PLATFORM", "kubernetes")
	os.Setenv("PAAS_VERSION", "v1.11.0")
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
	ingressClient := NewIngressClientAggregator(k8sClient, templates.NewIngressTemplateBuilder(false, false, ""), commonCRClient)

	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-gw"}}
	ingressTemplate := templates.Ingress{
		Name:        "test-ingress",
		Namespace:   "test-ns",
		Annotations: map[string]string{"annotation1": "val1"},
		Hostname:    "test-host.netcracker.com",
		ServiceName: "test-gw",
		Port:        8080,
	}

	ingressReq := types.NamespacedName{Namespace: "test-ns", Name: "test-ingress"}
	ingress := ingressTemplate.BuildK8sBetaIngress()

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(nil)
	err := ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.Nil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(getConflictError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1beta1.Ingress, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(nil)
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.Nil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1beta1.Ingress, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &v1beta1.Ingress{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1beta1.Ingress, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(getConflictError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)
}

func TestIngressClientImpl_ApplyOpenshiftRoute(t *testing.T) {
	os.Setenv("PAAS_PLATFORM", "openshift")
	os.Setenv("PAAS_VERSION", "v3.11.0")
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
	ingressClient := NewIngressClientAggregator(k8sClient, templates.NewIngressTemplateBuilder(false, false, ""), commonCRClient)

	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-gw"}}
	ingressTemplate := templates.Ingress{
		Name:        "test-ingress",
		Namespace:   "test-ns",
		Annotations: map[string]string{"annotation1": "val1"},
		Hostname:    "test-host.netcracker.com",
		ServiceName: "test-gw",
		Port:        8080,
	}

	ingressReq := types.NamespacedName{Namespace: "test-ns", Name: "test-ingress"}
	ingress := ingressTemplate.BuildOpenshiftRoute()

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(nil)
	err := ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.Nil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(context.Background(), ingress, &client.CreateOptions{FieldManager: "facadeOperator"}).Return(getConflictError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *openshiftv1.Route, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(nil)
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.Nil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *openshiftv1.Route, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(getUnknownError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)

	k8sClient.EXPECT().Get(context.Background(), ingressReq, &openshiftv1.Route{}, &client.GetOptions{}).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *openshiftv1.Route, _ ...client.GetOption) error {
		*obj = *ingress
		return nil
	})
	k8sClient.EXPECT().Update(context.Background(), ingress, &client.UpdateOptions{FieldManager: "facadeOperator"}).Return(getConflictError())
	err = ingressClient.Apply(context.Background(), req, ingressTemplate)
	assert.NotNil(t, err)
}
