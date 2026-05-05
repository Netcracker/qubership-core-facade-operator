package controllers

import (
	"context"
	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	monitoringv1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/monitoring/v1"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/services"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	mock_restclient "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/restclient"
	mock_services "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/services"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"go.uber.org/mock/gomock"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile_shouldApplyMeshRouterFailed_whenUnknownError(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	configloader.Init(configloader.EnvPropertySource())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, deploymentClient, serviceClient, podMonitorClient, configMapClient, k8sClient, statusUpdater, readyService, _, crPriorityService, hpaClient := getFacadeServiceReconciler(mockCtrl)
	ctx := context.Background()
	req := getFacadeServiceRequest()
	unknownErr := getUnknownError()
	publicGatewayImage := "publicGatewayImage"
	utils.MonitoringEnabled = "true"
	facadeService := &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: 2,
			Port:     8080,
			Gateway:  req.Name + "-composite",
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:      "10m",
				FacadeGatewayCpuRequest:    "1m",
				FacadeGatewayMemoryLimit:   "200Mi",
				FacadeGatewayMemoryRequest: "100Mi",
			},
		},
	}
	facadeGatewayName := req.Name + utils.GatewaySuffix

	var testMapper map[string]testStruct
	testMapper = map[string]testStruct{
		"WhileRegisteringGateway": {
			failedFunc: func() {
				reconciler.base.controlPlaneClient.(*mock_restclient.MockControlPlaneClient).EXPECT().RegisterGateway(gomock.Any(), req.Name+"-composite", facadeService).Return(errs.NewError(customerrors.ControlPlaneError, "control-plane err", unknownErr))
			},
			mockFunc: func() {
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).DoAndReturn(
					func(_ context.Context, _ types.NamespacedName, fs *facadeV1Alpha.FacadeService, _ *client.GetOptions) error {
						*fs = *facadeService
						return nil
					})
				deploymentClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, nil)
				readyService.EXPECT().IsUpdatingPhase(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				statusUpdater.EXPECT().SetUpdating(gomock.Any(), gomock.Any()).AnyTimes()
				statusUpdater.EXPECT().SetFail(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"WhileGettingImage": {
			failedFunc: func() {
				configMapClient.EXPECT().GetGatewayImage(gomock.Any(), req).Return("", unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileRegisteringGateway"].mockFunc()
				reconciler.base.controlPlaneClient.(*mock_restclient.MockControlPlaneClient).EXPECT().RegisterGateway(gomock.Any(), req.Name+"-composite", facadeService).Return(nil)
			},
		},
		"WhileApplyFacadeService": {
			failedFunc: func() {
				serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
					req,
					req.Name,
					facadeService.Spec.Gateway,
					facadeService.Spec.Port,
				)).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileGettingImage"].mockFunc()
				configMapClient.EXPECT().GetGatewayImage(gomock.Any(), req).Return(publicGatewayImage, nil)
			},
		},
		"WhileApplyMeshDeployment": {
			failedFunc: func() {
				deploymentClient.EXPECT().Apply(gomock.Any(), req, getDeployment(reconciler.base,
					req,
					facadeService.Spec.Gateway,
					facadeService.Spec.Gateway,
					publicGatewayImage,
					facadeService,
					true,
					"1",
				)).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileApplyFacadeService"].mockFunc()
				crPriorityService.EXPECT().UpdateAvailable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
					req,
					req.Name,
					facadeService.Spec.Gateway,
					facadeService.Spec.Port,
				)).Return(nil)
			},
		},
		"WhileApplyMeshService": {
			failedFunc: func() {
				serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
					req,
					facadeService.Spec.Gateway,
					facadeService.Spec.Gateway,
					facadeService.Spec.Port,
				)).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileApplyMeshDeployment"].mockFunc()
				deploymentClient.EXPECT().Apply(gomock.Any(), req, getDeployment(reconciler.base, req, facadeService.Spec.Gateway, facadeService.Spec.Gateway, publicGatewayImage, facadeService, true, "1")).Return(nil)
			},
		},
		"WhileApplyConfigMap": {
			failedFunc: func() {
				configMapClient.EXPECT().Apply(gomock.Any(), req, getConfigMap(
					req,
					facadeService.Spec.Gateway,
				)).Return(unknownErr)
				hpaClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any())
			},
			mockFunc: func() {
				testMapper["WhileApplyMeshService"].mockFunc()
				serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
					req,
					facadeService.Spec.Gateway,
					facadeService.Spec.Gateway,
					facadeService.Spec.Port,
				)).Return(nil)
			},
		},
		"WhileApplyPodMonitor": {
			failedFunc: func() {
				podMonitorClient.EXPECT().Create(gomock.Any(), req, getPodMonitor(
					req,
					facadeService.Spec.Gateway,
				)).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileApplyConfigMap"].mockFunc()
				configMapClient.EXPECT().Apply(gomock.Any(), req, getConfigMap(
					req,
					facadeService.Spec.Gateway,
				)).Return(nil)
			},
		},
		"WhileIsFacadeGateway": {
			failedFunc: func() {
				deploymentClient.EXPECT().IsFacadeGateway(gomock.Any(), req, facadeGatewayName).Return(false, unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileApplyPodMonitor"].mockFunc()
				podMonitorClient.EXPECT().Create(gomock.Any(), req, getPodMonitor(
					req,
					facadeService.Spec.Gateway,
				)).Return(nil)
				hpaClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any())
			},
		},
		"WhileDeletingFacadeGatewayDeployment": {
			failedFunc: func() {
				deploymentClient.EXPECT().Delete(gomock.Any(), req, facadeGatewayName).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileIsFacadeGateway"].mockFunc()
				deploymentClient.EXPECT().IsFacadeGateway(gomock.Any(), req, facadeGatewayName).Return(true, nil)
			},
		},
		"WhileDeletingFacadeGatewayConfigMap": {
			failedFunc: func() {
				configMapClient.EXPECT().Delete(gomock.Any(), req, facadeGatewayName+monitoringConfigSuffix).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeployment"].mockFunc()
				deploymentClient.EXPECT().Delete(gomock.Any(), req, facadeGatewayName).Return(nil)
			},
		},
		"WhileDeletingFacadeGatewayPodMonitor": {
			failedFunc: func() {
				podMonitorClient.EXPECT().Delete(gomock.Any(), req, getShortName(facadeGatewayName, podMonitorSuffix)).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayConfigMap"].mockFunc()
				configMapClient.EXPECT().Delete(gomock.Any(), req, facadeGatewayName+monitoringConfigSuffix).Return(nil)
			},
		},
		"WhileDeletingNotUsedMeshRouters": {
			failedFunc: func() {
				deploymentClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayCertificate"].mockFunc()
			},
		},
	}

	tests := []testStruct{
		{
			name:      "WhileRegisteringGateway",
			errorCode: customerrors.ControlPlaneError,
			details:   "control-plane err",
			failedFunc: func() {
				testMapper["WhileRegisteringGateway"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileRegisteringGateway"].mockFunc()
			},
		},
		{
			name:      "WhileGettingImage",
			errorCode: customerrors.GatewayImageError,
			details:   "gateway image is empty",
			failedFunc: func() {
				testMapper["WhileGettingImage"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileGettingImage"].mockFunc()
			},
		},
		{
			name:      "WhileApplyFacadeService",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileApplyFacadeService"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileApplyFacadeService"].mockFunc()
			},
		},
		{
			name:      "WhileApplyMeshDeployment",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileApplyMeshDeployment"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileApplyMeshDeployment"].mockFunc()
			},
		},
		{
			name:      "WhileApplyMeshService",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileApplyMeshService"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileApplyMeshService"].mockFunc()
			},
		},
		{
			name:      "WhileApplyConfigMap",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileApplyConfigMap"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileApplyConfigMap"].mockFunc()
			},
		},
		{
			name:      "WhileApplyPodMonitor",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileApplyPodMonitor"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileApplyPodMonitor"].mockFunc()
			},
		},
		{
			name:      "WhileIsFacadeGateway",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileIsFacadeGateway"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileIsFacadeGateway"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingFacadeGatewayDeployment",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeployment"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeployment"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingFacadeGatewayConfigMap",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingFacadeGatewayConfigMap"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayConfigMap"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingFacadeGatewayPodMonitor",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingFacadeGatewayPodMonitor"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayPodMonitor"].mockFunc()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			tt.failedFunc()

			_, err := reconciler.Reconcile(ctx, req)
			assert.NotNil(t, err)
			assert.Equal(t, tt.errorCode, err.(*errs.ErrCodeError).ErrorCode)
			assert.Equal(t, tt.details, err.(*errs.ErrCodeError).Detail)
			assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
		})
	}
}

func TestReconcile_shouldApplyFacadeRouterFailed_whenUnknownError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, deploymentClient, serviceClient, podMonitorClient, configMapClient, k8sClient, statusUpdater, readyService, _, _, hpaClient := getFacadeServiceReconciler(mockCtrl)
	ctx := context.Background()
	req := getFacadeServiceRequest()
	unknownErr := getUnknownError()
	publicGatewayImage := "publicGatewayImage"
	utils.MonitoringEnabled = "true"
	facadeService := &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: int64(2),
			Port:     8080,
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:      "10m",
				FacadeGatewayCpuRequest:    "1m",
				FacadeGatewayMemoryLimit:   "200Mi",
				FacadeGatewayMemoryRequest: "100Mi",
			},
		},
	}
	facadeGatewayName := req.Name + utils.GatewaySuffix

	var testMapper map[string]testStruct
	testMapper = map[string]testStruct{
		"WhileRegisteringGateway": {
			failedFunc: func() {
				reconciler.base.controlPlaneClient.(*mock_restclient.MockControlPlaneClient).EXPECT().RegisterGateway(gomock.Any(), req.Name, facadeService).Return(errs.NewError(customerrors.ControlPlaneError, "control-plane err", unknownErr))
			},
			mockFunc: func() {
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).DoAndReturn(
					func(_ context.Context, _ types.NamespacedName, fs *facadeV1Alpha.FacadeService, _ *client.GetOptions) error {
						*fs = *facadeService
						return nil
					})
				deploymentClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, nil)
				readyService.EXPECT().IsUpdatingPhase(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				statusUpdater.EXPECT().SetUpdating(gomock.Any(), gomock.Any()).AnyTimes()
				statusUpdater.EXPECT().SetFail(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"WhileGettingImage": {
			failedFunc: func() {
				configMapClient.EXPECT().GetGatewayImage(gomock.Any(), req).Return("", unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileRegisteringGateway"].mockFunc()
				reconciler.base.controlPlaneClient.(*mock_restclient.MockControlPlaneClient).EXPECT().RegisterGateway(gomock.Any(), req.Name, facadeService).Return(nil)
			},
		},
		"WhileApplyFacadeService": {
			failedFunc: func() {
				serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
					req,
					req.Name,
					facadeGatewayName,
					facadeService.Spec.Port,
				)).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileGettingImage"].mockFunc()
				configMapClient.EXPECT().GetGatewayImage(gomock.Any(), req).Return(publicGatewayImage, nil)
			},
		},
		"WhileApplyFacadeDeployment": {
			failedFunc: func() {
				deploymentClient.EXPECT().Apply(gomock.Any(), req, getDeployment(reconciler.base, req, req.Name, facadeGatewayName, publicGatewayImage, facadeService, false, "1")).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileApplyFacadeService"].mockFunc()
				serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
					req,
					req.Name,
					facadeGatewayName,
					facadeService.Spec.Port,
				)).Return(nil)
			},
		},
		"WhileApplyFacadeConfigMap": {
			failedFunc: func() {
				configMapClient.EXPECT().Apply(gomock.Any(), req, getConfigMap(
					req,
					facadeGatewayName,
				)).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileApplyFacadeDeployment"].mockFunc()
				deploymentClient.EXPECT().Apply(gomock.Any(), req, getDeployment(reconciler.base, req, req.Name, facadeGatewayName, publicGatewayImage, facadeService, false, "1")).Return(nil)
			},
		},
		"WhileCreateFacadePodMonitor": {
			failedFunc: func() {
				podMonitorClient.EXPECT().Create(gomock.Any(), req, getPodMonitor(
					req,
					facadeGatewayName,
				)).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileApplyFacadeConfigMap"].mockFunc()
				configMapClient.EXPECT().Apply(gomock.Any(), req, getConfigMap(
					req,
					facadeGatewayName,
				)).Return(nil)
				hpaClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	tests := []testStruct{
		{
			name:      "WhileRegisteringGateway",
			errorCode: customerrors.ControlPlaneError,
			details:   "control-plane err",
			failedFunc: func() {
				testMapper["WhileRegisteringGateway"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileRegisteringGateway"].mockFunc()
			},
		},
		{
			name:      "WhileGettingImage",
			errorCode: customerrors.GatewayImageError,
			details:   "gateway image is empty",
			failedFunc: func() {
				testMapper["WhileGettingImage"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileGettingImage"].mockFunc()
			},
		},
		{
			name:      "WhileApplyFacadeService",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileApplyFacadeService"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileApplyFacadeService"].mockFunc()
			},
		},
		{
			name:      "WhileApplyFacadeDeployment",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileApplyFacadeDeployment"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileApplyFacadeDeployment"].mockFunc()
			},
		},
		{
			name:      "WhileApplyFacadeConfigMap",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileApplyFacadeConfigMap"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileApplyFacadeConfigMap"].mockFunc()
			},
		},
		{
			name:      "WhileCreateFacadePodMonitor",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileCreateFacadePodMonitor"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileCreateFacadePodMonitor"].mockFunc()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			tt.failedFunc()

			_, err := reconciler.Reconcile(ctx, req)
			assert.NotNil(t, err)
			assert.Equal(t, tt.errorCode, err.(*errs.ErrCodeError).ErrorCode)
			assert.Equal(t, tt.details, err.(*errs.ErrCodeError).Detail)
			assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
		})
	}
}

func TestReconcile_shouldApplyFacadeRouter_whenNoErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	req := getFacadeServiceRequest()
	publicGatewayImage := "publicGatewayImage"
	utils.MonitoringEnabled = "true"

	reconciler, deploymentClient, serviceClient, podMonitorClient, configMapClient, k8sClient, statusUpdater, readyService, _, _, hpaClient := getFacadeServiceReconciler(mockCtrl)

	facadeService := &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: "2",
			Port:     8080,
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:      "10m",
				FacadeGatewayCpuRequest:    "1m",
				FacadeGatewayMemoryLimit:   "200Mi",
				FacadeGatewayMemoryRequest: "100Mi",
			},
		},
	}
	facadeGatewayName := req.Name + utils.GatewaySuffix

	gomock.InOrder(
		k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).DoAndReturn(
			func(_ context.Context, _ types.NamespacedName, fs *facadeV1Alpha.FacadeService, _ *client.GetOptions) error {
				*fs = *facadeService
				return nil
			}),
		deploymentClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, nil),
		readyService.EXPECT().IsUpdatingPhase(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes(),
		statusUpdater.EXPECT().SetUpdating(gomock.Any(), gomock.Any()).AnyTimes(),
		reconciler.base.controlPlaneClient.(*mock_restclient.MockControlPlaneClient).EXPECT().RegisterGateway(gomock.Any(), req.Name, facadeService).Return(nil),
		configMapClient.EXPECT().GetGatewayImage(gomock.Any(), req).Return(publicGatewayImage, nil),
		serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
			req,
			req.Name,
			facadeGatewayName,
			facadeService.Spec.Port,
		)).Return(nil),
		deploymentClient.EXPECT().Apply(gomock.Any(), req, getDeployment(reconciler.base, req, req.Name, facadeGatewayName, publicGatewayImage, facadeService, false, "1")).Return(nil),
		configMapClient.EXPECT().Apply(gomock.Any(), req, getConfigMap(
			req,
			facadeGatewayName,
		)).Return(nil),
		hpaClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()),
		podMonitorClient.EXPECT().Create(gomock.Any(), req, getPodMonitor(
			req,
			facadeGatewayName,
		)).Return(nil),
		k8sClient.EXPECT().List(gomock.Any(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
			DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, _ ...client.ListOption) error {
				*list = v1beta1.IngressList{}
				return nil
			}),
	)

	result, err := reconciler.Reconcile(ctx, req)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestReconcile_shouldApplyMeshRouter_whenNoErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	req := getFacadeServiceRequest()
	publicGatewayImage := "publicGatewayImage"
	utils.MonitoringEnabled = "true"

	reconciler, deploymentClient, serviceClient, podMonitorClient, configMapClient, k8sClient, statusUpdater, readyService, _, crPriorityService, hpaClient := getFacadeServiceReconciler(mockCtrl)

	facadeService := &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: int32(2),
			Port:     8080,
			Gateway:  req.Name + "-composite",
			Env: facade.FacadeServiceEnv{
				FacadeGatewayCpuLimit:      "10m",
				FacadeGatewayCpuRequest:    "1m",
				FacadeGatewayMemoryLimit:   "200Mi",
				FacadeGatewayMemoryRequest: "100Mi",
			},
		},
	}
	facadeGatewayName := req.Name + utils.GatewaySuffix

	gomock.InOrder(
		k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).DoAndReturn(
			func(_ context.Context, _ types.NamespacedName, fs *facadeV1Alpha.FacadeService, _ *client.GetOptions) error {
				*fs = *facadeService
				return nil
			}),
		deploymentClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, nil),
		readyService.EXPECT().IsUpdatingPhase(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes(),
		statusUpdater.EXPECT().SetUpdating(gomock.Any(), gomock.Any()).AnyTimes(),
		reconciler.base.controlPlaneClient.(*mock_restclient.MockControlPlaneClient).EXPECT().RegisterGateway(gomock.Any(), req.Name+"-composite", facadeService).Return(nil),
		configMapClient.EXPECT().GetGatewayImage(gomock.Any(), req).Return(publicGatewayImage, nil),
		serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
			req,
			req.Name,
			facadeService.Spec.Gateway,
			facadeService.Spec.Port,
		)).Return(nil),
		crPriorityService.EXPECT().UpdateAvailable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil),
		deploymentClient.EXPECT().Apply(gomock.Any(), req, getDeployment(reconciler.base, req, facadeService.Spec.Gateway, facadeService.Spec.Gateway, publicGatewayImage, facadeService, true, "1")).Return(nil),
		serviceClient.EXPECT().Apply(gomock.Any(), req, getService(
			req,
			facadeService.Spec.Gateway,
			facadeService.Spec.Gateway,
			facadeService.Spec.Port,
		)).Return(nil),
		configMapClient.EXPECT().Apply(gomock.Any(), req, getConfigMap(
			req,
			facadeService.Spec.Gateway,
		)).Return(nil),
		hpaClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()),
		podMonitorClient.EXPECT().Create(gomock.Any(), req, getPodMonitor(
			req,
			facadeService.Spec.Gateway,
		)).Return(nil),
		deploymentClient.EXPECT().IsFacadeGateway(gomock.Any(), req, facadeGatewayName).Return(true, nil),
		deploymentClient.EXPECT().Delete(gomock.Any(), req, facadeGatewayName).Return(nil),
		configMapClient.EXPECT().Delete(gomock.Any(), req, facadeGatewayName+monitoringConfigSuffix).Return(nil),
		podMonitorClient.EXPECT().Delete(gomock.Any(), req, getShortName(facadeGatewayName, podMonitorSuffix)).Return(nil),
		hpaClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()),
		k8sClient.EXPECT().List(gomock.Any(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
			DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, _ ...client.ListOption) error {
				*list = v1beta1.IngressList{}
				return nil
			}),
	)

	result, err := reconciler.Reconcile(ctx, req)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestReconcile_shouldDelete_whenNoErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	req := getFacadeServiceRequest()
	facadeService := getFacadeService(req)
	facadeDeployment := getFacadeDeployment(req, "testName", "testImage")

	reconciler, dplClient, serviceClient, podMonitorClient, configMapClient, k8sClient, statusUpdater, _, commonCRClient, _, hpaClient := getFacadeServiceReconciler(mockCtrl)

	k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).Return(getNotFoundError())
	deployments := &v1.DeploymentList{
		Items: []v1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test2",
				},
			},
		},
	}
	dplClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(deployments, nil)
	statusUpdater.EXPECT().SetUpdating(gomock.Any(), gomock.Any()).Times(0)

	k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			facadeService.Spec.Selector["app"] = req.Name + "-unknown"
			*fs = *facadeService
			return nil
		})
	k8sClient.EXPECT().Delete(gomock.Any(), facadeService).Return(nil)
	hpaClient.EXPECT().Delete(gomock.Any(), gomock.Any(), deployments.Items[0].GetName()).Return(nil)
	hpaClient.EXPECT().Delete(gomock.Any(), gomock.Any(), deployments.Items[1].GetName()).Return(nil)

	labels := facadeDeployment.GetLabels()
	labels[utils.MeshRouter] = "true"
	facadeDeployment.SetLabels(labels)
	dplClient.EXPECT().Get(gomock.Any(), gomock.Any(), req.Name+"-unknown").Return(facadeDeployment, nil)

	commonCRClient.EXPECT().GetAll(gomock.Any(), req).Return(
		[]facade.MeshGateway{
			&facadeV1Alpha.FacadeService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
			},
			&facadeV1Alpha.FacadeService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test2",
				},
			},
		}, nil)
	dplClient.EXPECT().Delete(gomock.Any(), req, "test1").Return(nil)
	dplClient.EXPECT().Delete(gomock.Any(), req, "test2").Return(nil)

	utils.MonitoringEnabled = "true"

	serviceClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1").Return(nil).Times(1)
	configMapClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1"+monitoringConfigSuffix).Return(nil)
	podMonitorClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1"+podMonitorSuffix).Return(nil)

	serviceClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test2").Return(nil).Times(1)
	configMapClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test2"+monitoringConfigSuffix).Return(nil)
	podMonitorClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test2"+podMonitorSuffix).Return(nil)

	dplClient.EXPECT().IsFacadeGateway(gomock.Any(), req, "testName"+utils.GatewaySuffix).Return(false, nil)

	k8sClient.EXPECT().List(gomock.Any(), &v1beta1.IngressList{}, client.MatchingFields{"metadata.annotations.app.kubernetes.io/managed-by": "facade-operator"}, client.InNamespace(req.Namespace)).
		DoAndReturn(func(_ context.Context, list *v1beta1.IngressList, opts ...client.ListOption) error {
			*list = v1beta1.IngressList{
				Items: []v1beta1.Ingress{{ObjectMeta: metav1.ObjectMeta{Name: "test-ingress-1-web"}}, {ObjectMeta: metav1.ObjectMeta{Name: "test-ingress-2-web"}}},
			}
			return nil
		})
	commonCRClient.EXPECT().FindByFields(gomock.Any(), req, client.MatchingFields{"spec.gatewayType": string(facade.Ingress)}).Return(
		[]facade.MeshGateway{
			&facadeV1Alpha.FacadeService{
				ObjectMeta: metav1.ObjectMeta{Name: "another-facade-srv"},
				Spec: facade.FacadeServiceSpec{
					Gateway:             "test-ingress-1",
					Port:                8080,
					MasterConfiguration: false,
					GatewayType:         facade.Ingress,
					Ingresses: []facade.IngressSpec{{
						Hostname:    "test-ingress.com",
						IsGrpc:      false,
						GatewayPort: 8080,
					}},
				},
			},
		}, nil)
	statusUpdater.EXPECT().SetUpdated(gomock.Any(), gomock.Any()).AnyTimes()

	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      "test-ingress-1-web",
	}
	ingress := v1beta1.Ingress{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "test-ingress-1-web", Namespace: req.Namespace},
	}
	k8sClient.EXPECT().Get(gomock.Any(), nameSpacedRequest, &v1beta1.Ingress{}, &client.GetOptions{}).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1beta1.Ingress, _ ...client.GetOption) error {
			*obj = ingress
			return nil
		})

	k8sClient.EXPECT().Delete(gomock.Any(), &ingress).Return(nil)

	nameSpacedRequest = types.NamespacedName{
		Namespace: req.Namespace,
		Name:      "test-ingress-2-web",
	}
	ingress = v1beta1.Ingress{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "test-ingress-2-web", Namespace: req.Namespace},
	}
	k8sClient.EXPECT().Get(gomock.Any(), nameSpacedRequest, &v1beta1.Ingress{}, &client.GetOptions{}).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *v1beta1.Ingress, _ ...client.GetOption) error {
			*obj = ingress
			return nil
		})

	k8sClient.EXPECT().Delete(gomock.Any(), &ingress).Return(nil)

	reconciler.base.controlPlaneClient.(*mock_restclient.MockControlPlaneClient).EXPECT().DropGateway(gomock.Any(), "testName")

	result, err := reconciler.Reconcile(ctx, req)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestServiceShouldBeDeleted(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	req := getFacadeServiceRequest()
	reconciler, _, _, _, _, _, _, _, _, _, _ := getFacadeServiceReconciler(mockCtrl)

	foundDeployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				utils.FacadeGateway: "false",
				utils.MeshRouter:    "false",
			},
		},
	}
	serviceName := "serviceName"
	serviceSelector := "serviceSelector"

	result := reconciler.base.serviceShouldBeDeleted(context.Background(), req, foundDeployment, serviceName, serviceSelector)
	assert.False(t, result)

	result = reconciler.base.serviceShouldBeDeleted(context.Background(), req, nil, serviceName, serviceSelector)
	assert.True(t, result)

	foundDeployment.SetLabels(map[string]string{
		utils.FacadeGateway: "true",
		utils.MeshRouter:    "true",
	})
	result = reconciler.base.serviceShouldBeDeleted(context.Background(), req, nil, serviceName, serviceSelector)
	assert.True(t, result)
}

func TestReconcile_shouldDeleteFailed_whenUnknownError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, dplClient, serviceClient, podMonitorClient, configMapClient, k8sClient, statusUpdater, _, commonCRClient, _, _ := getFacadeServiceReconciler(mockCtrl)
	ctx := context.Background()
	req := getFacadeServiceRequest()
	unknownErr := getUnknownError()
	notFoundErr := getNotFoundError()
	facadeService := getFacadeService(req)
	facadeDeployment := getFacadeDeployment(req, "testName", "testImage")

	var testMapper map[string]testStruct
	testMapper = map[string]testStruct{
		"WhileGettingGatewayBySelector": {
			failedFunc: func() {
				dplClient.EXPECT().Get(gomock.Any(), gomock.Any(), req.Name+"-unknown").Return(nil, unknownErr)
			},
			mockFunc: func() {
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).Return(notFoundErr)
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
					func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
						facadeService.Spec.Selector["app"] = req.Name + "-unknown"
						*fs = *facadeService
						return nil
					})
				dplClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, nil)
				statusUpdater.EXPECT().SetUpdating(gomock.Any(), gomock.Any()).AnyTimes()
				statusUpdater.EXPECT().SetUpdated(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		"WhileDeletingGatewayBySelector": {
			failedFunc: func() {
				k8sClient.EXPECT().Delete(gomock.Any(), facadeService).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileGettingGatewayBySelector"].mockFunc()
				labels := facadeDeployment.GetLabels()
				labels[utils.MeshRouter] = "true"
				facadeDeployment.SetLabels(labels)
				dplClient.EXPECT().Get(gomock.Any(), gomock.Any(), req.Name+"-unknown").Return(facadeDeployment, nil)
			},
		},
		"WhileGettingMeshRouterDeploymentsNames": {
			failedFunc: func() {
				dplClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, unknownErr)
			},
			mockFunc: func() {
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).Return(notFoundErr)
			},
		},
		"WhileGettingFacadeServiceList": {
			failedFunc: func() {
				commonCRClient.EXPECT().GetAll(gomock.Any(), req).Return(nil, unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileGettingMeshRouterDeploymentsNames"].mockFunc()
				deployments := &v1.DeploymentList{
					Items: []v1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test1",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test2",
							},
						},
					},
				}
				dplClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(deployments, nil)
			},
		},
		"WhileDeletingMeshRouterService": {
			failedFunc: func() {
				serviceClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1").Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileGettingFacadeServiceList"].mockFunc()
				commonCRClient.EXPECT().GetAll(gomock.Any(), req).Return([]facade.MeshGateway{
					&facadeV1Alpha.FacadeService{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test1",
						},
					},
					&facadeV1Alpha.FacadeService{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test2",
						},
					},
				}, nil)
			},
		},
		"WhileDeletingMeshRouterDeployment": {
			failedFunc: func() {
				dplClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1").Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingMeshRouterService"].mockFunc()
				serviceClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1").Return(nil)
			},
		},
		"WhileDeletingMeshRouterConfigMap": {
			failedFunc: func() {
				configMapClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1"+monitoringConfigSuffix).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingMeshRouterDeployment"].mockFunc()
				dplClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1").Return(nil)
			},
		},
		"WhileDeletingMeshRouterPodMonitor": {
			failedFunc: func() {
				podMonitorClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1"+podMonitorSuffix).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingMeshRouterConfigMap"].mockFunc()
				configMapClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "test1"+monitoringConfigSuffix).Return(nil)
				utils.MonitoringEnabled = "true"
			},
		},
		"WhileDeletingFacadeGatewayIsFacadeGateway": {
			failedFunc: func() {
				dplClient.EXPECT().IsFacadeGateway(gomock.Any(), gomock.Any(), "testName-gateway").Return(false, unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileGettingMeshRouterDeploymentsNames"].mockFunc()
				dplClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, nil)
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &corev1.Service{}, &client.GetOptions{}).Return(notFoundErr)
				utils.MonitoringEnabled = "false"
			},
		},
		"WhileDeletingFacadeGatewayDeleteDeployment": {
			failedFunc: func() {
				dplClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "testName-gateway").Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayIsFacadeGateway"].mockFunc()
				dplClient.EXPECT().IsFacadeGateway(gomock.Any(), gomock.Any(), "testName-gateway").Return(true, nil)
			},
		},
		"WhileDeletingFacadeGatewayDeleteConfigMap": {
			failedFunc: func() {
				configMapClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "testName-gateway"+monitoringConfigSuffix).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeleteDeployment"].mockFunc()
				dplClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "testName-gateway").Return(nil)
			},
		},
		"WhileDeletingFacadeGatewayDeletePodMonitor": {
			failedFunc: func() {
				podMonitorClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "testName-gateway"+podMonitorSuffix).Return(unknownErr)
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeleteConfigMap"].mockFunc()
				configMapClient.EXPECT().Delete(gomock.Any(), gomock.Any(), "testName-gateway"+monitoringConfigSuffix).Return(nil)
				utils.MonitoringEnabled = "true"
			},
		},
	}

	tests := []testStruct{
		{
			name:      "WhileGettingCR",
			errorCode: customerrors.UnexpectedKubernetesError,
			details:   "Failed to get facade CR",
			failedFunc: func() {
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).Return(unknownErr)
			},
			mockFunc: func() {
				statusUpdater.EXPECT().SetFail(gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		{
			name:      "WhileGettingFacadeService",
			errorCode: customerrors.UnexpectedKubernetesError,
			details:   fmt.Sprintf("Failed to get service %v", facadeService.Name),
			failedFunc: func() {
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &corev1.Service{}, &client.GetOptions{}).Return(unknownErr)
			},
			mockFunc: func() {
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).Return(notFoundErr)
				dplClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, nil)
			},
		},
		{
			name:      "WhileDeletingFacadeServiceByGatewayName",
			errorCode: customerrors.UnexpectedKubernetesError,
			details:   fmt.Sprintf("Failed to delete service %v", facadeService.Name),
			failedFunc: func() {
				k8sClient.EXPECT().Delete(gomock.Any(), facadeService).Return(unknownErr)
			},
			mockFunc: func() {
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &facadeV1Alpha.FacadeService{}, &client.GetOptions{}).Return(notFoundErr)
				k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
					func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
						facadeService.Spec.Selector["app"] = req.Name + "-gateway"
						*fs = *facadeService
						return nil
					})
				dplClient.EXPECT().GetMeshRouterDeployments(gomock.Any(), req).Return(nil, nil)
			},
		},
		{
			name:      "WhileGettingGatewayBySelector",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileGettingGatewayBySelector"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileGettingGatewayBySelector"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingGatewayBySelector",
			errorCode: customerrors.UnexpectedKubernetesError,
			details:   fmt.Sprintf("Failed to delete service %v", facadeService.Name),
			failedFunc: func() {
				testMapper["WhileDeletingGatewayBySelector"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingGatewayBySelector"].mockFunc()
			},
		},
		{
			name:      "WhileGettingMeshRouterDeploymentsNames",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileGettingMeshRouterDeploymentsNames"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileGettingMeshRouterDeploymentsNames"].mockFunc()
			},
		},
		{
			name:      "WhileGettingFacadeServiceList",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileGettingFacadeServiceList"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileGettingFacadeServiceList"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingMeshRouterService",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingMeshRouterService"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingMeshRouterService"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingMeshRouterDeployment",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingMeshRouterDeployment"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingMeshRouterDeployment"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingMeshRouterConfigMap",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingMeshRouterConfigMap"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingMeshRouterConfigMap"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingMeshRouterPodMonitor",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingMeshRouterPodMonitor"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingMeshRouterPodMonitor"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingFacadeGatewayIsFacadeGateway",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingFacadeGatewayIsFacadeGateway"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayIsFacadeGateway"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingFacadeGatewayDeleteDeployment",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeleteDeployment"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeleteDeployment"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingFacadeGatewayDeleteConfigMap",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeleteConfigMap"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeleteConfigMap"].mockFunc()
			},
		},
		{
			name:      "WhileDeletingFacadeGatewayDeletePodMonitor",
			errorCode: customerrors.UnknownErrorCode,
			details:   "Unknown error",
			failedFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeletePodMonitor"].failedFunc()
			},
			mockFunc: func() {
				testMapper["WhileDeletingFacadeGatewayDeletePodMonitor"].mockFunc()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			tt.failedFunc()

			_, err := reconciler.Reconcile(ctx, req)
			assert.NotNil(t, err)
			assert.Equal(t, tt.errorCode, err.(*errs.ErrCodeError).ErrorCode)
			assert.Equal(t, tt.details, err.(*errs.ErrCodeError).Detail)
			assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
		})
	}
}

type testStruct struct {
	name       string
	errorCode  errs.ErrorCode
	details    string
	failedFunc func()
	mockFunc   func()
}

func getShortName(entityName string, suffix string) string {
	if len(entityName)+len(suffix) > 64 {
		return entityName[0:63-len(suffix)] + suffix
	}
	return entityName + suffix
}

func getInstanceLabelValue(entityName string, suffix string) string {
	label := getShortName(entityName, utils.LabelsDelimiter+suffix)
	return labelRegexp.FindString(label)
}

func getFacadeResources(facadeService *facadeV1Alpha.FacadeService) corev1.ResourceRequirements {
	resources := corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%v", facadeService.Spec.Env.FacadeGatewayCpuRequest)),
			corev1.ResourceMemory: resource.MustParse(facadeService.Spec.Env.FacadeGatewayMemoryRequest),
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%v", facadeService.Spec.Env.FacadeGatewayCpuLimit)),
			corev1.ResourceMemory: resource.MustParse(facadeService.Spec.Env.FacadeGatewayMemoryLimit),
		},
	}

	return resources
}

func getDeployment(reconciler *FacadeCommonReconciler, req ctrl.Request, serviceName string, gatewayName string, publicGatewayImage string, facadeService *facadeV1Alpha.FacadeService, meshRouter bool, certGeneration string) *v1.Deployment {
	lastAppliedCR, _ := utils.JsonMarshal(utils.LastAppliedCr{
		ApiVersion: facadeService.GetAPIVersion(),
		Kind:       facadeService.GetKind(),
		Name:       facadeService.GetName(),
	})
	instanceLabelValue := getInstanceLabelValue(gatewayName, req.Namespace)
	deployment := templates.RouterDeployment{
		ServiceName:                serviceName,
		GatewayName:                gatewayName,
		NameSpace:                  req.Namespace,
		InstanceLabel:              instanceLabelValue,
		ArtifactDescriptionVersion: utils.ArtifactDescriptorVersion,
		ImageName:                  publicGatewayImage,
		Recourses:                  getFacadeResources(facadeService),
		TracingEnabled:             utils.TracingEnabled,
		TracingHost:                utils.TracingHost,
		IpStack:                    utils.IpStack,
		IpBind:                     utils.IpBind,
		MeshRouter:                 meshRouter,
		Replicas:                   reconciler.getFacadeReplicas(context.Background(), facadeService),
		GwTerminationGracePeriodS:  60,
		EnvoyConcurrency:           reconciler.getFacadeGatewayConcurrency(context.Background(), facadeService),
		LastAppliedCR:              lastAppliedCR,
	}

	return deployment.GetDeployment()
}

func getService(req ctrl.Request, name string, gatewayName string, port int32) *corev1.Service {
	service := templates.FacadeService{
		Name:         name,
		Namespace:    req.Namespace,
		NameSelector: gatewayName,
		Port:         port,
	}

	return service.GetService()
}

func getConfigMap(req ctrl.Request, gatewayName string) *corev1.ConfigMap {
	configMap := templates.FacadeConfigMap{
		Name:      gatewayName + monitoringConfigSuffix,
		Namespace: req.Namespace,
	}

	return configMap.GetConfigMap()
}

func getPodMonitor(req ctrl.Request, name string) *monitoringv1.PodMonitor {
	podMonitorName := name + podMonitorSuffix
	if len(name)+len(podMonitorSuffix) > 64 {
		podMonitorName = name[0:63-len(podMonitorSuffix)] + podMonitorSuffix
	}

	podMonitor := &templates.FacadePodMonitor{
		Name:         podMonitorName,
		NameSpace:    req.Namespace,
		NameLabel:    podMonitorName,
		NameSelector: name,
		TLSEnable:    true,
	}

	return podMonitor.GetPodMonitor()
}

func getFacadeServiceRequest() reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "testName",
		},
	}
}

func getFacadeServiceReconciler(ctrl *gomock.Controller) (
	*FacadeServiceReconciler, *mock_services.MockDeploymentClient,
	*mock_services.MockServiceClient, *mock_services.MockPodMonitorClient,
	*mock_services.MockConfigMapClient, *mock_client.MockClient,
	*mock_services.MockStatusUpdater, *mock_services.MockReadyService, *mock_services.MockCommonCRClient,
	*mock_services.MockCRPriorityService, *mock_services.MockHPAClient) {
	return getFacadeServiceReconcilerWithTLS(ctrl, true)
}

func getFacadeServiceReconcilerWithTLS(ctrl *gomock.Controller, tlsEnable bool) (
	*FacadeServiceReconciler, *mock_services.MockDeploymentClient,
	*mock_services.MockServiceClient, *mock_services.MockPodMonitorClient,
	*mock_services.MockConfigMapClient, *mock_client.MockClient,
	*mock_services.MockStatusUpdater, *mock_services.MockReadyService, *mock_services.MockCommonCRClient,
	*mock_services.MockCRPriorityService, *mock_services.MockHPAClient) {
	k8sClient := GetMockClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	statusUpdater := GetMockStatusUpdater(ctrl)
	readyService := GetMockReadyService(ctrl)
	serviceClient := GetMockServiceClient(ctrl)
	podMinitorClient := GetMockPodMonitorClient(ctrl)
	hpaClient := GetMockHPAClient(ctrl)
	configMapClient := GetMockConfigMapClient(ctrl)
	controlPlaneClient := mock_restclient.NewMockControlPlaneClient(ctrl)
	ingressBuilder := templates.NewIngressTemplateBuilder(false, false, "")
	crPriorityService := GetMockCRPriorityService(ctrl)
	commonCRClient := GetMockCommonCRClient(ctrl)
	ingressClient := services.NewIngressClientAggregator(k8sClient, ingressBuilder, commonCRClient)
	commonReconciler := NewFacadeCommonReconciler(k8sClient, serviceClient, deploymentClient, configMapClient, podMinitorClient, hpaClient, ingressClient, controlPlaneClient, ingressBuilder, statusUpdater, readyService, commonCRClient, crPriorityService)
	return NewFacadeServiceReconciler(commonReconciler),
		deploymentClient, serviceClient, podMinitorClient, configMapClient, k8sClient, statusUpdater, readyService, commonCRClient, crPriorityService, hpaClient
}

func getFacadeService(req reconcile.Request) *corev1.Service {
	templateFacadeService := &templates.FacadeService{
		Name:         "testName",
		Namespace:    req.Namespace,
		NameSelector: "gatewayName",
		Port:         1234,
	}

	return templateFacadeService.GetService()
}

func getFacadeDeployment(req reconcile.Request, gatewayName string, image string) *v1.Deployment {
	deployment := templates.RouterDeployment{
		ServiceName:                "serviceName",
		GatewayName:                gatewayName,
		NameSpace:                  req.Namespace,
		InstanceLabel:              "instanceLabelValue",
		ArtifactDescriptionVersion: "ArtifactDescriptorVersion",
		ImageName:                  image,
		Recourses:                  corev1.ResourceRequirements{},
		TracingEnabled:             "TracingEnabled",
		TracingHost:                "TracingHost",
		IpStack:                    "IpStack",
		IpBind:                     "IpBind",
		MeshRouter:                 false,
		Replicas:                   1,
	}

	return deployment.GetDeployment()
}

func TestFacadeServiceReconciler_getFacadeReplicas(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	configloader.Init(configloader.EnvPropertySource())

	rec := FacadeCommonReconciler{
		logger: logging.GetLogger("FacadeServiceReconciler"),
	}

	facadeService := &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: "2",
		},
	}
	res := rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(2), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: 2,
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(2), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: 0,
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(1), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: int32(2),
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(2), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: int32(0),
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(1), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: int64(2),
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(2), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: int64(0),
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(1), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: nil,
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(1), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: "",
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(1), res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: true,
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(1), res)

	utils.DefaultFacadeGatewayReplicas = "6"
	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Replicas: true,
		},
	}
	res = rec.getFacadeReplicas(context.Background(), facadeService)
	assert.Equal(t, int32(6), res)
}

func TestFacadeServiceReconciler_getFacadeGatewayConcurrency(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	configloader.Init(configloader.EnvPropertySource())
	rec := FacadeCommonReconciler{
		logger: logging.GetLogger("FacadeServiceReconciler"),
	}

	facadeService := &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Env: facade.FacadeServiceEnv{
				FacadeGatewayConcurrency: 1,
			},
		},
	}
	res := rec.getFacadeGatewayConcurrency(context.Background(), facadeService)
	assert.Equal(t, 1, res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Env: facade.FacadeServiceEnv{
				FacadeGatewayConcurrency: int32(32),
			},
		},
	}
	res = rec.getFacadeGatewayConcurrency(context.Background(), facadeService)
	assert.Equal(t, 32, res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Env: facade.FacadeServiceEnv{
				FacadeGatewayConcurrency: int64(64),
			},
		},
	}
	res = rec.getFacadeGatewayConcurrency(context.Background(), facadeService)
	assert.Equal(t, 64, res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Env: facade.FacadeServiceEnv{
				FacadeGatewayConcurrency: 0,
			},
		},
	}
	res = rec.getFacadeGatewayConcurrency(context.Background(), facadeService)
	assert.Equal(t, 0, res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Env: facade.FacadeServiceEnv{
				FacadeGatewayConcurrency: -1,
			},
		},
	}
	res = rec.getFacadeGatewayConcurrency(context.Background(), facadeService)
	assert.Equal(t, 0, res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Env: facade.FacadeServiceEnv{
				FacadeGatewayConcurrency: "5",
			},
		},
	}
	res = rec.getFacadeGatewayConcurrency(context.Background(), facadeService)
	assert.Equal(t, 5, res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Env: facade.FacadeServiceEnv{
				FacadeGatewayConcurrency: "",
			},
		},
	}
	res = rec.getFacadeGatewayConcurrency(context.Background(), facadeService)
	assert.Equal(t, 0, res)

	facadeService = &facadeV1Alpha.FacadeService{
		Spec: facade.FacadeServiceSpec{
			Env: facade.FacadeServiceEnv{},
		},
	}
	res = rec.getFacadeGatewayConcurrency(context.Background(), facadeService)
	assert.Equal(t, 0, res)
}
