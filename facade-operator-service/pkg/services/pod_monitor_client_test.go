package services

import (
	"context"
	"fmt"
	monitoringV1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/monitoring/v1"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestPodMonitorDelete_shouldFailed_whenUnknownErrorWhileDeleting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, pm *monitoringV1.PodMonitor, _ *client.GetOptions) error {
			*pm = *podMonitor
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, podMonitor).Return(unknownErr)

	err := podMonitorClient.Delete(testContext, req, podMonitor.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to delete PodMonitor %v", podMonitor.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestPodMonitorDelete_shouldFailed_whenUnknownErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}, &client.GetOptions{}).Return(unknownErr)
	k8sClient.EXPECT().Delete(testContext, podMonitor).Return(nil).Times(0)

	err := podMonitorClient.Delete(testContext, req, podMonitor.Name)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get PodMonitor %v", podMonitor.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestPodMonitorDelete_shouldNotFailed_whenIsNotFoundErrorWhileDeleting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, pm *monitoringV1.PodMonitor, _ *client.GetOptions) error {
			*pm = *podMonitor
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, podMonitor).Return(getNotFoundError())

	err := podMonitorClient.Delete(testContext, req, podMonitor.Name)
	assert.Nil(t, err)
}

func TestPodMonitorDelete_shouldNotFailed_whenIsNotFoundErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}, &client.GetOptions{}).Return(getNotFoundError())
	k8sClient.EXPECT().Delete(testContext, podMonitor).Return(nil).Times(0)

	err := podMonitorClient.Delete(testContext, req, podMonitor.Name)
	assert.Nil(t, err)
}

func TestPodMonitorDelete_shouldDelete_whenNoErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}, &client.GetOptions{}).DoAndReturn(
		func(_ context.Context, _ types.NamespacedName, pm *monitoringV1.PodMonitor, _ *client.GetOptions) error {
			*pm = *podMonitor
			return nil
		})
	k8sClient.EXPECT().Delete(testContext, podMonitor).Return(nil)

	err := podMonitorClient.Delete(testContext, req, podMonitor.Name)
	assert.Nil(t, err)
}

func TestPodMonitorCreate_shouldNotFailed_whenIsAlreadyExistsErrorWhileCreating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}).Return(nil)
	k8sClient.EXPECT().Update(testContext, podMonitor, getUpdateOptions()).Return(nil)

	err := podMonitorClient.Create(testContext, req, podMonitor)
	assert.Nil(t, err)
}

func TestPodMonitorCreate_shouldFailed_whenUnknownErrorWhileCreating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, podMonitor, getCreateOptions()).Return(unknownErr)

	err := podMonitorClient.Create(testContext, req, podMonitor)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to create PodMonitor %v", podMonitor.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestPodMonitorCreate_shouldFailed_whenUnknownErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()
	unknownErr := getUnknownError()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}).Return(unknownErr)
	k8sClient.EXPECT().Create(testContext, podMonitor, getCreateOptions()).Return(nil).Times(0)

	err := podMonitorClient.Create(testContext, req, podMonitor)
	assert.NotNil(t, err)
	assert.Equal(t, customerrors.UnexpectedKubernetesError, err.(*errs.ErrCodeError).ErrorCode)
	assert.Equal(t, fmt.Sprintf("Failed to get PodMonitor %v", podMonitor.Name), err.(*errs.ErrCodeError).Detail)
	assert.Equal(t, unknownErr, err.(*errs.ErrCodeError).Cause)
}

func TestPodMonitorCreate_shouldNotFailed_whenIsAlreadyExistsErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}).Return(getAlreadyExistError())
	k8sClient.EXPECT().Update(testContext, podMonitor, getUpdateOptions()).Return(nil).Times(1)

	err := podMonitorClient.Create(testContext, req, podMonitor)
	assert.Nil(t, err)
}

func TestPodMonitorCreate_shouldCreate_whenIsNotFoundErrorWhileGetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}).Return(getNotFoundError())
	k8sClient.EXPECT().Create(testContext, podMonitor, getCreateOptions()).Return(nil)

	err := podMonitorClient.Create(testContext, req, podMonitor)
	assert.Nil(t, err)
}

func TestPodMonitorCreate_shouldUpdate_whenNoErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podMonitorClient, k8sClient := getPodMonitorClient(ctrl)
	testContext := context.Background()

	req := getRequest()
	podMonitor, nameSpacedRequest := getFacadePodMonitor(req)

	k8sClient.EXPECT().Get(testContext, nameSpacedRequest, &monitoringV1.PodMonitor{}).Return(nil)
	k8sClient.EXPECT().Update(testContext, podMonitor, getUpdateOptions()).Return(nil)

	err := podMonitorClient.Create(testContext, req, podMonitor)
	assert.Nil(t, err)
}

func getPodMonitorClient(ctrl *gomock.Controller) (PodMonitorClient, *mock_client.MockClient) {
	k8sClient := GetMockClient(ctrl)
	return NewPodMonitorClient(k8sClient), k8sClient
}

func getFacadePodMonitor(req reconcile.Request) (*monitoringV1.PodMonitor, types.NamespacedName) {
	podMonitor := &templates.FacadePodMonitor{
		Name:         "podMonitorName",
		NameSpace:    req.Namespace,
		NameLabel:    "podMonitorName",
		NameSelector: "name",
	}
	nameSpacedRequest := types.NamespacedName{
		Namespace: podMonitor.NameSpace,
		Name:      podMonitor.Name,
	}

	return podMonitor.GetPodMonitor(), nameSpacedRequest
}
