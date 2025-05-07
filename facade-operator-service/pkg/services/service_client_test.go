package services

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	mock_client "github.com/netcracker/qubership-core-facade-operator/test/mock/client"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestServiceApply_shouldDeleteAndCreate_whenServiceHaveIncorrectTypeHeadLess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	os.Setenv("K8S_SERVICE_TYPE", "CLUSTER_IP")
	utils.ReloadServiceType()
	defer func() {
		os.Unsetenv("K8S_SERVICE_TYPE")
		utils.ReloadServiceType()
	}()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			fs.Name = facadeService.Name
			fs.Spec.ClusterIP = ""
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, facadeService).Return(nil)
	k8sClient.EXPECT().Create(testContext, facadeService, getCreateOptions()).Return(nil)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.Nil(t, err)
}

func TestServiceApply_shouldDeleteAndCreate_whenServiceHaveIncorrectTypeClusterIp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	os.Setenv("K8S_SERVICE_TYPE", "HEADLESS")
	utils.ReloadServiceType()
	defer func() {
		os.Unsetenv("K8S_SERVICE_TYPE")
		utils.ReloadServiceType()
	}()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			fs.Name = facadeService.Name
			fs.Spec.ClusterIP = "testClusterIP"
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, facadeService).Return(nil)
	k8sClient.EXPECT().Create(testContext, facadeService, getCreateOptions()).Return(nil)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldNotFailed_whenServiceDeletingFailedWithIsNotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	notFoundError := getNotFoundError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, facadeService).Return(notFoundError)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldFailed_whenServiceDeletingFailedWithUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, facadeService).Return(unknownErr)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to delete service %v", facadeService.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestServiceDelete_shouldNotFailed_whenServiceGettingReturnIsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	notFoundError := getNotFoundError()

	req := getRequest()
	_, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).Return(notFoundError)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldReturnError_whenServiceGettingFailedWithUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	_, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).Return(unknownErr)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get service %v", req.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestServiceDelete_shouldDelete_whenNoOneErrorFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, facadeService).Return(nil)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.Nil(t, err)
}

func TestServiceApply_shouldReturnExpectedError_whenServiceUpdateReturnConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	conflictErr := getConflictError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			fs.Spec.ClusterIP = "testClusterIP"
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Update(testContext, facadeService, getUpdateOptions()).Return(conflictErr)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.NotNil(t, err)
	switch e := err.(type) {
	default:
		assert.Equal(t, &customerrors.ExpectedError{}, e)
	}
}

func TestServiceApply_shouldReturnError_whenServiceUpdateFailedWithUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			fs.Spec.ClusterIP = "testClusterIP"
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Update(testContext, facadeService, getUpdateOptions()).Return(unknownErr)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to update service %v", facadeService.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestServiceApply_shouldReturnExpectedError_whenServiceCreateReturnIsAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	alreadyExistErr := getAlreadyExistError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, facadeService, getCreateOptions()).Return(alreadyExistErr)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.NotNil(t, err)
	switch e := err.(type) {
	default:
		assert.Equal(t, &customerrors.ExpectedError{}, e)
	}
}

func TestServiceApply_shouldReturnError_whenServiceCreateFailedWithUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, facadeService, getCreateOptions()).Return(unknownErr)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to create service %v", facadeService.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestServiceApply_shouldReturnError_whenGettingServiceReturnsUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()
	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).Return(unknownErr)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get service %v", facadeService.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestServiceApply_shouldCreate_whenServiceNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, facadeService, getCreateOptions()).Return(nil)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.Nil(t, err)
}

func TestServiceApply_shouldUpdate_whenServiceExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			fs.Spec.ClusterIP = "testClusterIP"
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})
	k8sClient.EXPECT().Update(testContext, facadeService, getUpdateOptions()).Return(nil)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.Nil(t, err)
}

func getFacadeService(req reconcile.Request) (*corev1.Service, types.NamespacedName) {
	templateFacadeService := &templates.FacadeService{
		Name:         "testName",
		Namespace:    req.Namespace,
		NameSelector: "gatewayName",
		Port:         1234,
	}
	facadeService := templateFacadeService.GetService()
	nameSpacedRequest := types.NamespacedName{
		Namespace: templateFacadeService.Namespace,
		Name:      templateFacadeService.Name,
	}

	return facadeService, nameSpacedRequest
}

func getRequest() reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "testName",
		},
	}
}

func getServiceClient(ctrl *gomock.Controller) (ServiceClient, *mock_client.MockClient) {
	k8sClient := GetMockClient(ctrl)
	return NewServiceClient(k8sClient), k8sClient
}
