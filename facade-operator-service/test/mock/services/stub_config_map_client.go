// Code generated by MockGen. DO NOT EDIT.
// Source: ../../pkg/services/config_map_client.go
//
// Generated by this command:
//
//	mockgen -source=../../pkg/services/config_map_client.go -destination=./services/stub_config_map_client.go -package=mock_services
//

// Package mock_services is a generated GoMock package.
package mock_services

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// MockConfigMapClient is a mock of ConfigMapClient interface.
type MockConfigMapClient struct {
	ctrl     *gomock.Controller
	recorder *MockConfigMapClientMockRecorder
	isgomock struct{}
}

// MockConfigMapClientMockRecorder is the mock recorder for MockConfigMapClient.
type MockConfigMapClientMockRecorder struct {
	mock *MockConfigMapClient
}

// NewMockConfigMapClient creates a new mock instance.
func NewMockConfigMapClient(ctrl *gomock.Controller) *MockConfigMapClient {
	mock := &MockConfigMapClient{ctrl: ctrl}
	mock.recorder = &MockConfigMapClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConfigMapClient) EXPECT() *MockConfigMapClientMockRecorder {
	return m.recorder
}

// Apply mocks base method.
func (m *MockConfigMapClient) Apply(ctx context.Context, req controllerruntime.Request, configMap *v1.ConfigMap) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Apply", ctx, req, configMap)
	ret0, _ := ret[0].(error)
	return ret0
}

// Apply indicates an expected call of Apply.
func (mr *MockConfigMapClientMockRecorder) Apply(ctx, req, configMap any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Apply", reflect.TypeOf((*MockConfigMapClient)(nil).Apply), ctx, req, configMap)
}

// Delete mocks base method.
func (m *MockConfigMapClient) Delete(ctx context.Context, req controllerruntime.Request, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, req, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockConfigMapClientMockRecorder) Delete(ctx, req, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockConfigMapClient)(nil).Delete), ctx, req, name)
}

// GetGatewayImage mocks base method.
func (m *MockConfigMapClient) GetGatewayImage(ctx context.Context, req controllerruntime.Request) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGatewayImage", ctx, req)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGatewayImage indicates an expected call of GetGatewayImage.
func (mr *MockConfigMapClientMockRecorder) GetGatewayImage(ctx, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGatewayImage", reflect.TypeOf((*MockConfigMapClient)(nil).GetGatewayImage), ctx, req)
}
