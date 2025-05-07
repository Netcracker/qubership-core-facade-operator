package services

import (
	"context"
	"github.com/netcracker/qubership-core-facade-operator/api/facade"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/api/facade/v1"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	mock_client "github.com/netcracker/qubership-core-facade-operator/test/mock/client"
	mock_services "github.com/netcracker/qubership-core-facade-operator/test/mock/services"
	"go.uber.org/mock/gomock"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type testCheckDeploymentReadyStruct struct {
	name          string
	facadeService facade.MeshGateway
	mockFunc      func()
	errorDetail   string
	result        ctrl.Result
}

func TestCheckDeploymentReady(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	readyService, _, deploymentClientMock, statusUpdater := getReadyService(ctrl)

	readyDeployment := &v1.Deployment{
		Status: v1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			UnavailableReplicas: 0,
		},
	}

	scaledUpDeployment := &v1.Deployment{
		Status: v1.DeploymentStatus{
			Replicas:            2,
			ReadyReplicas:       1,
			UnavailableReplicas: 0,
		},
	}

	failedDeployment := &v1.Deployment{
		Status: v1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			UnavailableReplicas: 1,
		},
	}

	tests := []testCheckDeploymentReadyStruct{
		{
			name:          "ReadyFacade",
			facadeService: getCRWithName("ReadyFacade", ""),
			mockFunc: func() {
				deploymentClientMock.EXPECT().Get(gomock.Any(), gomock.Any(), "ReadyFacade"+utils.GatewaySuffix).Return(readyDeployment, nil).AnyTimes()
				statusUpdater.EXPECT().SetUpdated(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			result: reconcile.Result{},
		},
		{
			name:          "ReadyComposite",
			facadeService: getCRWithName("name", "ReadyComposite"),
			mockFunc: func() {
				deploymentClientMock.EXPECT().Get(gomock.Any(), gomock.Any(), "ReadyComposite").Return(readyDeployment, nil).AnyTimes()
				statusUpdater.EXPECT().SetUpdated(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			result: reconcile.Result{},
		},
		{
			name:          "RequeueWhenScaledUpDeployment",
			facadeService: getCRWithName("name", "RequeueWhenScaledUpDeployment"),
			mockFunc: func() {
				deploymentClientMock.EXPECT().Get(gomock.Any(), gomock.Any(), "RequeueWhenScaledUpDeployment").Return(scaledUpDeployment, nil).AnyTimes()
				statusUpdater.EXPECT().SetUpdated(gomock.Any(), gomock.Any()).Times(0)
			},
			result: reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second},
		},
		{
			name:          "RequeueWhenFailedDeployment",
			facadeService: getCRWithName("name", "RequeueWhenFailedDeployment"),
			mockFunc: func() {
				deploymentClientMock.EXPECT().Get(gomock.Any(), gomock.Any(), "RequeueWhenFailedDeployment").Return(failedDeployment, nil).AnyTimes()
				statusUpdater.EXPECT().SetUpdated(gomock.Any(), gomock.Any()).Times(0)
			},
			result: reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockFunc != nil {
				tt.mockFunc()
			}

			result, err := readyService.CheckDeploymentReady(context.Background(), reconcile.Request{}, tt.facadeService)
			if tt.errorDetail != "" {
				assert.NotNil(t, err)
				assert.Equal(t, tt.errorDetail, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.result, result)
			}
		})
	}
}

func getCRWithName(crName string, gatewayName string) *facadeV1.Gateway {
	return &facadeV1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		},
		Spec: facade.FacadeServiceSpec{
			Gateway: gatewayName,
		},
	}
}

type testReadyServiceStruct struct {
	name          string
	facadeService *facadeV1.Gateway
	result        bool
}

func TestIsUpdatingPhase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	readyService, _, _, _ := getReadyService(ctrl)

	tests := []testReadyServiceStruct{
		{
			name:          "Success",
			facadeService: getCRWithGeneration(1, 1, facadeV1.UpdatingPhase),
			result:        true,
		},
		{
			name:          "DiffGenerations",
			facadeService: getCRWithGeneration(1, 0, facadeV1.UpdatingPhase),
			result:        false,
		},
		{
			name:          "DiffPhase",
			facadeService: getCRWithGeneration(1, 0, facadeV1.BackingOffPhase),
			result:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readyService.IsUpdatingPhase(context.Background(), reconcile.Request{}, tt.facadeService)
			assert.Equal(t, tt.result, result)
		})
	}
}

func getCRWithGeneration(objectGeneration, statusGeneration int64, phase facadeV1.Phase) *facadeV1.Gateway {
	return &facadeV1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "name",
			Generation: objectGeneration,
		},
		Status: facadeV1.GatewayStatus{
			ObservedGeneration: statusGeneration,
			Phase:              phase,
		},
	}
}

func getReadyService(ctrl *gomock.Controller) (ReadyService, *mock_client.MockClient, *mock_services.MockDeploymentClient, *mock_services.MockStatusUpdater) {
	k8sClient := GetMockClient(ctrl)
	deploymentClient := getDeploymentClientMock(ctrl)
	statusUpdater := getStatusUpdaterMock(ctrl)
	return NewReadyService(deploymentClient, statusUpdater), k8sClient, deploymentClient, statusUpdater
}

func getDeploymentClientMock(ctrl *gomock.Controller) *mock_services.MockDeploymentClient {
	return mock_services.NewMockDeploymentClient(ctrl)
}

func getStatusUpdaterMock(ctrl *gomock.Controller) *mock_services.MockStatusUpdater {
	return mock_services.NewMockStatusUpdater(ctrl)
}
