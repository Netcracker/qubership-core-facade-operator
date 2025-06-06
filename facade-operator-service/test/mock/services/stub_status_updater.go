// Code generated by MockGen. DO NOT EDIT.
// Source: ../../pkg/services/status_updater.go
//
// Generated by this command:
//
//	mockgen -source=../../pkg/services/status_updater.go -destination=./services/stub_status_updater.go -package=mock_services
//

// Package mock_services is a generated GoMock package.
package mock_services

import (
	context "context"
	reflect "reflect"

	facade "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	gomock "go.uber.org/mock/gomock"
)

// MockStatusUpdater is a mock of StatusUpdater interface.
type MockStatusUpdater struct {
	ctrl     *gomock.Controller
	recorder *MockStatusUpdaterMockRecorder
	isgomock struct{}
}

// MockStatusUpdaterMockRecorder is the mock recorder for MockStatusUpdater.
type MockStatusUpdaterMockRecorder struct {
	mock *MockStatusUpdater
}

// NewMockStatusUpdater creates a new mock instance.
func NewMockStatusUpdater(ctrl *gomock.Controller) *MockStatusUpdater {
	mock := &MockStatusUpdater{ctrl: ctrl}
	mock.recorder = &MockStatusUpdaterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStatusUpdater) EXPECT() *MockStatusUpdaterMockRecorder {
	return m.recorder
}

// SetFail mocks base method.
func (m *MockStatusUpdater) SetFail(ctx context.Context, resource facade.MeshGateway) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetFail", ctx, resource)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetFail indicates an expected call of SetFail.
func (mr *MockStatusUpdaterMockRecorder) SetFail(ctx, resource any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetFail", reflect.TypeOf((*MockStatusUpdater)(nil).SetFail), ctx, resource)
}

// SetUpdated mocks base method.
func (m *MockStatusUpdater) SetUpdated(ctx context.Context, resource facade.MeshGateway) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetUpdated", ctx, resource)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetUpdated indicates an expected call of SetUpdated.
func (mr *MockStatusUpdaterMockRecorder) SetUpdated(ctx, resource any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetUpdated", reflect.TypeOf((*MockStatusUpdater)(nil).SetUpdated), ctx, resource)
}

// SetUpdating mocks base method.
func (m *MockStatusUpdater) SetUpdating(ctx context.Context, resource facade.MeshGateway) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetUpdating", ctx, resource)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetUpdating indicates an expected call of SetUpdating.
func (mr *MockStatusUpdaterMockRecorder) SetUpdating(ctx, resource any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetUpdating", reflect.TypeOf((*MockStatusUpdater)(nil).SetUpdating), ctx, resource)
}
