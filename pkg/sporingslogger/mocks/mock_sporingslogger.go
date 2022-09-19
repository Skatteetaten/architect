// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/sporingslogger/sporingslogger.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	config "github.com/skatteetaten/architect/v2/pkg/config"
	runtime "github.com/skatteetaten/architect/v2/pkg/config/runtime"
	docker "github.com/skatteetaten/architect/v2/pkg/docker"
	sporingslogger "github.com/skatteetaten/architect/v2/pkg/sporingslogger"
)

// MockSporingslogger is a mock of Sporingslogger interface.
type MockSporingslogger struct {
	ctrl     *gomock.Controller
	recorder *MockSporingsloggerMockRecorder
}

// MockSporingsloggerMockRecorder is the mock recorder for MockSporingslogger.
type MockSporingsloggerMockRecorder struct {
	mock *MockSporingslogger
}

// NewMockSporingslogger creates a new mock instance.
func NewMockSporingslogger(ctrl *gomock.Controller) *MockSporingslogger {
	mock := &MockSporingslogger{ctrl: ctrl}
	mock.recorder = &MockSporingsloggerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSporingslogger) EXPECT() *MockSporingsloggerMockRecorder {
	return m.recorder
}

// ScanImage mocks base method.
func (m *MockSporingslogger) ScanImage(buildFolder string) ([]sporingslogger.Dependency, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ScanImage", buildFolder)
	ret0, _ := ret[0].([]sporingslogger.Dependency)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ScanImage indicates an expected call of ScanImage.
func (mr *MockSporingsloggerMockRecorder) ScanImage(buildFolder interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ScanImage", reflect.TypeOf((*MockSporingslogger)(nil).ScanImage), buildFolder)
}

// SendBaseImageMetadata mocks base method.
func (m *MockSporingslogger) SendBaseImageMetadata(application config.ApplicationSpec, imageInfo *runtime.ImageInfo, containerConfig *docker.ContainerConfig) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendBaseImageMetadata", application, imageInfo, containerConfig)
}

// SendBaseImageMetadata indicates an expected call of SendBaseImageMetadata.
func (mr *MockSporingsloggerMockRecorder) SendBaseImageMetadata(application, imageInfo, containerConfig interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendBaseImageMetadata", reflect.TypeOf((*MockSporingslogger)(nil).SendBaseImageMetadata), application, imageInfo, containerConfig)
}

// SendImageMetadata mocks base method.
func (m *MockSporingslogger) SendImageMetadata(data interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendImageMetadata", data)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendImageMetadata indicates an expected call of SendImageMetadata.
func (mr *MockSporingsloggerMockRecorder) SendImageMetadata(data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendImageMetadata", reflect.TypeOf((*MockSporingslogger)(nil).SendImageMetadata), data)
}