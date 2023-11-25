// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/autom8ter/grpcx/providers (interfaces: Stream)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	providers "github.com/autom8ter/grpcx/providers"
	gomock "github.com/golang/mock/gomock"
)

// MockStream is a mock of Stream interface.
type MockStream struct {
	ctrl     *gomock.Controller
	recorder *MockStreamMockRecorder
}

// MockStreamMockRecorder is the mock recorder for MockStream.
type MockStreamMockRecorder struct {
	mock *MockStream
}

// NewMockStream creates a new mock instance.
func NewMockStream(ctrl *gomock.Controller) *MockStream {
	mock := &MockStream{ctrl: ctrl}
	mock.recorder = &MockStreamMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStream) EXPECT() *MockStreamMockRecorder {
	return m.recorder
}

// AsyncSubscribe mocks base method.
func (m *MockStream) AsyncSubscribe(arg0 context.Context, arg1, arg2 string, arg3 providers.MessageHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AsyncSubscribe", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// AsyncSubscribe indicates an expected call of AsyncSubscribe.
func (mr *MockStreamMockRecorder) AsyncSubscribe(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AsyncSubscribe", reflect.TypeOf((*MockStream)(nil).AsyncSubscribe), arg0, arg1, arg2, arg3)
}

// Publish mocks base method.
func (m *MockStream) Publish(arg0 context.Context, arg1 string, arg2 map[string]interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Publish", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Publish indicates an expected call of Publish.
func (mr *MockStreamMockRecorder) Publish(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Publish", reflect.TypeOf((*MockStream)(nil).Publish), arg0, arg1, arg2)
}

// Subscribe mocks base method.
func (m *MockStream) Subscribe(arg0 context.Context, arg1, arg2 string, arg3 providers.MessageHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// Subscribe indicates an expected call of Subscribe.
func (mr *MockStreamMockRecorder) Subscribe(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockStream)(nil).Subscribe), arg0, arg1, arg2, arg3)
}
