// Code generated by MockGen. DO NOT EDIT.
// Source: service.go

// Package mock_entity is a generated GoMock package.
package mock_entity

import (
	context "context"
	reflect "reflect"
	time "time"

	core "github.com/totegamma/concurrent/x/core"
	gomock "go.uber.org/mock/gomock"
)

// MockService is a mock of Service interface.
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
}

// MockServiceMockRecorder is the mock recorder for MockService.
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance.
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// Affiliation mocks base method.
func (m *MockService) Affiliation(ctx context.Context, document, signature, meta string) (core.Entity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Affiliation", ctx, document, signature, meta)
	ret0, _ := ret[0].(core.Entity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Affiliation indicates an expected call of Affiliation.
func (mr *MockServiceMockRecorder) Affiliation(ctx, document, signature, meta interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Affiliation", reflect.TypeOf((*MockService)(nil).Affiliation), ctx, document, signature, meta)
}

// Count mocks base method.
func (m *MockService) Count(ctx context.Context) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Count", ctx)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Count indicates an expected call of Count.
func (mr *MockServiceMockRecorder) Count(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Count", reflect.TypeOf((*MockService)(nil).Count), ctx)
}

// Delete mocks base method.
func (m *MockService) Delete(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockServiceMockRecorder) Delete(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockService)(nil).Delete), ctx, id)
}

// Get mocks base method.
func (m *MockService) Get(ctx context.Context, ccid string) (core.Entity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, ccid)
	ret0, _ := ret[0].(core.Entity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockServiceMockRecorder) Get(ctx, ccid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockService)(nil).Get), ctx, ccid)
}

// GetAddress mocks base method.
func (m *MockService) GetAddress(ctx context.Context, ccid string) (core.Address, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAddress", ctx, ccid)
	ret0, _ := ret[0].(core.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAddress indicates an expected call of GetAddress.
func (mr *MockServiceMockRecorder) GetAddress(ctx, ccid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAddress", reflect.TypeOf((*MockService)(nil).GetAddress), ctx, ccid)
}

// IsUserExists mocks base method.
func (m *MockService) IsUserExists(ctx context.Context, user string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsUserExists", ctx, user)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsUserExists indicates an expected call of IsUserExists.
func (mr *MockServiceMockRecorder) IsUserExists(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsUserExists", reflect.TypeOf((*MockService)(nil).IsUserExists), ctx, user)
}

// List mocks base method.
func (m *MockService) List(ctx context.Context) ([]core.Entity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx)
	ret0, _ := ret[0].([]core.Entity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockServiceMockRecorder) List(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockService)(nil).List), ctx)
}

// ListModified mocks base method.
func (m *MockService) ListModified(ctx context.Context, modified time.Time) ([]core.Entity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModified", ctx, modified)
	ret0, _ := ret[0].([]core.Entity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListModified indicates an expected call of ListModified.
func (mr *MockServiceMockRecorder) ListModified(ctx, modified interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModified", reflect.TypeOf((*MockService)(nil).ListModified), ctx, modified)
}

// PullEntityFromRemote mocks base method.
func (m *MockService) PullEntityFromRemote(ctx context.Context, id, domain string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PullEntityFromRemote", ctx, id, domain)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PullEntityFromRemote indicates an expected call of PullEntityFromRemote.
func (mr *MockServiceMockRecorder) PullEntityFromRemote(ctx, id, domain interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PullEntityFromRemote", reflect.TypeOf((*MockService)(nil).PullEntityFromRemote), ctx, id, domain)
}

// ResolveHost mocks base method.
func (m *MockService) ResolveHost(ctx context.Context, user, hint string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveHost", ctx, user, hint)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveHost indicates an expected call of ResolveHost.
func (mr *MockServiceMockRecorder) ResolveHost(ctx, user, hint interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveHost", reflect.TypeOf((*MockService)(nil).ResolveHost), ctx, user, hint)
}

// Tombstone mocks base method.
func (m *MockService) Tombstone(ctx context.Context, document, signature string) (core.Entity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Tombstone", ctx, document, signature)
	ret0, _ := ret[0].(core.Entity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Tombstone indicates an expected call of Tombstone.
func (mr *MockServiceMockRecorder) Tombstone(ctx, document, signature interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Tombstone", reflect.TypeOf((*MockService)(nil).Tombstone), ctx, document, signature)
}

// Update mocks base method.
func (m *MockService) Update(ctx context.Context, entity *core.Entity) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, entity)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockServiceMockRecorder) Update(ctx, entity interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockService)(nil).Update), ctx, entity)
}

// UpdateAddress mocks base method.
func (m *MockService) UpdateAddress(ctx context.Context, ccid, domain string, signedAt time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAddress", ctx, ccid, domain, signedAt)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateAddress indicates an expected call of UpdateAddress.
func (mr *MockServiceMockRecorder) UpdateAddress(ctx, ccid, domain, signedAt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAddress", reflect.TypeOf((*MockService)(nil).UpdateAddress), ctx, ccid, domain, signedAt)
}
