//go:generate go run go.uber.org/mock/mockgen -source=repository.go -destination=mock/repository.go
package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/totegamma/concurrent/client"
	"github.com/totegamma/concurrent/core"
)

// Repository is timeline repository interface
type Repository interface {
	GetTimeline(ctx context.Context, key string) (core.Timeline, error)
	GetTimelineFromRemote(ctx context.Context, host string, key string) (core.Timeline, error)
	UpsertTimeline(ctx context.Context, timeline core.Timeline) (core.Timeline, error)
	DeleteTimeline(ctx context.Context, key string) error

	GetItem(ctx context.Context, timelineID string, objectID string) (core.TimelineItem, error)
	CreateItem(ctx context.Context, item core.TimelineItem) (core.TimelineItem, error)
	DeleteItem(ctx context.Context, timelineID string, objectID string) error
	DeleteItemByResourceID(ctx context.Context, resourceID string) error

	ListTimelineBySchema(ctx context.Context, schema string) ([]core.Timeline, error)
	ListTimelineByAuthor(ctx context.Context, author string) ([]core.Timeline, error)
	ListTimelineByAuthorOwned(ctx context.Context, author string) ([]core.Timeline, error)

	GetRecentItems(ctx context.Context, timelineID string, until time.Time, limit int) ([]core.TimelineItem, error)
	GetImmediateItems(ctx context.Context, timelineID string, since time.Time, limit int) ([]core.TimelineItem, error)

	GetChunksFromCache(ctx context.Context, timelines []string, chunk string) (map[string]core.Chunk, error)
	GetChunksFromDB(ctx context.Context, timelines []string, chunk string) (map[string]core.Chunk, error)
	GetChunkIterators(ctx context.Context, timelines []string, chunk string) (map[string]string, error)
	PublishEvent(ctx context.Context, event core.Event) error

	ListTimelineSubscriptions(ctx context.Context) (map[string]int64, error)
	Count(ctx context.Context) (int64, error)

	Subscribe(ctx context.Context, channels []string, event chan<- core.Event) error

	SetNormalizationCache(ctx context.Context, timelineID string, value string) error
	GetNormalizationCache(ctx context.Context, timelineID string) (string, error)

	Query(ctx context.Context, timelineID, schema, owner, author string, until time.Time, limit int) ([]core.TimelineItem, error)

	LookupChunkItrs(ctx context.Context, timelines []string, epoch string) (map[string]string, error)
	LoadChunkBodies(ctx context.Context, query map[string]string) (map[string]core.Chunk, error)

	GetMetrics() map[string]int64
}

type repository struct {
	db     *gorm.DB
	rdb    *redis.Client
	mc     *memcache.Client
	client client.Client
	schema core.SchemaService
	config core.Config

	lookupChunkItrsCacheMisses int64
	lookupChunkItrsCacheHits   int64
	loadChunkBodiesCacheMisses int64
	loadChunkBodiesCacheHits   int64
}

// NewRepository creates a new timeline repository
func NewRepository(db *gorm.DB, rdb *redis.Client, mc *memcache.Client, client client.Client, schema core.SchemaService, config core.Config) Repository {

	var count int64
	err := db.Model(&core.Timeline{}).Count(&count).Error
	if err != nil {
		slog.Error(
			"failed to count timelines",
			slog.String("error", err.Error()),
		)
	}

	mc.Set(&memcache.Item{Key: "timeline_count", Value: []byte(strconv.FormatInt(count, 10))})

	return &repository{
		db,
		rdb,
		mc,
		client,
		schema,
		config,
		0, 0, 0, 0,
	}
}

func (r *repository) GetMetrics() map[string]int64 {
	return map[string]int64{
		"lookup_chunk_itr_cache_misses":  r.lookupChunkItrsCacheMisses,
		"lookup_chunk_itr_cache_hits":    r.lookupChunkItrsCacheHits,
		"load_chunk_bodies_cache_misses": r.loadChunkBodiesCacheMisses,
		"load_chunk_bodies_cache_hits":   r.loadChunkBodiesCacheHits,
	}
}

const (
	normaalizationCachePrefix = "tl:norm:"
	normaalizationCacheTTL    = 60 * 15 // 15 minutes

	tlItrCachePrefix  = "tl:itr:"
	tlBodyCachePrefix = "tl:body:"

	defaultChunkSize = 32
)

func (r *repository) LookupChunkItrs(ctx context.Context, normalized []string, epoch string) (map[string]string, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.LookupChunkItr")
	defer span.End()

	keys := make([]string, len(normalized))
	keytable := make(map[string]string)
	for i, timeline := range normalized {
		key := tlItrCachePrefix + timeline + ":" + epoch
		keys[i] = key
		keytable[key] = timeline
	}

	cache, err := r.mc.GetMulti(keys)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	var result = map[string]string{}
	var missed = []string{}
	for _, key := range keys {
		timeline := keytable[key]
		if cache[key] != nil {
			result[timeline] = string(cache[key].Value)
			r.lookupChunkItrsCacheHits++
		} else {
			missed = append(missed, timeline)
			r.lookupChunkItrsCacheMisses++
		}
	}

	var domainMap = make(map[string][]string)
	for _, timeline := range missed {
		split := strings.Split(timeline, "@")
		domain := split[len(split)-1]
		if len(split) >= 2 {
			if _, ok := domainMap[domain]; !ok {
				domainMap[domain] = make([]string, 0)
			}
			if domain == r.config.FQDN {
				domainMap[domain] = append(domainMap[domain], split[0])
			} else {
				domainMap[domain] = append(domainMap[domain], timeline)
			}
		}
	}

	for domain, timelines := range domainMap {
		if domain == r.config.FQDN {
			res, err := r.lookupLocalItrs(ctx, timelines, epoch)
			if err != nil {
				span.RecordError(err)
				continue
			}
			for k, v := range res {
				result[k] = v
			}
		} else {
			res, err := r.lookupRemoteItrs(ctx, domain, timelines, epoch)
			if err != nil {
				span.RecordError(err)
				continue
			}
			for k, v := range res {
				result[k] = v
			}
		}
	}

	return result, nil
}

func (r *repository) LoadChunkBodies(ctx context.Context, query map[string]string) (map[string]core.Chunk, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.LoadChunkBodies")
	defer span.End()

	keys := []string{}
	keytable := map[string]string{}
	for timeline, epoch := range query {
		key := tlBodyCachePrefix + timeline + ":" + epoch
		keys = append(keys, key)
		keytable[key] = timeline
	}

	cache, err := r.mc.GetMulti(keys)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	result := make(map[string]core.Chunk)
	var missed = map[string]string{}

	for _, key := range keys {
		timeline := keytable[key]
		if cache[key] != nil {
			var items []core.TimelineItem
			cacheStr := string(cache[key].Value)
			cacheStr = cacheStr[1:]
			cacheStr = "[" + cacheStr + "]"
			err = json.Unmarshal([]byte(cacheStr), &items)
			if err != nil {
				span.RecordError(err)
				continue
			}
			result[timeline] = core.Chunk{
				Key:   key,
				Epoch: query[timeline],
				Items: items,
			}
			r.loadChunkBodiesCacheHits++
		} else {
			missed[timeline] = query[timeline]
			r.loadChunkBodiesCacheMisses++
		}
	}

	var domainMap = make(map[string]map[string]string)
	for timeline, epoch := range missed {
		split := strings.Split(timeline, "@")
		domain := split[len(split)-1]
		if len(split) >= 2 {
			if _, ok := domainMap[domain]; !ok {
				domainMap[domain] = make(map[string]string)
			}
			domainMap[domain][timeline] = epoch
		}
	}

	for domain, q := range domainMap {
		if domain == r.config.FQDN {
			for timeline, epoch := range q {
				res, err := r.loadLocalBody(ctx, timeline, epoch)
				if err != nil {
					span.RecordError(err)
					continue
				}
				result[timeline] = res
			}
		} else {
			res, err := r.loadRemoteBodies(ctx, domain, q)
			if err != nil {
				span.RecordError(err)
				continue
			}
			for k, v := range res {
				result[k] = v
			}
		}
	}

	return result, nil
}

func (r *repository) lookupLocalItrs(ctx context.Context, timelines []string, epoch string) (map[string]string, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.LookupLocalItr")
	defer span.End()

	dbids := []string{}
	for _, timeline := range timelines {
		dbid := timeline
		if strings.Contains(dbid, "@") {
			split := strings.Split(timeline, "@")
			if len(split) > 1 && split[len(split)-1] != r.config.FQDN {
				span.RecordError(fmt.Errorf("invalid timeline id: %s", timeline))
				continue
			}
			dbid = split[0]
		}
		if len(dbid) == 27 {
			if dbid[0] != 't' {
				span.RecordError(fmt.Errorf("timeline typed-id must start with 't' %s", timeline))
				continue
			}
			dbid = dbid[1:]
		}
		if len(dbid) != 26 {
			span.RecordError(fmt.Errorf("timeline id must be 26 characters long %s", timeline))
			continue
		}
		dbids = append(dbids, dbid)
	}

	result := make(map[string]string)
	if len(dbids) > 0 {
		var res []struct {
			TimelineID string
			MaxCDate   time.Time
		}

		err := r.db.WithContext(ctx).
			Model(&core.TimelineItem{}).
			Select("timeline_id, max(c_date) as max_c_date").
			Where("timeline_id in (?) and c_date <= ?", dbids, core.Chunk2RecentTime(epoch)).
			Group("timeline_id").
			Scan(&res).Error
		if err != nil {
			fmt.Println("err", err)
			span.RecordError(err)
			return nil, err
		}

		for _, item := range res {
			id := "t" + item.TimelineID + "@" + r.config.FQDN
			key := tlItrCachePrefix + id + ":" + epoch
			value := core.Time2Chunk(item.MaxCDate)
			r.mc.Set(&memcache.Item{Key: key, Value: []byte(value)})
			result[id] = value
		}
	}

	return result, nil
}

func (r *repository) lookupRemoteItrs(ctx context.Context, domain string, timelines []string, epoch string) (map[string]string, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.LookupRemoteItr")
	defer span.End()

	result, err := r.client.GetChunkItrs(ctx, domain, timelines, epoch, nil)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	currentSubscriptions := r.GetCurrentSubs(ctx)

	for timeline, itr := range result {

		// 最新のチャンクに関しては、socketが張られてるキャッシュしか温められないのでそれだけ保持
		if epoch == core.Time2Chunk(time.Now()) && !slices.Contains(currentSubscriptions, timeline) {
			continue
		}

		key := tlItrCachePrefix + timeline + ":" + epoch
		r.mc.Set(&memcache.Item{Key: key, Value: []byte(itr)})
	}

	return result, nil
}

func (r *repository) loadLocalBody(ctx context.Context, timeline string, epoch string) (core.Chunk, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.LoadLocalBody")
	defer span.End()

	chunkDate := core.Chunk2RecentTime(epoch)
	prevChunkDate := core.Chunk2RecentTime(core.PrevChunk(epoch))

	timelineID := timeline
	if strings.Contains(timelineID, "@") {
		timelineID = strings.Split(timelineID, "@")[0]
	}
	if len(timelineID) == 27 {
		if timelineID[0] != 't' {
			return core.Chunk{}, fmt.Errorf("timeline typed-id must start with 't'")
		}
		timelineID = timelineID[1:]
	}

	var items []core.TimelineItem

	err := r.db.WithContext(ctx).
		Where("timeline_id = ? and c_date <= ?", timelineID, chunkDate).
		Order("c_date desc").
		Limit(defaultChunkSize).
		Find(&items).Error

	// 得られた中で最も古いアイテムがチャンクをまたいでない場合、取得漏れがある可能性がある
	// 代わりに、チャンク内のレンジの全てのアイテムを取得する
	if items[len(items)-1].CDate.After(prevChunkDate) {
		err = r.db.WithContext(ctx).
			Where("timeline_id = ? and ? < c_date and c_date <= ?", timelineID, prevChunkDate, chunkDate).
			Order("c_date desc").
			Find(&items).Error
	}

	if err != nil {
		span.RecordError(err)
		return core.Chunk{}, err
	}

	// append domain to timelineID
	for i, item := range items {
		items[i].TimelineID = item.TimelineID + "@" + r.config.FQDN
	}

	b, err := json.Marshal(items)
	if err != nil {
		span.RecordError(err)
		return core.Chunk{}, err
	}
	key := tlBodyCachePrefix + timeline + ":" + epoch
	cacheStr := "," + string(b[1:len(b)-1])
	err = r.mc.Set(&memcache.Item{Key: key, Value: []byte(cacheStr)})
	if err != nil {
		span.RecordError(err)
		return core.Chunk{}, err
	}

	return core.Chunk{
		Key:   key,
		Epoch: epoch,
		Items: items,
	}, nil

}

func (r *repository) loadRemoteBodies(ctx context.Context, remote string, query map[string]string) (map[string]core.Chunk, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.LoadRemoteBody")
	defer span.End()

	result, err := r.client.GetChunkBodies(ctx, remote, query, nil)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	currentSubscriptions := r.GetCurrentSubs(ctx)
	for timeline, chunk := range result {

		// 最新のチャンクに関しては、socketが張られてるキャッシュしか温められないのでそれだけ保持
		if chunk.Epoch == core.Time2Chunk(time.Now()) && !slices.Contains(currentSubscriptions, timeline) {
			continue
		}

		key := tlBodyCachePrefix + timeline + ":" + chunk.Epoch
		b, err := json.Marshal(chunk.Items)
		if err != nil {
			span.RecordError(err)
			continue
		}
		cacheStr := "," + string(b[1:len(b)-1])
		err = r.mc.Set(&memcache.Item{Key: key, Value: []byte(cacheStr)})
		if err != nil {
			span.RecordError(err)
			continue
		}
	}

	return result, nil
}

func (r *repository) SetNormalizationCache(ctx context.Context, timelineID string, value string) error {
	return r.mc.Set(&memcache.Item{Key: normaalizationCachePrefix + timelineID, Value: []byte(value), Expiration: normaalizationCacheTTL})
}

func (r *repository) GetNormalizationCache(ctx context.Context, timelineID string) (string, error) {
	item, err := r.mc.Get(normaalizationCachePrefix + timelineID)
	if err != nil {
		return "", err
	}
	return string(item.Value), nil
}

func (r *repository) normalizeLocalDBID(id string) (string, error) {

	normalized := id

	split := strings.Split(normalized, "@")
	if len(split) == 2 {
		normalized = split[0]

		if split[1] != r.config.FQDN {
			return "", fmt.Errorf("invalid timeline id: %s", id)
		}
	}

	if len(normalized) == 27 {
		if normalized[0] != 't' {
			return "", fmt.Errorf("timeline id must start with 't'")
		}
		normalized = normalized[1:]
	}

	if len(normalized) != 26 {
		return "", fmt.Errorf("timeline id must be 26 characters long")
	}

	return normalized, nil
}

func (r *repository) preprocess(ctx context.Context, timeline *core.Timeline) error {

	var err error
	timeline.ID, err = r.normalizeLocalDBID(timeline.ID)
	if err != nil {
		return err
	}

	if timeline.SchemaID == 0 {
		schemaID, err := r.schema.UrlToID(ctx, timeline.Schema)
		if err != nil {
			return err
		}
		timeline.SchemaID = schemaID
	}

	if timeline.PolicyID == 0 && timeline.Policy != "" {
		policyID, err := r.schema.UrlToID(ctx, timeline.Policy)
		if err != nil {
			return err
		}
		timeline.PolicyID = policyID
	}

	return nil
}

func (r *repository) postprocess(ctx context.Context, timeline *core.Timeline) error {

	if len(timeline.ID) == 26 {
		timeline.ID = "t" + timeline.ID
	}

	if timeline.SchemaID != 0 && timeline.Schema == "" {
		schemaUrl, err := r.schema.IDToUrl(ctx, timeline.SchemaID)
		if err != nil {
			return err
		}
		timeline.Schema = schemaUrl
	}

	if timeline.PolicyID != 0 && timeline.Policy == "" {
		policyUrl, err := r.schema.IDToUrl(ctx, timeline.PolicyID)
		if err != nil {
			return err
		}
		timeline.Policy = policyUrl
	}

	return nil
}

// Total returns the total number of messages
func (r *repository) Count(ctx context.Context) (int64, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.Count")
	defer span.End()

	item, err := r.mc.Get("timeline_count")
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	count, err := strconv.ParseInt(string(item.Value), 10, 64)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}
	return count, nil
}

func (r *repository) PublishEvent(ctx context.Context, event core.Event) error {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.PublishEvent")
	defer span.End()

	jsonstr, _ := json.Marshal(event)

	err := r.rdb.Publish(context.Background(), event.Timeline, jsonstr).Err()
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(
			ctx, "fail to publish message to Redis",
			slog.String("error", err.Error()),
			slog.String("module", "timeline"),
		)
	}

	return nil
}

// GetTimelineFromRemote gets a timeline from remote
func (r *repository) GetTimelineFromRemote(ctx context.Context, host string, key string) (core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.GetTimelineFromRemote")
	defer span.End()

	// check cache
	cacheKey := "timeline:" + key + "@" + host
	item, err := r.mc.Get(cacheKey)
	if err == nil {
		var timeline core.Timeline
		err = json.Unmarshal(item.Value, &timeline)
		if err == nil {
			return timeline, nil
		}
		span.RecordError(err)
	}

	timeline, err := r.client.GetTimeline(ctx, host, key, nil)
	if err != nil {
		span.RecordError(err)
		return core.Timeline{}, err
	}

	// save to cache
	body, err := json.Marshal(timeline)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(
			ctx, "fail to marshal timeline",
			slog.String("error", err.Error()),
			slog.String("module", "timeline"),
		)
		return core.Timeline{}, err
	}

	err = r.mc.Set(&memcache.Item{Key: cacheKey, Value: body, Expiration: 300}) // 5 minutes
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(
			ctx, "fail to save cache",
			slog.String("error", err.Error()),
			slog.String("module", "timeline"),
		)
	}

	return timeline, nil
}

// GetChunksFromCache gets chunks from cache
func (r *repository) GetChunksFromCache(ctx context.Context, timelines []string, chunk string) (map[string]core.Chunk, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.GetChunksFromCache")
	defer span.End()

	targetKeyMap, err := r.GetChunkIterators(ctx, timelines, chunk)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	targetKeys := make([]string, 0)
	for _, targetKey := range targetKeyMap {
		targetKeys = append(targetKeys, targetKey)
	}

	if len(targetKeys) == 0 {
		return map[string]core.Chunk{}, nil
	}

	caches, err := r.mc.GetMulti(targetKeys)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	result := make(map[string]core.Chunk)
	for _, timeline := range timelines {
		targetKey := targetKeyMap[timeline]
		cache, ok := caches[targetKey]
		if !ok || len(cache.Value) == 0 {
			continue
		}

		var items []core.TimelineItem
		cacheStr := string(cache.Value)
		cacheStr = cacheStr[1:]
		cacheStr = "[" + cacheStr + "]"
		err = json.Unmarshal([]byte(cacheStr), &items)
		if err != nil {
			span.RecordError(err)
			continue
		}
		result[timeline] = core.Chunk{
			Key:   targetKey,
			Items: items,
		}
	}

	return result, nil
}

// GetChunksFromDB gets chunks from db and cache them
func (r *repository) GetChunksFromDB(ctx context.Context, timelines []string, chunk string) (map[string]core.Chunk, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.GetChunksFromDB")
	defer span.End()

	targetKeyMap, err := r.GetChunkIterators(ctx, timelines, chunk)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	targetKeys := make([]string, 0)
	for _, targetKey := range targetKeyMap {
		targetKeys = append(targetKeys, targetKey)
	}

	result := make(map[string]core.Chunk)
	for _, timeline := range timelines {
		targetKey := targetKeyMap[timeline]
		var items []core.TimelineItem
		chunkDate := core.Chunk2RecentTime(chunk)

		timelineID := timeline
		if strings.Contains(timelineID, "@") {
			timelineID = strings.Split(timelineID, "@")[0]
		}
		if len(timelineID) == 27 {
			if timelineID[0] != 't' {
				return nil, fmt.Errorf("timeline typed-id must start with 't'")
			}
			timelineID = timelineID[1:]
		}

		err = r.db.WithContext(ctx).Where("timeline_id = ? and c_date <= ?", timelineID, chunkDate).Order("c_date desc").Limit(100).Find(&items).Error
		if err != nil {
			span.RecordError(err)
			continue
		}

		// append domain to timelineID
		for i, item := range items {
			items[i].TimelineID = item.TimelineID + "@" + r.config.FQDN
		}

		result[timeline] = core.Chunk{
			Key:   targetKey,
			Items: items,
		}

		b, err := json.Marshal(items)
		if err != nil {
			span.RecordError(err)
			continue
		}
		cacheStr := "," + string(b[1:len(b)-1])
		err = r.mc.Set(&memcache.Item{Key: targetKey, Value: []byte(cacheStr)})
		if err != nil {
			span.RecordError(err)
			continue
		}
	}

	return result, nil
}

// GetChunkIterators returns a list of iterated chunk keys
func (r *repository) GetChunkIterators(ctx context.Context, timelines []string, chunk string) (map[string]string, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.GetChunkIterators")
	defer span.End()

	keys := make([]string, len(timelines))
	for i, timeline := range timelines {
		keys[i] = "timeline:itr:all:" + timeline + ":" + chunk
	}

	cache, err := r.mc.GetMulti(keys)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	result := make(map[string]string)
	for i, timeline := range timelines {
		if cache[keys[i]] != nil { // hit
			result[timeline] = string(cache[keys[i]].Value)
		} else { // miss
			var item core.TimelineItem
			chunkTime := core.Chunk2RecentTime(chunk)
			dbid := timeline
			if strings.Contains(dbid, "@") {
				dbid = strings.Split(timeline, "@")[0]
			}
			if len(dbid) == 27 {
				if dbid[0] != 't' {
					return nil, fmt.Errorf("timeline typed-id must start with 't'")
				}
				dbid = dbid[1:]
			}
			err := r.db.WithContext(ctx).Where("timeline_id = ? and c_date <= ?", dbid, chunkTime).Order("c_date desc").First(&item).Error
			if err != nil {
				continue
			}
			key := "timeline:body:all:" + timeline + ":" + core.Time2Chunk(item.CDate)
			r.mc.Set(&memcache.Item{Key: keys[i], Value: []byte(key)})
			result[timeline] = key
		}
	}

	return result, nil
}

// GetItem returns a timeline item by TimelineID and ObjectID
func (r *repository) GetItem(ctx context.Context, timelineID string, objectID string) (core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.GetItem")
	defer span.End()

	var item core.TimelineItem
	err := r.db.WithContext(ctx).First(&item, "timeline_id = ? and resource_id = ?", timelineID, objectID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return core.TimelineItem{}, core.NewErrorNotFound()
		}
		span.RecordError(err)
		return core.TimelineItem{}, err
	}

	return item, nil
}

// CreateItem creates a new timeline item
func (r *repository) CreateItem(ctx context.Context, item core.TimelineItem) (core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.CreateItem")
	defer span.End()

	if len(item.TimelineID) == 27 {
		if item.TimelineID[0] != 't' {
			return core.TimelineItem{}, fmt.Errorf("timeline typed-id must start with 't'")
		}
		item.TimelineID = item.TimelineID[1:]
	}

	schemaID, err := r.schema.UrlToID(ctx, item.Schema)
	if err != nil {
		return core.TimelineItem{}, err
	}
	item.SchemaID = schemaID

	err = r.db.WithContext(ctx).Create(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return core.TimelineItem{}, core.NewErrorAlreadyExists()
		}
		span.RecordError(err)
		return item, err
	}

	timelineID := "t" + item.TimelineID + "@" + r.config.FQDN

	json, err := json.Marshal(item)
	if err != nil {
		span.RecordError(err)
		return item, err
	}

	val := "," + string(json)
	itemChunk := core.Time2Chunk(item.CDate)
	itrKey := tlItrCachePrefix + timelineID + ":" + itemChunk
	cacheKey := tlBodyCachePrefix + timelineID + ":" + itemChunk

	// もし今からPrependするbodyブロックにイテレーターが向いてない場合は向きを変えておく必要がある
	// これが発生するのは、タイムラインが久々に更新されたときで、最近のイテレーターが古いbodyブロックを向いている状態になっている
	// そのため、イテレーターを更新しないと、古いbodyブロック(更新されない)を見続けてしまう為、新しく書き込んだデータが読み込まれない。
	// Note:
	// この処理は今から挿入するアイテムが最新のチャンクであることが前提になっている。
	// 古いデータを挿入する場合は、書き込みを行ったチャンクから最新のチャンクまでのイテレーターを更新する必要があるかも。
	// 範囲でforを回して、キャッシュをdeleteする処理を追加する必要があるだろう...
	r.mc.Replace(&memcache.Item{Key: itrKey, Value: []byte(itemChunk)})
	r.mc.Prepend(&memcache.Item{Key: cacheKey, Value: []byte(val)})

	item.TimelineID = "t" + item.TimelineID

	return item, nil
}

// DeleteItem deletes a timeline item
func (r *repository) DeleteItem(ctx context.Context, timelineID string, objectID string) error {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.DeleteItem")
	defer span.End()

	return r.db.WithContext(ctx).Delete(&core.TimelineItem{}, "timeline_id = ? and resource_id = ?", timelineID, objectID).Error
}

func (r *repository) DeleteItemByResourceID(ctx context.Context, resourceID string) error {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.DeleteItemByResourceID")
	defer span.End()

	return r.db.WithContext(ctx).Delete(&core.TimelineItem{}, "resource_id = ?", resourceID).Error
}

// GetTimelineRecent returns a list of timeline items by TimelineID and time range
func (r *repository) GetRecentItems(ctx context.Context, timelineID string, until time.Time, limit int) ([]core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.GetRecentItems")
	defer span.End()

	var items []core.TimelineItem
	err := r.db.WithContext(ctx).Where("timeline_id = ? and c_date < ?", timelineID, until).Order("c_date desc").Limit(limit).Find(&items).Error
	return items, err
}

// GetTimelineImmediate returns a list of timeline items by TimelineID and time range
func (r *repository) GetImmediateItems(ctx context.Context, timelineID string, since time.Time, limit int) ([]core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.GetImmediateItems")
	defer span.End()

	var items []core.TimelineItem
	err := r.db.WithContext(ctx).Where("timeline_id = ? and c_date > ?", timelineID, since).Order("c_date asec").Limit(limit).Find(&items).Error
	return items, err
}

// GetTimeline returns a timeline by ID
func (r *repository) GetTimeline(ctx context.Context, id string) (core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.GetTimeline")
	defer span.End()

	id, err := r.normalizeLocalDBID(id)
	if err != nil {
		return core.Timeline{}, err
	}

	var timeline core.Timeline
	err = r.db.WithContext(ctx).First(&timeline, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return core.Timeline{}, core.NewErrorNotFound()
		}
		span.RecordError(err)
		return core.Timeline{}, err
	}

	err = r.postprocess(ctx, &timeline)
	if err != nil {
		return core.Timeline{}, err
	}

	return timeline, err
}

// Create updates a timeline
func (r *repository) UpsertTimeline(ctx context.Context, timeline core.Timeline) (core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.UpsertTimeline")
	defer span.End()

	err := r.preprocess(ctx, &timeline)
	if err != nil {
		return core.Timeline{}, err
	}

	err = r.db.WithContext(ctx).Save(&timeline).Error
	if err != nil {
		return core.Timeline{}, err
	}

	err = r.postprocess(ctx, &timeline)
	if err != nil {
		return core.Timeline{}, err
	}

	r.mc.Increment("timeline_count", 1)

	return timeline, err
}

// GetListBySchema returns list of schemas by schema
func (r *repository) ListTimelineBySchema(ctx context.Context, schema string) ([]core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.ListTimelineBySchema")
	defer span.End()

	id, err := r.schema.UrlToID(ctx, schema)
	if err != nil {
		return []core.Timeline{}, err
	}

	var timelines []core.Timeline
	err = r.db.WithContext(ctx).Where("schema_id = ? and indexable = true", id).Find(&timelines).Error

	for i := range timelines {
		err := r.postprocess(ctx, &timelines[i])
		if err != nil {
			return []core.Timeline{}, err
		}
	}

	return timelines, err
}

// GetListByAuthor returns list of schemas by owner
func (r *repository) ListTimelineByAuthor(ctx context.Context, author string) ([]core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.ListTimelineByAuthor")
	defer span.End()

	var timelines []core.Timeline
	err := r.db.WithContext(ctx).Where("Author = ?", author).Find(&timelines).Error

	for i := range timelines {
		err := r.postprocess(ctx, &timelines[i])
		if err != nil {
			return []core.Timeline{}, err
		}
	}

	return timelines, err
}

func (r *repository) ListTimelineByAuthorOwned(ctx context.Context, author string) ([]core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.ListTimelineByAuthorOwned")
	defer span.End()

	var timelines []core.Timeline
	err := r.db.WithContext(ctx).Where("Author = ? and domain_owned = false", author).Find(&timelines).Error

	for i := range timelines {
		err := r.postprocess(ctx, &timelines[i])
		if err != nil {
			return []core.Timeline{}, err
		}
	}

	return timelines, err
}

// Delete deletes a timeline
func (r *repository) DeleteTimeline(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.DeleteTimeline")
	defer span.End()

	id, err := r.normalizeLocalDBID(id)
	if err != nil {
		return err
	}

	// delete items
	err = r.db.WithContext(ctx).Delete(&core.TimelineItem{}, "timeline_id = ?", id).Error
	if err != nil {
		return err
	}

	err = r.db.WithContext(ctx).Delete(&core.Timeline{}, "id = ?", id).Error
	if err != nil {
		return err
	}

	r.mc.Decrement("timeline_count", 1)

	return nil
}

// List Timeline Subscriptions
func (r *repository) ListTimelineSubscriptions(ctx context.Context) (map[string]int64, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.ListTimelineSubscriptions")
	defer span.End()

	query_l := r.rdb.PubSubChannels(ctx, "*")
	timelines, err := query_l.Result()
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	query_n := r.rdb.PubSubNumSub(ctx, timelines...)
	result, err := query_n.Result()
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return result, nil
}

func (r *repository) Subscribe(ctx context.Context, channels []string, event chan<- core.Event) error {

	if len(channels) == 0 {
		return nil
	}

	pubsub := r.rdb.Subscribe(ctx, channels...)
	defer pubsub.Close()

	chanstr := strings.Join(channels, ",")
	err := r.rdb.Publish(context.Background(), "concrnt:subscription:updated", chanstr).Err()
	if err != nil {
		slog.ErrorContext(
			ctx, "fail to publish message to Redis",
			slog.String("error", err.Error()),
			slog.String("module", "timeline"),
		)
	}

	psch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-psch:
			var item core.Event
			err := json.Unmarshal([]byte(msg.Payload), &item)
			if err != nil {
				slog.Error(
					"failed to unmarshal message",
					slog.String("error", err.Error()),
				)
				continue
			}
			event <- item
		}
	}
}

func (r *repository) GetCurrentSubs(ctx context.Context) []string {

	query := r.rdb.PubSubChannels(ctx, "*")
	channels := query.Val()

	uniqueChannelsMap := make(map[string]bool)
	for _, channel := range channels {
		uniqueChannelsMap[channel] = true
	}

	uniqueChannels := make([]string, 0)
	for channel := range uniqueChannelsMap {
		split := strings.Split(channel, "@")
		if len(split) != 2 {
			continue
		}
		uniqueChannels = append(uniqueChannels, channel)
	}

	return uniqueChannels
}

func (r *repository) Query(ctx context.Context, timelineID, schema, owner, author string, until time.Time, limit int) ([]core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Repository.Query")
	defer span.End()

	query := r.db.WithContext(ctx).Model(&core.TimelineItem{})

	if timelineID != "" {
		if len(timelineID) == 27 {
			if timelineID[0] != 't' {
				return nil, fmt.Errorf("timeline typed-id must start with 't'")
			}
			timelineID = timelineID[1:]
		}

		query = query.Where("timeline_id = ?", timelineID)
	}

	if schema != "" {
		schemaID, err := r.schema.UrlToID(ctx, schema)
		if err != nil {
			return nil, err
		}
		query = query.Where("schema_id = ?", schemaID)
	}

	if owner != "" {
		query = query.Where("owner = ?", owner)
	}

	if author != "" {
		query = query.Where("author = ?", author)
	}

	var items []core.TimelineItem
	err := query.Where("c_date < ?", until).Order("c_date desc").Limit(limit).Find(&items).Error
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return items, nil
}
