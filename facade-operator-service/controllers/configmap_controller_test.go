package controllers

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	mock_services "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/services"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"go.uber.org/mock/gomock"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile_shouldFailed_whenUnknownErrorWhileGetting(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, configMapClientMock, k8sClient := getConfigMapReconciler(mockCtrl)
	testContext := context.Background()
	req := getDeploymentRequest()
	unknownErr := getUnknownError()

	newImage := "newImage"
	configMapClientMock.EXPECT().GetGatewayImage(testContext, req).Return(newImage, nil)

	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, getDeploymentListOption(req)).Return(unknownErr)

	_, err := reconciler.Reconcile(testContext, req)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UpdateImageUnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, "Failed to get facade gateway deployments", err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestReconcile_shouldNotFailed_whenIsNotFoundErrorWhileGetting(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, configMapClientMock, k8sClient := getConfigMapReconciler(mockCtrl)
	testContext := context.Background()
	req := getDeploymentRequest()

	newImage := "newImage"
	configMapClientMock.EXPECT().GetGatewayImage(testContext, req).Return(newImage, nil)

	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, getDeploymentListOption(req)).Return(getNotFoundError())

	result, err := reconciler.Reconcile(testContext, req)
	assert.Nil(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestReconcile_shouldFailed_whenUnknownErrorFound(t *testing.T) {
	configloader.Init(configloader.EnvPropertySource())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, configMapClientMock, k8sClient := getConfigMapReconciler(mockCtrl)
	testContext := context.Background()
	req := getDeploymentRequest()
	unknownErr := getUnknownError()

	newImage := "newImage"
	oldImage := "oldImage"
	configMapClientMock.EXPECT().GetGatewayImage(testContext, req).Return(newImage, nil)

	facadeDeployments := getDeploymentList(req, oldImage)
	facadeDeploymentsWithNewImage := getDeploymentList(req, newImage)
	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, getDeploymentListOption(req)).DoAndReturn(
		func(_ context.Context, dpl *v1.DeploymentList, _ ...client.ListOption) error {
			*dpl = *facadeDeployments
			return nil
		})

	k8sClient.EXPECT().Update(testContext, &facadeDeploymentsWithNewImage.Items[0]).Return(unknownErr).Times(1)
	k8sClient.EXPECT().Update(testContext, &facadeDeploymentsWithNewImage.Items[1]).Return(nil).Times(0)

	_, err := reconciler.Reconcile(testContext, req)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UpdateImageUnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to update Deployment: %v", "gatewayName1"), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestReconcile_shouldRetry_whenConflictErrorFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, configMapClientMock, k8sClient := getConfigMapReconciler(mockCtrl)
	testContext := context.Background()
	req := getDeploymentRequest()

	newImage := "newImage"
	oldImage := "oldImage"
	configMapClientMock.EXPECT().GetGatewayImage(testContext, req).Return(newImage, nil)

	facadeDeployments := getDeploymentList(req, oldImage)
	facadeDeploymentsWithNewImage := getDeploymentList(req, newImage)
	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, getDeploymentListOption(req)).DoAndReturn(
		func(_ context.Context, dpl *v1.DeploymentList, _ ...client.ListOption) error {
			*dpl = *facadeDeployments
			return nil
		})

	k8sClient.EXPECT().Update(testContext, &facadeDeploymentsWithNewImage.Items[0]).Return(getConflictError()).Times(1)
	k8sClient.EXPECT().Update(testContext, &facadeDeploymentsWithNewImage.Items[1]).Return(nil).Times(1)

	result, err := reconciler.Reconcile(testContext, req)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ctrl.Result{Requeue: true}, result)
}

func TestReconcile_shouldNotUpdateImage_whenSameImages(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, configMapClientMock, k8sClient := getConfigMapReconciler(mockCtrl)
	testContext := context.Background()
	req := getDeploymentRequest()

	newImage := "newImage"
	configMapClientMock.EXPECT().GetGatewayImage(testContext, req).Return(newImage, nil)

	facadeDeployments := getDeploymentList(req, newImage)
	facadeDeploymentsWithNewImage := getDeploymentList(req, newImage)
	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, getDeploymentListOption(req)).DoAndReturn(
		func(_ context.Context, dpl *v1.DeploymentList, _ ...client.ListOption) error {
			*dpl = *facadeDeployments
			return nil
		})

	k8sClient.EXPECT().Update(testContext, &facadeDeploymentsWithNewImage.Items[0]).Return(nil).Times(0)
	k8sClient.EXPECT().Update(testContext, &facadeDeploymentsWithNewImage.Items[1]).Return(nil).Times(0)

	k8sClient.EXPECT().Update(testContext, &facadeDeployments.Items[0]).Return(nil).Times(0)
	k8sClient.EXPECT().Update(testContext, &facadeDeployments.Items[1]).Return(nil).Times(0)

	result, err := reconciler.Reconcile(testContext, req)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestReconcile_shouldUpdateImage_whenErrorsNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reconciler, configMapClientMock, k8sClient := getConfigMapReconciler(mockCtrl)
	testContext := context.Background()
	req := getDeploymentRequest()

	newImage := "newImage"
	oldImage := "oldImage"
	configMapClientMock.EXPECT().GetGatewayImage(testContext, req).Return(newImage, nil)

	facadeDeployments := getDeploymentList(req, oldImage)
	facadeDeploymentsWithNewImage := getDeploymentList(req, newImage)
	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, getDeploymentListOption(req)).DoAndReturn(
		func(_ context.Context, dpl *v1.DeploymentList, _ ...client.ListOption) error {
			*dpl = *facadeDeployments
			return nil
		})

	k8sClient.EXPECT().Update(testContext, &facadeDeploymentsWithNewImage.Items[0]).Return(nil).Times(1)
	k8sClient.EXPECT().Update(testContext, &facadeDeploymentsWithNewImage.Items[1]).Return(nil).Times(1)

	result, err := reconciler.Reconcile(testContext, req)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ctrl.Result{}, result)
}

func getDeploymentList(req reconcile.Request, image string) *v1.DeploymentList {
	return &v1.DeploymentList{
		Items: []v1.Deployment{
			*getFacadeDeployment(req, "gatewayName1", image),
			*getFacadeDeployment(req, "gatewayName2", image),
		},
	}
}

func getDeploymentListOption(req ctrl.Request) []client.ListOption {
	return []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{
			utils.FacadeGateway: "true",
		}),
	}
}

func getDeploymentRequest() reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "testName",
		},
	}
}

func getConfigMapReconciler(ctrl *gomock.Controller) (*ConfigMapReconciller, *mock_services.MockConfigMapClient, *mock_client.MockClient) {
	k8sClient := GetMockClient(ctrl)
	configMapClient := GetMockConfigMapClient(ctrl)
	return NewConfigMapReconciller(k8sClient, configMapClient), configMapClient, k8sClient
}
