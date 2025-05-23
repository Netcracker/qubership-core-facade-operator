// Code generated by MockGen. DO NOT EDIT.
// Source: ../../pkg/services/cr_priority_service.go
//
// Generated by this command:
//
//	mockgen -source=../../pkg/services/cr_priority_service.go -destination=./services/stub_cr_priority_service.go -package=mock_services
//

// Package mock_services is a generated GoMock package.
package mock_services

import (
	context "context"
	reflect "reflect"

	facade "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	gomock "go.uber.org/mock/gomock"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// MockCRPriorityService is a mock of CRPriorityService interface.
type MockCRPriorityService struct {
	ctrl     *gomock.Controller
	recorder *MockCRPriorityServiceMockRecorder
	isgomock struct{}
}

// MockCRPriorityServiceMockRecorder is the mock recorder for MockCRPriorityService.
type MockCRPriorityServiceMockRecorder struct {
	mock *MockCRPriorityService
}

// NewMockCRPriorityService creates a new mock instance.
func NewMockCRPriorityService(ctrl *gomock.Controller) *MockCRPriorityService {
	mock := &MockCRPriorityService{ctrl: ctrl}
	mock.recorder = &MockCRPriorityServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCRPriorityService) EXPECT() *MockCRPriorityServiceMockRecorder {
	return m.recorder
}

// UpdateAvailable mocks base method.
func (m *MockCRPriorityService) UpdateAvailable(ctx context.Context, req controllerruntime.Request, gatewayName string, cr facade.MeshGateway) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAvailable", ctx, req, gatewayName, cr)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateAvailable indicates an expected call of UpdateAvailable.
func (mr *MockCRPriorityServiceMockRecorder) UpdateAvailable(ctx, req, gatewayName, cr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAvailable", reflect.TypeOf((*MockCRPriorityService)(nil).UpdateAvailable), ctx, req, gatewayName, cr)
}
