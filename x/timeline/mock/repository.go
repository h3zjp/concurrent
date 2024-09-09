// Code generated by MockGen. DO NOT EDIT.
// Source: repository.go
//
// Generated by this command:
//
//	mockgen -source=repository.go -destination=mock/repository.go
//

// Package mock_timeline is a generated GoMock package.
package mock_timeline

import (
	context "context"
	reflect "reflect"
	time "time"

	core "github.com/totegamma/concurrent/core"
	gomock "go.uber.org/mock/gomock"
)

// MockRepository is a mock of Repository interface.
type MockRepository struct {
	ctrl     *gomock.Controller
	recorder *MockRepositoryMockRecorder
}

// MockRepositoryMockRecorder is the mock recorder for MockRepository.
type MockRepositoryMockRecorder struct {
	mock *MockRepository
}

// NewMockRepository creates a new mock instance.
func NewMockRepository(ctrl *gomock.Controller) *MockRepository {
	mock := &MockRepository{ctrl: ctrl}
	mock.recorder = &MockRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRepository) EXPECT() *MockRepositoryMockRecorder {
	return m.recorder
}

// Count mocks base method.
func (m *MockRepository) Count(ctx context.Context) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Count", ctx)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Count indicates an expected call of Count.
func (mr *MockRepositoryMockRecorder) Count(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Count", reflect.TypeOf((*MockRepository)(nil).Count), ctx)
}

// CreateItem mocks base method.
func (m *MockRepository) CreateItem(ctx context.Context, item core.TimelineItem) (core.TimelineItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateItem", ctx, item)
	ret0, _ := ret[0].(core.TimelineItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateItem indicates an expected call of CreateItem.
func (mr *MockRepositoryMockRecorder) CreateItem(ctx, item any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateItem", reflect.TypeOf((*MockRepository)(nil).CreateItem), ctx, item)
}

// DeleteItem mocks base method.
func (m *MockRepository) DeleteItem(ctx context.Context, timelineID, objectID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteItem", ctx, timelineID, objectID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteItem indicates an expected call of DeleteItem.
func (mr *MockRepositoryMockRecorder) DeleteItem(ctx, timelineID, objectID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteItem", reflect.TypeOf((*MockRepository)(nil).DeleteItem), ctx, timelineID, objectID)
}

// DeleteItemByResourceID mocks base method.
func (m *MockRepository) DeleteItemByResourceID(ctx context.Context, resourceID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteItemByResourceID", ctx, resourceID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteItemByResourceID indicates an expected call of DeleteItemByResourceID.
func (mr *MockRepositoryMockRecorder) DeleteItemByResourceID(ctx, resourceID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteItemByResourceID", reflect.TypeOf((*MockRepository)(nil).DeleteItemByResourceID), ctx, resourceID)
}

// DeleteTimeline mocks base method.
func (m *MockRepository) DeleteTimeline(ctx context.Context, key string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTimeline", ctx, key)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTimeline indicates an expected call of DeleteTimeline.
func (mr *MockRepositoryMockRecorder) DeleteTimeline(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTimeline", reflect.TypeOf((*MockRepository)(nil).DeleteTimeline), ctx, key)
}

// GetImmediateItems mocks base method.
func (m *MockRepository) GetImmediateItems(ctx context.Context, timelineID string, since time.Time, limit int) ([]core.TimelineItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImmediateItems", ctx, timelineID, since, limit)
	ret0, _ := ret[0].([]core.TimelineItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImmediateItems indicates an expected call of GetImmediateItems.
func (mr *MockRepositoryMockRecorder) GetImmediateItems(ctx, timelineID, since, limit any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImmediateItems", reflect.TypeOf((*MockRepository)(nil).GetImmediateItems), ctx, timelineID, since, limit)
}

// GetItem mocks base method.
func (m *MockRepository) GetItem(ctx context.Context, timelineID, objectID string) (core.TimelineItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetItem", ctx, timelineID, objectID)
	ret0, _ := ret[0].(core.TimelineItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetItem indicates an expected call of GetItem.
func (mr *MockRepositoryMockRecorder) GetItem(ctx, timelineID, objectID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetItem", reflect.TypeOf((*MockRepository)(nil).GetItem), ctx, timelineID, objectID)
}

// GetMetrics mocks base method.
func (m *MockRepository) GetMetrics() map[string]int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMetrics")
	ret0, _ := ret[0].(map[string]int64)
	return ret0
}

// GetMetrics indicates an expected call of GetMetrics.
func (mr *MockRepositoryMockRecorder) GetMetrics() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMetrics", reflect.TypeOf((*MockRepository)(nil).GetMetrics))
}

// GetNormalizationCache mocks base method.
func (m *MockRepository) GetNormalizationCache(ctx context.Context, timelineID string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNormalizationCache", ctx, timelineID)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNormalizationCache indicates an expected call of GetNormalizationCache.
func (mr *MockRepositoryMockRecorder) GetNormalizationCache(ctx, timelineID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNormalizationCache", reflect.TypeOf((*MockRepository)(nil).GetNormalizationCache), ctx, timelineID)
}

// GetRecentItems mocks base method.
func (m *MockRepository) GetRecentItems(ctx context.Context, timelineID string, until time.Time, limit int) ([]core.TimelineItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRecentItems", ctx, timelineID, until, limit)
	ret0, _ := ret[0].([]core.TimelineItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRecentItems indicates an expected call of GetRecentItems.
func (mr *MockRepositoryMockRecorder) GetRecentItems(ctx, timelineID, until, limit any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRecentItems", reflect.TypeOf((*MockRepository)(nil).GetRecentItems), ctx, timelineID, until, limit)
}

// GetTimeline mocks base method.
func (m *MockRepository) GetTimeline(ctx context.Context, key string) (core.Timeline, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTimeline", ctx, key)
	ret0, _ := ret[0].(core.Timeline)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTimeline indicates an expected call of GetTimeline.
func (mr *MockRepositoryMockRecorder) GetTimeline(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTimeline", reflect.TypeOf((*MockRepository)(nil).GetTimeline), ctx, key)
}

// GetTimelineFromRemote mocks base method.
func (m *MockRepository) GetTimelineFromRemote(ctx context.Context, host, key string) (core.Timeline, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTimelineFromRemote", ctx, host, key)
	ret0, _ := ret[0].(core.Timeline)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTimelineFromRemote indicates an expected call of GetTimelineFromRemote.
func (mr *MockRepositoryMockRecorder) GetTimelineFromRemote(ctx, host, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTimelineFromRemote", reflect.TypeOf((*MockRepository)(nil).GetTimelineFromRemote), ctx, host, key)
}

// ListTimelineByAuthor mocks base method.
func (m *MockRepository) ListTimelineByAuthor(ctx context.Context, author string) ([]core.Timeline, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTimelineByAuthor", ctx, author)
	ret0, _ := ret[0].([]core.Timeline)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTimelineByAuthor indicates an expected call of ListTimelineByAuthor.
func (mr *MockRepositoryMockRecorder) ListTimelineByAuthor(ctx, author any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTimelineByAuthor", reflect.TypeOf((*MockRepository)(nil).ListTimelineByAuthor), ctx, author)
}

// ListTimelineByAuthorOwned mocks base method.
func (m *MockRepository) ListTimelineByAuthorOwned(ctx context.Context, author string) ([]core.Timeline, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTimelineByAuthorOwned", ctx, author)
	ret0, _ := ret[0].([]core.Timeline)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTimelineByAuthorOwned indicates an expected call of ListTimelineByAuthorOwned.
func (mr *MockRepositoryMockRecorder) ListTimelineByAuthorOwned(ctx, author any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTimelineByAuthorOwned", reflect.TypeOf((*MockRepository)(nil).ListTimelineByAuthorOwned), ctx, author)
}

// ListTimelineBySchema mocks base method.
func (m *MockRepository) ListTimelineBySchema(ctx context.Context, schema string) ([]core.Timeline, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTimelineBySchema", ctx, schema)
	ret0, _ := ret[0].([]core.Timeline)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTimelineBySchema indicates an expected call of ListTimelineBySchema.
func (mr *MockRepositoryMockRecorder) ListTimelineBySchema(ctx, schema any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTimelineBySchema", reflect.TypeOf((*MockRepository)(nil).ListTimelineBySchema), ctx, schema)
}

// ListTimelineSubscriptions mocks base method.
func (m *MockRepository) ListTimelineSubscriptions(ctx context.Context) (map[string]int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTimelineSubscriptions", ctx)
	ret0, _ := ret[0].(map[string]int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTimelineSubscriptions indicates an expected call of ListTimelineSubscriptions.
func (mr *MockRepositoryMockRecorder) ListTimelineSubscriptions(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTimelineSubscriptions", reflect.TypeOf((*MockRepository)(nil).ListTimelineSubscriptions), ctx)
}

// LoadChunkBodies mocks base method.
func (m *MockRepository) LoadChunkBodies(ctx context.Context, query map[string]string) (map[string]core.Chunk, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadChunkBodies", ctx, query)
	ret0, _ := ret[0].(map[string]core.Chunk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoadChunkBodies indicates an expected call of LoadChunkBodies.
func (mr *MockRepositoryMockRecorder) LoadChunkBodies(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadChunkBodies", reflect.TypeOf((*MockRepository)(nil).LoadChunkBodies), ctx, query)
}

// LookupChunkItrs mocks base method.
func (m *MockRepository) LookupChunkItrs(ctx context.Context, timelines []string, epoch string) (map[string]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LookupChunkItrs", ctx, timelines, epoch)
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LookupChunkItrs indicates an expected call of LookupChunkItrs.
func (mr *MockRepositoryMockRecorder) LookupChunkItrs(ctx, timelines, epoch any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LookupChunkItrs", reflect.TypeOf((*MockRepository)(nil).LookupChunkItrs), ctx, timelines, epoch)
}

// PublishEvent mocks base method.
func (m *MockRepository) PublishEvent(ctx context.Context, event core.Event) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishEvent", ctx, event)
	ret0, _ := ret[0].(error)
	return ret0
}

// PublishEvent indicates an expected call of PublishEvent.
func (mr *MockRepositoryMockRecorder) PublishEvent(ctx, event any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishEvent", reflect.TypeOf((*MockRepository)(nil).PublishEvent), ctx, event)
}

// Query mocks base method.
func (m *MockRepository) Query(ctx context.Context, timelineID, schema, owner, author string, until time.Time, limit int) ([]core.TimelineItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Query", ctx, timelineID, schema, owner, author, until, limit)
	ret0, _ := ret[0].([]core.TimelineItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Query indicates an expected call of Query.
func (mr *MockRepositoryMockRecorder) Query(ctx, timelineID, schema, owner, author, until, limit any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockRepository)(nil).Query), ctx, timelineID, schema, owner, author, until, limit)
}

// SetNormalizationCache mocks base method.
func (m *MockRepository) SetNormalizationCache(ctx context.Context, timelineID, value string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetNormalizationCache", ctx, timelineID, value)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetNormalizationCache indicates an expected call of SetNormalizationCache.
func (mr *MockRepositoryMockRecorder) SetNormalizationCache(ctx, timelineID, value any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNormalizationCache", reflect.TypeOf((*MockRepository)(nil).SetNormalizationCache), ctx, timelineID, value)
}

// Subscribe mocks base method.
func (m *MockRepository) Subscribe(ctx context.Context, channels []string, event chan<- core.Event) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe", ctx, channels, event)
	ret0, _ := ret[0].(error)
	return ret0
}

// Subscribe indicates an expected call of Subscribe.
func (mr *MockRepositoryMockRecorder) Subscribe(ctx, channels, event any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockRepository)(nil).Subscribe), ctx, channels, event)
}

// UpsertTimeline mocks base method.
func (m *MockRepository) UpsertTimeline(ctx context.Context, timeline core.Timeline) (core.Timeline, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpsertTimeline", ctx, timeline)
	ret0, _ := ret[0].(core.Timeline)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpsertTimeline indicates an expected call of UpsertTimeline.
func (mr *MockRepositoryMockRecorder) UpsertTimeline(ctx, timeline any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpsertTimeline", reflect.TypeOf((*MockRepository)(nil).UpsertTimeline), ctx, timeline)
}
