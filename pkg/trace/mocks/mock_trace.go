// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/trace/trace.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	config "github.com/skatteetaten/architect/v2/pkg/config"
	runtime "github.com/skatteetaten/architect/v2/pkg/config/runtime"
	docker "github.com/skatteetaten/architect/v2/pkg/docker"
)

// MockTrace is a mock of Trace interface.
type MockTrace struct {
	ctrl     *gomock.Controller
	recorder *MockTraceMockRecorder
}

// MockTraceMockRecorder is the mock recorder for MockTrace.
type MockTraceMockRecorder struct {
	mock *MockTrace
}

// NewMockTrace creates a new mock instance.
func NewMockTrace(ctrl *gomock.Controller) *MockTrace {
	mock := &MockTrace{ctrl: ctrl}
	mock.recorder = &MockTraceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTrace) EXPECT() *MockTraceMockRecorder {
	return m.recorder
}

// AddBaseImageMetadata mocks base method.
func (m *MockTrace) SendBaseImageMetadata(application config.ApplicationSpec, imageInfo *runtime.ImageInfo, containerConfig *docker.ContainerConfig) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendBaseImageMetadata", application, imageInfo, containerConfig)
}

// AddBaseImageMetadata indicates an expected call of AddBaseImageMetadata.
func (mr *MockTraceMockRecorder) AddBaseImageMetadata(application, imageInfo, containerConfig interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendBaseImageMetadata", reflect.TypeOf((*MockTrace)(nil).SendBaseImageMetadata), application, imageInfo, containerConfig)
}

// AddImageMetadata mocks base method.
func (m *MockTrace) SendImageMetadata(data interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendImageMetadata", data)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddImageMetadata indicates an expected call of AddImageMetadata.
func (mr *MockTraceMockRecorder) AddImageMetadata(data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendImageMetadata", reflect.TypeOf((*MockTrace)(nil).SendImageMetadata), data)
}
