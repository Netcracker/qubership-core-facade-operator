package services

import (
	"context"
	"fmt"
	"os"
	"testing"

	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	// First Get in Apply - sees wrong type
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			fs.Name = facadeService.Name
			fs.Spec.ClusterIP = ""
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: facadeService.Spec.Selector["app"],
			Labels: map[string]string{
				utils.FacadeGateway: "true",
			},
		},
	}

	// Get in Delete (called from Apply when recreating)
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(mockDeployment, nil)
	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(nil)

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
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	// First Get in Apply - sees wrong type
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			fs.Name = facadeService.Name
			fs.Spec.ClusterIP = "testClusterIP"
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: facadeService.Spec.Selector["app"],
			Labels: map[string]string{
				utils.FacadeGateway: "true",
			},
		},
	}

	// Get in Delete (called from Apply when recreating)
	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(mockDeployment, nil)
	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(nil)

	k8sClient.EXPECT().Create(testContext, facadeService, getCreateOptions()).Return(nil)

	err := serviceClient.Apply(testContext, req, facadeService)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldNotFailed_whenServiceDeletingFailedWithIsNotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()
	notFoundError := getNotFoundError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ResourceVersion = "testResourceVersion"
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: facadeService.Spec.Selector["app"],
			Labels: map[string]string{
				utils.FacadeGateway: "true",
			},
		},
	}
	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(mockDeployment, nil)

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(notFoundError)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldFailed_whenServiceDeletingFailedWithUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ResourceVersion = "testResourceVersion"
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: facadeService.Spec.Selector["app"],
			Labels: map[string]string{
				utils.FacadeGateway: "true",
			},
		},
	}
	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(mockDeployment, nil)

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(unknownErr)

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
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ResourceVersion = "testResourceVersion"
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: facadeService.Spec.Selector["app"],
			Labels: map[string]string{
				utils.FacadeGateway: "true",
			},
		},
	}
	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(mockDeployment, nil)

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(nil)

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

func TestServiceDelete_shouldSkip_whenServiceHasNoAppSelector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)
	facadeService.Spec.Selector = map[string]string{}

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			return nil
		})

	err := serviceClient.Delete(testContext, req, req.Name, "gatewayName")
	assert.Nil(t, err)
}

func TestServiceDelete_shouldFallbackToDeploymentValidation_whenSelectorDoesNotMatchExpectedGateway(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)
	facadeService.Spec.Selector["app"] = "actualGateway"

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "actualGateway",
			Labels: map[string]string{
				utils.FacadeGateway: "true",
			},
		},
	}
	deploymentClient.EXPECT().Get(testContext, req, "actualGateway").Return(mockDeployment, nil)

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(nil)

	err := serviceClient.Delete(testContext, req, req.Name, "differentGateway")
	assert.Nil(t, err)
}

func TestServiceDelete_shouldDelete_whenSelectorMatchesExpectedGateway(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(nil)

	err := serviceClient.Delete(testContext, req, req.Name, facadeService.Spec.Selector["app"])
	assert.Nil(t, err)
}

func TestServiceDelete_shouldDelete_whenDeploymentAlreadyDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(nil, getNotFoundError())

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(nil)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldDelete_whenDeploymentGetReturnsNilWithNilError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	// Simulate DeploymentClient.Get returning (nil, nil) - this is what happens for NotFound
	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(nil, nil)

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(nil)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldSkip_whenServiceSelectsNonFacadeDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   facadeService.Spec.Selector["app"],
			Labels: map[string]string{},
		},
	}
	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(mockDeployment, nil)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldDelete_whenServiceSelectsFacadeGateway(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ObjectMeta.ResourceVersion = "testResourceVersion"
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: facadeService.Spec.Selector["app"],
			Labels: map[string]string{
				utils.FacadeGateway: "true",
			},
		},
	}
	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(mockDeployment, nil)

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(nil)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.Nil(t, err)
}

func TestServiceDelete_shouldReturnExpectedError_whenDeleteReturnsConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	deploymentClient := GetMockDeploymentClient(ctrl)
	serviceClient = NewServiceClient(k8sClient, deploymentClient)
	testContext := context.Background()
	conflictErr := getConflictError()

	req := getRequest()
	facadeService, nameSpacedRequest := getFacadeService(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, fs *corev1.Service, _ *client.GetOptions) error {
			*fs = *facadeService
			fs.ResourceVersion = "testResourceVersion"
			return nil
		})

	mockDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: facadeService.Spec.Selector["app"],
			Labels: map[string]string{
				utils.FacadeGateway: "true",
			},
		},
	}
	deploymentClient.EXPECT().Get(testContext, req, facadeService.Spec.Selector["app"]).Return(mockDeployment, nil)

	k8sClient.EXPECT().Delete(testContext, gomock.Any(), getDeleteOptions("testResourceVersion")).Return(conflictErr)

	err := serviceClient.Delete(testContext, req, req.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to delete service %v", req.Name), err.(*errs.ErrCodeError).Detail)
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
	deploymentClient := GetMockDeploymentClient(ctrl)
	return NewServiceClient(k8sClient, deploymentClient), k8sClient
}

func getSharedService(req reconcile.Request) (*corev1.Service, types.NamespacedName) {
	tpl := &templates.FacadeService{
		Name:         "egress-gateway",
		Namespace:    req.Namespace,
		NameSelector: "egress-gateway-gateway",
		Port:         8080,
	}
	svc := tpl.GetSharedService()
	return svc, types.NamespacedName{Namespace: req.Namespace, Name: tpl.Name}
}

func getPatchOptions() *client.PatchOptions {
	return &client.PatchOptions{FieldManager: "facadeOperator"}
}

func TestEnsureShared_shouldCreate_whenServiceNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	ctx := context.Background()
	req := getRequest()
	svc, namespacedName := getSharedService(req)

	k8sClient.EXPECT().Get(ctx, namespacedName, &corev1.Service{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(ctx, gomock.AssignableToTypeOf(svc), getCreateOptions()).DoAndReturn(
		func(_ context.Context, created *corev1.Service, _ *client.CreateOptions) error {
			assert.Nil(t, created.OwnerReferences)
			return nil
		})

	err := serviceClient.EnsureShared(ctx, req, svc)
	assert.Nil(t, err)
}

func TestEnsureShared_shouldStripFacadeOwnerRef_whenFacadeRefPresent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	ctx := context.Background()
	req := getRequest()
	svc, namespacedName := getSharedService(req)

	facadeRef := metav1.OwnerReference{
		APIVersion: "netcracker.com/v1alpha",
		Kind:       "FacadeService",
		Name:       "egress-gateway",
		UID:        "uid-1",
	}
	foreignRef := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       "some-deployment",
		UID:        "uid-2",
	}
	// the live service is chart-shaped: istio selector and an assigned clusterIP, both
	// different from the template argument, so we can prove the spec is not overwritten
	foundSvc := svc.DeepCopy()
	foundSvc.OwnerReferences = []metav1.OwnerReference{facadeRef, foreignRef}
	foundSvc.ResourceVersion = "rv1"
	foundSvc.Spec.Selector = map[string]string{"gateway.networking.k8s.io/gateway-name": "egress-gateway"}
	foundSvc.Spec.ClusterIP = "10.96.0.42"
	chartSelector := foundSvc.Spec.Selector

	k8sClient.EXPECT().Get(ctx, namespacedName, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, s *corev1.Service, _ *client.GetOptions) error {
			*s = *foundSvc
			return nil
		})
	k8sClient.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(svc), gomock.Any(), getPatchOptions()).DoAndReturn(
		func(_ context.Context, patched *corev1.Service, _ client.Patch, _ *client.PatchOptions) error {
			assert.Equal(t, []metav1.OwnerReference{foreignRef}, patched.OwnerReferences)
			// the live spec must survive: we patch metadata only, never the chart's fields
			assert.Equal(t, chartSelector, patched.Spec.Selector)
			assert.Equal(t, "10.96.0.42", patched.Spec.ClusterIP)
			assert.NotEqual(t, svc.Spec.Selector, patched.Spec.Selector)
			return nil
		})

	err := serviceClient.EnsureShared(ctx, req, svc)
	assert.Nil(t, err)
}

func TestEnsureShared_shouldNoop_whenNoFacadeOwnerRefs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	ctx := context.Background()
	req := getRequest()
	svc, namespacedName := getSharedService(req)

	foundSvc := svc.DeepCopy()
	foundSvc.OwnerReferences = nil

	k8sClient.EXPECT().Get(ctx, namespacedName, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, s *corev1.Service, _ *client.GetOptions) error {
			*s = *foundSvc
			return nil
		})
	// No Create or Patch expected — gomock will fail the test if called.

	err := serviceClient.EnsureShared(ctx, req, svc)
	assert.Nil(t, err)
}

func TestEnsureShared_shouldReturnExpectedError_whenPatchConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	ctx := context.Background()
	req := getRequest()
	svc, namespacedName := getSharedService(req)

	foundSvc := svc.DeepCopy()
	foundSvc.OwnerReferences = []metav1.OwnerReference{
		{APIVersion: "netcracker.com/v1alpha", Kind: "FacadeService", Name: "egress-gateway", UID: "uid-1"},
	}
	foundSvc.ResourceVersion = "rv1"

	conflictErr := &k8serrors.StatusError{ErrStatus: metav1.Status{Status: "409", Code: 409, Reason: metav1.StatusReasonConflict}}

	k8sClient.EXPECT().Get(ctx, namespacedName, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, s *corev1.Service, _ *client.GetOptions) error {
			*s = *foundSvc
			return nil
		})
	k8sClient.EXPECT().Patch(ctx, gomock.Any(), gomock.Any(), getPatchOptions()).Return(conflictErr)

	err := serviceClient.EnsureShared(ctx, req, svc)
	assert.NotNil(t, err)
	assert.IsType(t, &customerrors.ExpectedError{}, err)
}

func TestEnsureShared_shouldReturnNil_whenPatchNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	ctx := context.Background()
	req := getRequest()
	svc, namespacedName := getSharedService(req)

	foundSvc := svc.DeepCopy()
	foundSvc.OwnerReferences = []metav1.OwnerReference{
		{APIVersion: "netcracker.com/v1alpha", Kind: "FacadeService", Name: "egress-gateway", UID: "uid-1"},
	}

	k8sClient.EXPECT().Get(ctx, namespacedName, &corev1.Service{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, s *corev1.Service, _ *client.GetOptions) error {
			*s = *foundSvc
			return nil
		})
	k8sClient.EXPECT().Patch(ctx, gomock.Any(), gomock.Any(), getPatchOptions()).Return(getNotFoundError())

	err := serviceClient.EnsureShared(ctx, req, svc)
	assert.Nil(t, err)
}

func TestEnsureShared_shouldReturnError_whenGetFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceClient, k8sClient := getServiceClient(ctrl)
	ctx := context.Background()
	req := getRequest()
	svc, namespacedName := getSharedService(req)

	k8sClient.EXPECT().Get(ctx, namespacedName, &corev1.Service{}, &client.GetOptions{}).Return(fmt.Errorf("unexpected error"))

	err := serviceClient.EnsureShared(ctx, req, svc)
	assert.NotNil(t, err)
	assert.IsType(t, &errs.ErrCodeError{}, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
}
