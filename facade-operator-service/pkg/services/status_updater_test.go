package services

import (
	"context"
	"github.com/netcracker/qubership-core-facade-operator/api/facade"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/api/facade/v1"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	mock_client "github.com/netcracker/qubership-core-facade-operator/test/mock/client"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"go.uber.org/mock/gomock"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type testStatusUpdaterStruct struct {
	name        string
	errorDetail string
	facdeCR     facade.MeshGateway
	mockFunc    func()
	executeFunc func(ctx context.Context, resource facade.MeshGateway) error
}

func TestUpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	statusUpdater, k8sClient := getStatusUpdater(ctrl)

	subResourceWriterMock := getMockSubResourceWriter(ctrl)
	k8sClient.EXPECT().Status().Return(subResourceWriterMock).AnyTimes()

	facdeCR := getFacadeCR()

	tests := []testStatusUpdaterStruct{
		{
			name:        "NilCR",
			facdeCR:     nil,
			executeFunc: statusUpdater.SetUpdating,
		},
		{
			name:    "SetUpdatingSuccess",
			facdeCR: facdeCR,
			mockFunc: func() {
				subResourceWriterMock.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, obj client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
						assert.Equal(t, int64(1), obj.(*facadeV1.Gateway).Status.ObservedGeneration)
						assert.Equal(t, facadeV1.UpdatingPhase, obj.(*facadeV1.Gateway).Status.Phase)
						return nil
					})
			},
			executeFunc: statusUpdater.SetUpdating,
		},
		{
			name:    "SetUpdatedSuccess",
			facdeCR: facdeCR,
			mockFunc: func() {
				subResourceWriterMock.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, obj client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
						assert.Equal(t, int64(1), obj.(*facadeV1.Gateway).Status.ObservedGeneration)
						assert.Equal(t, facadeV1.UpdatedPhase, obj.(*facadeV1.Gateway).Status.Phase)
						return nil
					})
			},
			executeFunc: statusUpdater.SetUpdated,
		},
		{
			name:    "SetFailSuccess",
			facdeCR: facdeCR,
			mockFunc: func() {
				subResourceWriterMock.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, obj client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
						assert.Equal(t, int64(1), obj.(*facadeV1.Gateway).Status.ObservedGeneration)
						assert.Equal(t, facadeV1.BackingOffPhase, obj.(*facadeV1.Gateway).Status.Phase)
						return nil
					})
			},
			executeFunc: statusUpdater.SetFail,
		},
		{
			name:        "WithoutKind",
			facdeCR:     &facadeV1.Gateway{},
			errorDetail: "Unknown error while unmarshal CR",
			executeFunc: statusUpdater.SetUpdating,
		},
		{
			name:        "ErrorWhilePatch",
			facdeCR:     facdeCR,
			errorDetail: "Unknown error while patching CR",
			mockFunc: func() {
				subResourceWriterMock.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, obj client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
						return errs.NewError(customerrors.UnexpectedKubernetesError, "testError", nil)
					})
			},
			executeFunc: statusUpdater.SetUpdating,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockFunc != nil {
				tt.mockFunc()
			}

			// err := statusUpdater.SetUpdating(ctx, tt.facdeCR)
			err := tt.executeFunc(ctx, tt.facdeCR)
			if tt.errorDetail != "" {
				assert.NotNil(t, err)
				assert.Equal(t, tt.errorDetail, err.(*errs.ErrCodeError).Detail)
			} else {
				assert.Nil(t, err)
			}

		})
	}
}

func getFacadeCR() *facadeV1.Gateway {
	return &facadeV1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FacadeService",
			APIVersion: "qubership.org/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "name",
			Generation: 1,
		},
		Spec: facade.FacadeServiceSpec{
			Port:    8080,
			Gateway: "gateway",
		},
	}
}

func getStatusUpdater(ctrl *gomock.Controller) (StatusUpdater, *mock_client.MockClient) {
	k8sClient := GetMockClient(ctrl)
	return NewStatusUpdater(k8sClient), k8sClient
}

func getMockSubResourceWriter(ctrl *gomock.Controller) *mock_client.MockSubResourceWriter {
	return mock_client.NewMockSubResourceWriter(ctrl)
}
