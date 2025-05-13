package controllers

import (
	"errors"
	mock_client "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/client"
	mock_services "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/services"
	"go.uber.org/mock/gomock"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getNotFoundError() error {
	return &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Status: "404",
			Code:   404,
			Reason: metav1.StatusReasonNotFound,
		},
	}
}

func getUnknownError() error {
	return errors.New("UnknownError")
}

func getConflictError() error {
	return &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Status: "409",
			Code:   409,
			Reason: metav1.StatusReasonConflict,
		},
	}
}

func GetMockDeploymentClient(ctrl *gomock.Controller) *mock_services.MockDeploymentClient {
	return mock_services.NewMockDeploymentClient(ctrl)
}

func GetMockServiceClient(ctrl *gomock.Controller) *mock_services.MockServiceClient {
	return mock_services.NewMockServiceClient(ctrl)
}

func GetMockConfigMapClient(ctrl *gomock.Controller) *mock_services.MockConfigMapClient {
	return mock_services.NewMockConfigMapClient(ctrl)
}

func GetMockPodMonitorClient(ctrl *gomock.Controller) *mock_services.MockPodMonitorClient {
	return mock_services.NewMockPodMonitorClient(ctrl)
}

func GetMockHPAClient(ctrl *gomock.Controller) *mock_services.MockHPAClient {
	return mock_services.NewMockHPAClient(ctrl)
}

func GetMockClient(ctrl *gomock.Controller) *mock_client.MockClient {
	return mock_client.NewMockClient(ctrl)
}

func GetMockStatusUpdater(ctrl *gomock.Controller) *mock_services.MockStatusUpdater {
	return mock_services.NewMockStatusUpdater(ctrl)
}

func GetMockReadyService(ctrl *gomock.Controller) *mock_services.MockReadyService {
	return mock_services.NewMockReadyService(ctrl)
}

func GetMockCommonCRClient(ctrl *gomock.Controller) *mock_services.MockCommonCRClient {
	return mock_services.NewMockCommonCRClient(ctrl)
}

func GetMockCRPriorityService(ctrl *gomock.Controller) *mock_services.MockCRPriorityService {
	return mock_services.NewMockCRPriorityService(ctrl)
}
