// Code generated by MockGen. DO NOT EDIT.
// Source: interface.go

// Package mock_syncer is a generated GoMock package.
package mock_syncer

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	syncer "github.com/meshplus/pier/internal/syncer"
)

// MockSyncer is a mock of Syncer interface.
type MockSyncer struct {
	ctrl     *gomock.Controller
	recorder *MockSyncerMockRecorder
}

// MockSyncerMockRecorder is the mock recorder for MockSyncer.
type MockSyncerMockRecorder struct {
	mock *MockSyncer
}

// NewMockSyncer creates a new mock instance.
func NewMockSyncer(ctrl *gomock.Controller) *MockSyncer {
	mock := &MockSyncer{ctrl: ctrl}
	mock.recorder = &MockSyncerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSyncer) EXPECT() *MockSyncerMockRecorder {
	return m.recorder
}

// Start mocks base method.
func (m *MockSyncer) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockSyncerMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockSyncer)(nil).Start))
}

// Stop mocks base method.
func (m *MockSyncer) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockSyncerMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockSyncer)(nil).Stop))
}

// RegisterIBTPHandler mocks base method.
func (m *MockSyncer) RegisterIBTPHandler(handler syncer.IBTPHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterIBTPHandler", handler)
	ret0, _ := ret[0].(error)
	return ret0
}

// RegisterIBTPHandler indicates an expected call of RegisterIBTPHandler.
func (mr *MockSyncerMockRecorder) RegisterIBTPHandler(handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterIBTPHandler", reflect.TypeOf((*MockSyncer)(nil).RegisterIBTPHandler), handler)
}

// RegisterAppchainHandler mocks base method.
func (m *MockSyncer) RegisterAppchainHandler(handler syncer.AppchainHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterAppchainHandler", handler)
	ret0, _ := ret[0].(error)
	return ret0
}

// RegisterAppchainHandler indicates an expected call of RegisterAppchainHandler.
func (mr *MockSyncerMockRecorder) RegisterAppchainHandler(handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterAppchainHandler", reflect.TypeOf((*MockSyncer)(nil).RegisterAppchainHandler), handler)
}

// RegisterRecoverHandler mocks base method.
func (m *MockSyncer) RegisterRecoverHandler(handleRecover syncer.RecoverUnionHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterRecoverHandler", handleRecover)
	ret0, _ := ret[0].(error)
	return ret0
}

// RegisterRecoverHandler indicates an expected call of RegisterRecoverHandler.
func (mr *MockSyncerMockRecorder) RegisterRecoverHandler(handleRecover interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterRecoverHandler", reflect.TypeOf((*MockSyncer)(nil).RegisterRecoverHandler), handleRecover)
}
