// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/docker/registry.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	io "io"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	runtime "github.com/skatteetaten/architect/v2/pkg/config/runtime"
	docker "github.com/skatteetaten/architect/v2/pkg/docker"
)

// MockRegistry is a mock of Registry interface.
type MockRegistry struct {
	ctrl     *gomock.Controller
	recorder *MockRegistryMockRecorder
}

func (m *MockRegistry) DownloadArtifact(c *config.MavenGav) (nexus.Deliverable, error) {
	//TODO implement me
	panic("implement me")
}

// MockRegistryMockRecorder is the mock recorder for MockRegistry.
type MockRegistryMockRecorder struct {
	mock *MockRegistry
}

// NewMockRegistry creates a new mock instance.
func NewMockRegistry(ctrl *gomock.Controller) *MockRegistry {
	mock := &MockRegistry{ctrl: ctrl}
	mock.recorder = &MockRegistryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegistry) EXPECT() *MockRegistryMockRecorder {
	return m.recorder
}

// GetContainerConfig mocks base method.
func (m *MockRegistry) GetContainerConfig(ctx context.Context, repository, digest string) (*docker.ContainerConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContainerConfig", ctx, repository, digest)
	ret0, _ := ret[0].(*docker.ContainerConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetContainerConfig indicates an expected call of GetContainerConfig.
func (mr *MockRegistryMockRecorder) GetContainerConfig(ctx, repository, digest interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContainerConfig", reflect.TypeOf((*MockRegistry)(nil).GetContainerConfig), ctx, repository, digest)
}

// GetImageConfig mocks base method.
func (m *MockRegistry) GetImageConfig(ctx context.Context, repository, digest string) (map[string]interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImageConfig", ctx, repository, digest)
	ret0, _ := ret[0].(map[string]interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImageConfig indicates an expected call of GetImageConfig.
func (mr *MockRegistryMockRecorder) GetImageConfig(ctx, repository, digest interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImageConfig", reflect.TypeOf((*MockRegistry)(nil).GetImageConfig), ctx, repository, digest)
}

// GetImageInfo mocks base method.
func (m *MockRegistry) GetImageInfo(ctx context.Context, repository, tag string) (*runtime.ImageInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImageInfo", ctx, repository, tag)
	ret0, _ := ret[0].(*runtime.ImageInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImageInfo indicates an expected call of GetImageInfo.
func (mr *MockRegistryMockRecorder) GetImageInfo(ctx, repository, tag interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImageInfo", reflect.TypeOf((*MockRegistry)(nil).GetImageInfo), ctx, repository, tag)
}

// GetManifest mocks base method.
func (m *MockRegistry) GetManifest(ctx context.Context, repository, digest string) (*docker.ManifestV2, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetManifest", ctx, repository, digest)
	ret0, _ := ret[0].(*docker.ManifestV2)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetManifest indicates an expected call of GetManifest.
func (mr *MockRegistryMockRecorder) GetManifest(ctx, repository, digest interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetManifest", reflect.TypeOf((*MockRegistry)(nil).GetManifest), ctx, repository, digest)
}

// GetTags mocks base method.
func (m *MockRegistry) GetTags(ctx context.Context, repository string) (*docker.TagsAPIResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTags", ctx, repository)
	ret0, _ := ret[0].(*docker.TagsAPIResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTags indicates an expected call of GetTags.
func (mr *MockRegistryMockRecorder) GetTags(ctx, repository interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTags", reflect.TypeOf((*MockRegistry)(nil).GetTags), ctx, repository)
}

// LayerExists mocks base method.
func (m *MockRegistry) LayerExists(ctx context.Context, repository, layerDigest string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LayerExists", ctx, repository, layerDigest)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LayerExists indicates an expected call of LayerExists.
func (mr *MockRegistryMockRecorder) LayerExists(ctx, repository, layerDigest interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LayerExists", reflect.TypeOf((*MockRegistry)(nil).LayerExists), ctx, repository, layerDigest)
}

// MountLayer mocks base method.
func (m *MockRegistry) MountLayer(ctx context.Context, srcRepository, dstRepository, layerDigest string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MountLayer", ctx, srcRepository, dstRepository, layerDigest)
	ret0, _ := ret[0].(error)
	return ret0
}

// MountLayer indicates an expected call of MountLayer.
func (mr *MockRegistryMockRecorder) MountLayer(ctx, srcRepository, dstRepository, layerDigest interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MountLayer", reflect.TypeOf((*MockRegistry)(nil).MountLayer), ctx, srcRepository, dstRepository, layerDigest)
}

// PullLayer mocks base method.
func (m *MockRegistry) PullLayer(ctx context.Context, repository, layerDigest string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PullLayer", ctx, repository, layerDigest)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PullLayer indicates an expected call of PullLayer.
func (mr *MockRegistryMockRecorder) PullLayer(ctx, repository, layerDigest interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PullLayer", reflect.TypeOf((*MockRegistry)(nil).PullLayer), ctx, repository, layerDigest)
}

// PushLayer mocks base method.
func (m *MockRegistry) PushLayer(ctx context.Context, layer io.Reader, dstRepository, layerDigest string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PushLayer", ctx, layer, dstRepository, layerDigest)
	ret0, _ := ret[0].(error)
	return ret0
}

// PushLayer indicates an expected call of PushLayer.
func (mr *MockRegistryMockRecorder) PushLayer(ctx, layer, dstRepository, layerDigest interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushLayer", reflect.TypeOf((*MockRegistry)(nil).PushLayer), ctx, layer, dstRepository, layerDigest)
}

// PushManifest mocks base method.
func (m *MockRegistry) PushManifest(ctx context.Context, manifest []byte, repository, tag string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PushManifest", ctx, manifest, repository, tag)
	ret0, _ := ret[0].(error)
	return ret0
}

// PushManifest indicates an expected call of PushManifest.
func (mr *MockRegistryMockRecorder) PushManifest(ctx, manifest, repository, tag interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushManifest", reflect.TypeOf((*MockRegistry)(nil).PushManifest), ctx, manifest, repository, tag)
}
