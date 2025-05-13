package services

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestConfigMap_GetGatewayImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      CoreGatewayImageConfigMap,
	}

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).
		DoAndReturn(func(_ context.Context, nameSpacedRequest types.NamespacedName, foundConfigMap *corev1.ConfigMap, _ *client.GetOptions) error {
			foundConfigMap.Data = map[string]string{"image": "test-mage:latest"}
			return nil
		})
	img, err := configMapClient.GetGatewayImage(testContext, req)
	assert.Nil(t, err)
	assert.Equal(t, "test-mage:latest", img)
}

func TestConfigMap_GetGatewayImage_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      CoreGatewayImageConfigMap,
	}

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).Return(getNotFoundError())
	img, err := configMapClient.GetGatewayImage(testContext, req)
	assert.Nil(t, err)
	assert.Empty(t, img)
}

func TestConfigMap_GetGatewayImage_UnknownErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()
	req := getRequest()

	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      CoreGatewayImageConfigMap,
	}

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).Return(unknownErr)
	_, err := configMapClient.GetGatewayImage(testContext, req)
	assert.NotNil(t, err)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestConfigMapApply_shouldFailed_whenUnknownErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	configMap, nameSpacedRequest := getFacadeConfigMap(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).Return(unknownErr)
	k8sClient.EXPECT().Create(testContext, configMap, getCreateOptions()).Return(nil).Times(0)

	err := configMapClient.Apply(testContext, req, configMap)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get ConfigMap %v", configMap.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestConfigMapApply_shouldReturnExpectedError_whenIsAlreadyExistsErrorWhileCreating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()
	alreadyExistErr := getAlreadyExistError()

	req := getRequest()
	configMap, nameSpacedRequest := getFacadeConfigMap(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, configMap, getCreateOptions()).Return(alreadyExistErr)

	err := configMapClient.Apply(testContext, req, configMap)
	assert.NotNil(t, err)
	switch e := err.(type) {
	default:
		assert.Equal(t, &customerrors.ExpectedError{}, e)
	}
}

func TestConfigMapApply_shouldFailed_whenUnknownErrorWhileCreating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	configMap, nameSpacedRequest := getFacadeConfigMap(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, configMap, getCreateOptions()).Return(unknownErr)

	err := configMapClient.Apply(testContext, req, configMap)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to create ConfigMap %v", configMap.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestConfigMapApply_shouldCreate_whenExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	configMap, nameSpacedRequest := getFacadeConfigMap(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, configMap, getCreateOptions()).Return(nil)

	err := configMapClient.Apply(testContext, req, configMap)
	assert.Nil(t, err)
}

func TestConfigMapApply_shouldReturnExpectedError_whenIsConflictErrorWhileUpdating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()
	conflictErr := getConflictError()

	req := getRequest()
	configMap, nameSpacedRequest := getFacadeConfigMap(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, cm *corev1.ConfigMap, _ *client.GetOptions) error {
			configMap.ObjectMeta.ResourceVersion = "ResourceVersion"
			*cm = *configMap
			return nil
		})
	k8sClient.EXPECT().Update(testContext, configMap, getUpdateOptions()).Return(conflictErr)

	err := configMapClient.Apply(testContext, req, configMap)
	assert.NotNil(t, err)
	switch e := err.(type) {
	default:
		assert.Equal(t, &customerrors.ExpectedError{}, e)
	}
}

func TestConfigMapApply_shouldFailed_whenUnknownErrorWhileUpdating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	configMap, nameSpacedRequest := getFacadeConfigMap(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, cm *corev1.ConfigMap, _ *client.GetOptions) error {
			configMap.ObjectMeta.ResourceVersion = "ResourceVersion"
			*cm = *configMap
			return nil
		})
	k8sClient.EXPECT().Update(testContext, configMap, getUpdateOptions()).Return(unknownErr)

	err := configMapClient.Apply(testContext, req, configMap)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to update ConfigMap %v", configMap.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestConfigMapApply_shouldUpdate_whenExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMapClient, k8sClient := getConfigMapClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	configMap, nameSpacedRequest := getFacadeConfigMap(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &corev1.ConfigMap{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, cm *corev1.ConfigMap, _ *client.GetOptions) error {
			configMap.ObjectMeta.ResourceVersion = "ResourceVersion"
			*cm = *configMap
			return nil
		})
	k8sClient.EXPECT().Update(testContext, configMap, getUpdateOptions()).Return(nil)

	err := configMapClient.Apply(testContext, req, configMap)
	assert.Nil(t, err)
}

func getConfigMapClient(ctrl *gomock.Controller) (ConfigMapClient, *mock_client.MockClient) {
	k8sClient := GetMockClient(ctrl)
	return NewConfigMapClient(k8sClient), k8sClient
}

func getFacadeConfigMap(req reconcile.Request) (*corev1.ConfigMap, types.NamespacedName) {
	configMap := templates.FacadeConfigMap{
		Name:      "Name",
		Namespace: req.Namespace,
	}
	nameSpacedRequest := types.NamespacedName{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}

	return configMap.GetConfigMap(), nameSpacedRequest
}
