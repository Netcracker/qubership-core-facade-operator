package services

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"go.uber.org/mock/gomock"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestGetMasterCR(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	unknownErr := getUnknownError()
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(unknownErr)
	masterCR, err := deploymentClient.GetMasterCR(testContext, req, deployment.Name)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
	assert.Equal(t, "", masterCR)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			dp.ObjectMeta.ResourceVersion = "testResourceVersion"
			dp.Labels = make(map[string]string)
			dp.Labels[utils.MasterCR] = "test1"
			return nil
		})
	masterCR, err = deploymentClient.GetMasterCR(testContext, req, deployment.Name)
	assert.Nil(t, err)
	assert.Equal(t, "test1", masterCR)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			dp.ObjectMeta.ResourceVersion = "testResourceVersion"
			dp.Spec.Template.Labels = make(map[string]string)
			dp.Spec.Template.Labels[utils.MasterCR] = "test1"
			return nil
		})
	masterCR, err = deploymentClient.GetMasterCR(testContext, req, deployment.Name)
	assert.Nil(t, err)
	assert.Equal(t, "test1", masterCR)
}

func TestRemoveMasterCRLabel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	unknownErr := getUnknownError()
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(unknownErr)
	err := deploymentClient.DeleteMasterCRLabel(testContext, req, deployment.Name)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			dp.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	err = deploymentClient.DeleteMasterCRLabel(testContext, req, deployment.Name)
	assert.Nil(t, err)

	deployment.ObjectMeta.Labels = make(map[string]string)
	deployment.ObjectMeta.Labels[utils.MasterCR] = "1"
	deployment.Spec.Template.Labels = make(map[string]string)
	deployment.Spec.Template.Labels[utils.MasterCR] = "1"
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			*dp = *deployment
			dp.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Update(testContext, deployment, getUpdateOptions()).Times(0).Return(nil)
	k8sClient.EXPECT().Update(testContext, gomock.Any(), getUpdateOptions()).DoAndReturn(
		func(_ context.Context, dp *v1.Deployment, arg2 ...interface{}) error {
			assert.Equal(t, "", dp.ObjectMeta.Labels[utils.MasterCR])
			assert.Equal(t, "", dp.Spec.Template.Labels[utils.MasterCR])
			return nil
		})
	err = deploymentClient.DeleteMasterCRLabel(testContext, req, deployment.Name)
	assert.Nil(t, err)
}

func TestDeploymentGetMeshRouterDeploymentsNames_shouldFailed_whenErrorWhileListDeployments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{
			utils.MeshRouter: "true",
		}),
	}
	unknownErr := getUnknownError()

	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, opts).Return(unknownErr)

	deployments, err := deploymentClient.GetMeshRouterDeployments(testContext, req)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, "Failed to get facade gateway deployments", err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
	assert.Nil(t, deployments)
}

func TestDeploymentGetMeshRouterDeploymentsNames_shouldReturnNil_whenNoFoundDeployments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{
			utils.MeshRouter: "true",
		}),
	}
	foundDeployments := &v1.DeploymentList{}

	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, opts).DoAndReturn(
		func(_ context.Context, dpl *v1.DeploymentList, _ ...client.ListOption) error {
			*dpl = *foundDeployments
			return nil
		})

	deployments, err := deploymentClient.GetMeshRouterDeployments(testContext, req)
	assert.Nil(t, err)
	assert.Nil(t, deployments.Items)
}

func TestDeploymentGetMeshRouterDeploymentsNames_shouldReturnNames_whenNoErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, _ := getFacadeDeployment(req)
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{
			utils.MeshRouter: "true",
		}),
	}
	secondDeployment, _ := getFacadeDeploymentWithName(req, "secondName")
	foundDeployments := &v1.DeploymentList{
		Items: []v1.Deployment{
			*deployment,
			*secondDeployment,
		},
	}

	k8sClient.EXPECT().List(testContext, &v1.DeploymentList{}, opts).DoAndReturn(
		func(_ context.Context, dpl *v1.DeploymentList, _ ...client.ListOption) error {
			*dpl = *foundDeployments
			return nil
		})

	deployments, err := deploymentClient.GetMeshRouterDeployments(testContext, req)
	assert.Nil(t, err)
	assert.NotNil(t, deployments)
	assert.Equal(t, deployments, foundDeployments)
}

func TestDeploymentDelete_shouldNotFailed_whenIsNotFoundErrorWhileDeleting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			*dp = *deployment
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, deployment).Return(getNotFoundError())

	err := deploymentClient.Delete(testContext, req, deployment.Name)
	assert.Nil(t, err)
}

func TestDeploymentDelete_shouldFailed_whenUnknownErrorWhileDeleting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			*dp = *deployment
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, deployment).Return(unknownErr)

	err := deploymentClient.Delete(testContext, req, deployment.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to delete deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestDeploymentDelete_shouldNotDelete_whenDeploymentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Delete(testContext, deployment).Return(nil).Times(0)

	err := deploymentClient.Delete(testContext, req, deployment.Name)
	assert.Nil(t, err)
}

func TestDeploymentDelete_shouldFailed_whenUnknownErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(unknownErr)
	k8sClient.EXPECT().Delete(testContext, deployment).Return(nil).Times(0)

	err := deploymentClient.Delete(testContext, req, deployment.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestDeploymentDelete_shouldDelete_whenNoErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			*dp = *deployment
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, deployment).Return(nil)

	err := deploymentClient.Delete(testContext, req, deployment.Name)
	assert.Nil(t, err)
}

func TestDeploymentApply_shouldReturnExpectedError_whenIsAlreadyExistsErrorWhileCreating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, deployment, getCreateOptions()).Return(getAlreadyExistError())

	err := deploymentClient.Apply(testContext, req, deployment)
	assert.NotNil(t, err)
	switch e := err.(type) {
	default:
		assert.Equal(t, &customerrors.ExpectedError{}, e)
	}
}

func TestDeploymentApply_shouldReturnExpectedError_whenIsConflictErrorWhileUpdating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			dp.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Update(testContext, deployment, getUpdateOptions()).Return(getConflictError())

	err := deploymentClient.Apply(testContext, req, deployment)
	assert.NotNil(t, err)
	switch e := err.(type) {
	default:
		assert.Equal(t, &customerrors.ExpectedError{}, e)
	}
}

func TestDeploymentApply_shouldFailed_whenUnknownErrorWhileCreating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, deployment, getCreateOptions()).Return(unknownErr)

	err := deploymentClient.Apply(testContext, req, deployment)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to create deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestDeploymentApply_shouldFailed_whenUnknownErrorWhileUpdating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			dp.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Update(testContext, deployment, getUpdateOptions()).Return(unknownErr)

	err := deploymentClient.Apply(testContext, req, deployment)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to update deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestDeploymentApply_shouldFailed_whenUnknownErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(unknownErr)
	k8sClient.EXPECT().Create(testContext, deployment, getCreateOptions()).Return(nil).Times(0)
	k8sClient.EXPECT().Update(testContext, deployment, getUpdateOptions()).Return(nil).Times(0)

	err := deploymentClient.Apply(testContext, req, deployment)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestDeploymentApply_shouldCreate_whenNoErrorsAndDeploymentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, deployment, getCreateOptions()).Return(nil)

	err := deploymentClient.Apply(testContext, req, deployment)
	assert.Nil(t, err)
}

func TestDeploymentApply_shouldUpdate_whenNoErrorsAndDeploymentExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			dp.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Update(testContext, deployment, getUpdateOptions()).Return(nil)

	err := deploymentClient.Apply(testContext, req, deployment)
	assert.Nil(t, err)
}

func TestDeploymentIsFacadeGateway_shouldReturnFalseAndError_whenUnknownErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(unknownErr)

	isFacade, err := deploymentClient.IsFacadeGateway(testContext, req, deployment.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
	assert.False(t, isFacade)
}

func TestDeploymentIsFacadeGateway_shouldReturnFalse_whenFacadeGatewayNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()
	notFoundErr := getNotFoundError()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(notFoundErr)

	isFacade, err := deploymentClient.IsFacadeGateway(testContext, req, deployment.Name)
	assert.Nil(t, err)
	assert.False(t, isFacade)
}

func TestDeploymentIsFacadeGateway_shouldReturnFalse_whenFacadeGatewayLabelFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)
	deployment.Spec.Template.ObjectMeta.Labels[utils.FacadeGateway] = "false"
	deployment.Labels[utils.FacadeGateway] = "false"

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			*dp = *deployment
			return nil
		})

	isFacade, err := deploymentClient.IsFacadeGateway(testContext, req, deployment.Name)
	assert.Nil(t, err)
	assert.False(t, isFacade)
}

func TestDeploymentIsFacadeGateway_shouldReturnFalse_whenCompositeGateway(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)
	deployment.Spec.Template.ObjectMeta.Labels[utils.FacadeGateway] = "false"
	deployment.Labels[utils.FacadeGateway] = "false"
	deployment.Spec.Template.ObjectMeta.Labels[utils.MeshRouter] = "true"
	deployment.Labels[utils.MeshRouter] = "true"

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			*dp = *deployment
			return nil
		})

	isFacade, err := deploymentClient.IsFacadeGateway(testContext, req, deployment.Name)
	assert.Nil(t, err)
	assert.False(t, isFacade)
}

func TestDeploymentIsFacadeGateway_shouldReturnTrue_whenFacadeGatewayLabelTrue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			*dp = *deployment
			return nil
		})

	isFacade, err := deploymentClient.IsFacadeGateway(testContext, req, deployment.Name)
	assert.Nil(t, err)
	assert.True(t, isFacade)
}

func TestDeploymentGet_shouldFailed_whenErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(unknownErr)

	foundDeployment, err := deploymentClient.Get(testContext, req, deployment.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get deployment: %v", deployment.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
	assert.Nil(t, foundDeployment)
}

func TestDeploymentGet_shouldReturnNil_whenDeploymentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).Return(getNotFoundError())

	foundDeployment, err := deploymentClient.Get(testContext, req, deployment.Name)
	assert.Nil(t, err)
	assert.Nil(t, foundDeployment)
}

func TestDeploymentGet_shouldReturnDeployment_whenExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient, k8sClient := getDeploymentClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	deployment, nameSpacedRequest := getFacadeDeployment(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &v1.Deployment{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, dp *v1.Deployment, _ *client.GetOptions) error {
			*dp = *deployment
			return nil
		})

	foundDeployment, err := deploymentClient.Get(testContext, req, deployment.Name)
	assert.Nil(t, err)
	assert.Equal(t, deployment, foundDeployment)
}

func getDeploymentClient(ctrl *gomock.Controller) (DeploymentClient, *mock_client.MockClient) {
	k8sClient := GetMockClient(ctrl)
	return NewDeploymentClientImpl(k8sClient), k8sClient
}

func getFacadeDeployment(req reconcile.Request) (*v1.Deployment, types.NamespacedName) {
	return getFacadeDeploymentWithName(req, "gatewayName")
}

func getFacadeDeploymentWithName(req reconcile.Request, gatewayName string) (*v1.Deployment, types.NamespacedName) {
	deployment := templates.RouterDeployment{
		ServiceName:                "serviceName",
		GatewayName:                gatewayName,
		NameSpace:                  req.Namespace,
		InstanceLabel:              "instanceLabelValue",
		ArtifactDescriptionVersion: "ArtifactDescriptorVersion",
		ImageName:                  "publicGatewayImage",
		Recourses:                  corev1.ResourceRequirements{},
		IpStack:                    "IpStack",
		IpBind:                     "IpBind",
		MeshRouter:                 false,
		Replicas:                   1,
	}
	nameSpacedRequest := types.NamespacedName{
		Namespace: deployment.NameSpace,
		Name:      deployment.GatewayName,
	}

	return deployment.GetDeployment(), nameSpacedRequest
}
