package services

import (
	"errors"
	mock_client "github.com/netcracker/qubership-core-facade-operator/test/mock/client"
	mock_services "github.com/netcracker/qubership-core-facade-operator/test/mock/services"
	"go.uber.org/mock/gomock"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetMockCommonCRClient(ctrl *gomock.Controller) *mock_services.MockCommonCRClient {
	return mock_services.NewMockCommonCRClient(ctrl)
}

func GetMockClient(ctrl *gomock.Controller) *mock_client.MockClient {
	return mock_client.NewMockClient(ctrl)
}

func getUpdateOptions() *client.UpdateOptions {
	return &client.UpdateOptions{
		FieldManager: "facadeOperator",
	}
}

func getCreateOptions() *client.CreateOptions {
	return &client.CreateOptions{
		FieldManager: "facadeOperator",
	}
}

func getAlreadyExistError() error {
	return &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Status: "409",
			Code:   409,
			Reason: metav1.StatusReasonAlreadyExists,
		},
	}
}

func getNotFoundError() error {
	return &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Status: "404",
			Code:   404,
			Reason: metav1.StatusReasonNotFound,
		},
	}
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

func getUnknownError() error {
	return errors.New("UnknownError")
}
