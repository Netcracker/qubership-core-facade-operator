// Code generated by MockGen. DO NOT EDIT.
// Source: ../../pkg/services/pod_monitor_client.go
//
// Generated by this command:
//
//	mockgen -source=../../pkg/services/pod_monitor_client.go -destination=./services/stub_pod_monitor_client.go -package=mock_services
//

// Package mock_services is a generated GoMock package.
package mock_services

import (
	context "context"
	reflect "reflect"

	v1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/monitoring/v1"
	gomock "go.uber.org/mock/gomock"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// MockPodMonitorClient is a mock of PodMonitorClient interface.
type MockPodMonitorClient struct {
	ctrl     *gomock.Controller
	recorder *MockPodMonitorClientMockRecorder
	isgomock struct{}
}

// MockPodMonitorClientMockRecorder is the mock recorder for MockPodMonitorClient.
type MockPodMonitorClientMockRecorder struct {
	mock *MockPodMonitorClient
}

// NewMockPodMonitorClient creates a new mock instance.
func NewMockPodMonitorClient(ctrl *gomock.Controller) *MockPodMonitorClient {
	mock := &MockPodMonitorClient{ctrl: ctrl}
	mock.recorder = &MockPodMonitorClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPodMonitorClient) EXPECT() *MockPodMonitorClientMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockPodMonitorClient) Create(ctx context.Context, req controllerruntime.Request, podMonitor *v1.PodMonitor) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, req, podMonitor)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockPodMonitorClientMockRecorder) Create(ctx, req, podMonitor any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockPodMonitorClient)(nil).Create), ctx, req, podMonitor)
}

// Delete mocks base method.
func (m *MockPodMonitorClient) Delete(ctx context.Context, req controllerruntime.Request, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, req, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockPodMonitorClientMockRecorder) Delete(ctx, req, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockPodMonitorClient)(nil).Delete), ctx, req, name)
}
