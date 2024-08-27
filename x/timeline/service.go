package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/totegamma/concurrent/cdid"
	"github.com/totegamma/concurrent/core"
)

type service struct {
	repository   Repository
	entity       core.EntityService
	domain       core.DomainService
	semanticid   core.SemanticIDService
	subscription core.SubscriptionService
	policy       core.PolicyService
	config       core.Config

	socketCounter int64
}

// NewService creates a new service
func NewService(
	repository Repository,
	entity core.EntityService,
	domain core.DomainService,
	semanticid core.SemanticIDService,
	subscription core.SubscriptionService,
	policy core.PolicyService,
	config core.Config,
) core.TimelineService {
	return &service{
		repository,
		entity,
		domain,
		semanticid,
		subscription,
		policy,
		config,
		0,
	}
}

// Count returns the count number of messages
func (s *service) Count(ctx context.Context) (int64, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.Count")
	defer span.End()

	return s.repository.Count(ctx)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *service) CurrentRealtimeConnectionCount() int64 {
	return atomic.LoadInt64(&s.socketCounter)
}

func (s *service) GetChunksFromRemote(ctx context.Context, host string, timelines []string, pivot time.Time) (map[string]core.Chunk, error) {
	return s.repository.GetChunksFromRemote(ctx, host, timelines, pivot)
}

// NormalizeTimelineID normalizes timelineID
// t+<hash> -> t+<hash>@<localdomain>
// t+<hash>@<anydomain> -> t+<hash>@<anydomain>
// t+<hash>@<anyuser> -> t+<hash>@<anydomain>
// <semanticID>@<localuser> -> t+<hash>@<localdomain>
// <semanticID>@<remoteuser> -> <semanticID>@<userID>@<domainname>
// <semanticID>@<userID>@<localdomain> -> t+<hash>@<localdomain>
// <semanticID>@<userID>@<remotedomain> -> <semanticID>@<userID>@<remotedomain>
func (s *service) NormalizeTimelineID(ctx context.Context, timeline string) (string, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.NormalizeTimelineID")
	defer span.End()

	// CheckCache
	cached, err := s.repository.GetNormalizationCache(ctx, timeline)
	if err == nil {
		return cached, nil
	}

	var normalized string

	split := strings.Split(timeline, "@")

	if len(split) == 1 {
		return timeline + "@" + s.config.FQDN, nil
	}

	id := split[0]
	domain := split[len(split)-1]

	var userid string
	if len(split) == 3 {
		userid = split[1]
	}

	if core.IsCCID(domain) {
		userid = domain
		entity, err := s.entity.Get(ctx, domain)
		if err != nil {
			span.SetAttributes(attribute.String("timeline", timeline))
			span.RecordError(err)
			return "", err
		}
		domain = entity.Domain
	}

	if domain == s.config.FQDN {
		if cdid.IsSeemsCDID(id, 't') {
			normalized = id + "@" + domain
		} else {
			target, err := s.semanticid.Lookup(ctx, id, userid)
			if err != nil {
				span.SetAttributes(attribute.String("timeline", timeline))
				err = errors.Wrap(err, "failed to lookup semanticID")
				span.RecordError(err)
				return "", err
			}
			normalized = target + "@" + domain
		}
	} else {
		if cdid.IsSeemsCDID(id, 't') {
			normalized = id + "@" + domain
		} else {
			normalized = id + "@" + userid + "@" + domain
		}
	}

	err = s.repository.SetNormalizationCache(ctx, timeline, normalized)
	if err != nil {
		span.RecordError(err)
	}

	return normalized, nil
}

// GetChunks returns chunks by timelineID and time
func (s *service) GetChunks(ctx context.Context, timelines []string, until time.Time) (map[string]core.Chunk, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.GetChunks")
	defer span.End()

	var normalized = make([]string, len(timelines))
	var normalizeMap = make(map[string]string)

	// normalize timelineID and validate
	for i, timeline := range timelines {
		n, err := s.NormalizeTimelineID(ctx, timeline)
		if err != nil {
			continue
		}
		normalized[i] = n
		normalizeMap[n] = timeline
	}

	// first, try to get from cache
	untilChunk := core.Time2Chunk(until)
	items, err := s.repository.GetChunksFromCache(ctx, normalized, untilChunk)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get chunks from cache", slog.String("error", err.Error()), slog.String("module", "timeline"))
		span.RecordError(err)
		return nil, err
	}

	// if not found in cache, get from db
	missingTimelines := make([]string, 0)
	for _, timeline := range normalized {
		if _, ok := items[timeline]; !ok {
			missingTimelines = append(missingTimelines, timeline)
		}
	}

	if len(missingTimelines) > 0 {
		// get from db
		dbItems, err := s.repository.GetChunksFromDB(ctx, missingTimelines, untilChunk)
		if err != nil {
			slog.ErrorContext(ctx, "failed to get chunks from db", slog.String("error", err.Error()), slog.String("module", "timeline"))
			span.RecordError(err)
			return nil, err
		}
		// merge
		for k, v := range dbItems {
			items[k] = v
		}
	}

	// recover original timelineID
	recovered := make(map[string]core.Chunk)
	for k, v := range items {
		recovered[normalizeMap[k]] = v
	}

	return recovered, nil
}

func (s *service) GetRecentItemsFromSubscription(ctx context.Context, subscription string, until time.Time, limit int) ([]core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.GetRecentItemsFromSubscription")
	defer span.End()

	sub, err := s.subscription.GetSubscription(ctx, subscription)
	if err != nil {
		return nil, err
	}

	timelines := make([]string, 0)
	for _, t := range sub.Items {
		timelines = append(timelines, t.ID)
	}

	return s.GetRecentItems(ctx, timelines, until, limit)
}

// GetRecentItems returns recent message from timelines
func (s *service) GetRecentItems(ctx context.Context, timelines []string, until time.Time, limit int) ([]core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.GetRecentItems")
	defer span.End()

	var normalized = []string{}
	var domainMap = make(map[string][]string)

	// normalize timelineID and validate
	for _, timeline := range timelines {
		n, err := s.NormalizeTimelineID(ctx, timeline)
		if err != nil {
			continue
		}

		split := strings.Split(n, "@")
		domain := split[len(split)-1]
		if len(split) >= 2 {
			if _, ok := domainMap[domain]; !ok {
				domainMap[domain] = make([]string, 0)
			}
			domainMap[domain] = append(domainMap[domain], n)
		}

		normalized = append(normalized, n)
	}

	// first, try to get from cache regardless of local or remote
	untilChunk := core.Time2Chunk(until)
	items, err := s.repository.GetChunksFromCache(ctx, normalized, untilChunk)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get chunks from cache", slog.String("error", err.Error()), slog.String("module", "timeline"))
		span.RecordError(err)
		return nil, err
	}

	for host, timelines := range domainMap {
		if host == s.config.FQDN {
			chunks, err := s.repository.GetChunksFromDB(ctx, timelines, untilChunk)
			if err != nil {
				slog.ErrorContext(ctx, "failed to get chunks from db", slog.String("error", err.Error()), slog.String("module", "timeline"))
				span.RecordError(err)
				return nil, err
			}
			for timeline, chunk := range chunks {
				items[timeline] = chunk
			}
		} else {
			chunks, err := s.repository.GetChunksFromRemote(ctx, host, timelines, until)
			if err != nil {
				slog.ErrorContext(ctx, "failed to get chunks from remote", slog.String("error", err.Error()), slog.String("module", "timeline"))
				span.RecordError(err)
				continue
			}
			for timeline, chunk := range chunks {
				items[timeline] = chunk
			}
		}
	}

	// summary messages and remove earlier than until
	var messages []core.TimelineItem
	for _, item := range items {
		for _, timelineItem := range item.Items {
			if timelineItem.CDate.After(until) {
				continue
			}
			messages = append(messages, timelineItem)
		}
	}

	var uniq []core.TimelineItem
	m := make(map[string]bool)
	for _, elem := range messages {
		if !m[elem.ResourceID] {
			m[elem.ResourceID] = true
			uniq = append(uniq, elem)
		}
	}

	sort.Slice(uniq, func(l, r int) bool {
		return uniq[l].CDate.After(uniq[r].CDate)
	})

	chopped := uniq[:min(len(uniq), limit)]

	return chopped, nil
}

func (s *service) GetImmediateItemsFromSubscription(ctx context.Context, subscription string, since time.Time, limit int) ([]core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.GetImmediateItemsFromSubscription")
	defer span.End()

	sub, err := s.subscription.GetSubscription(ctx, subscription)
	if err != nil {
		return nil, err
	}

	timelines := make([]string, 0)
	for _, t := range sub.Items {
		timelines = append(timelines, t.ID)
	}

	return s.GetImmediateItems(ctx, timelines, since, limit)
}

// GetImmediateItems returns immediate message from timelines
func (s *service) GetImmediateItems(ctx context.Context, timelines []string, since time.Time, limit int) ([]core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.GetImmediateItems")
	defer span.End()

	return nil, fmt.Errorf("not implemented")
}

// Post posts events to the local timeline.
func (s *service) PostItem(ctx context.Context, timeline string, item core.TimelineItem, document, signature string) (core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.PostItem")
	defer span.End()

	span.SetAttributes(attribute.String("timeline", timeline))

	query := strings.Split(timeline, "@")
	if len(query) != 2 {
		return core.TimelineItem{}, fmt.Errorf("Invalid format: %v", timeline)
	}

	timelineID, timelineHost := query[0], query[1]

	if core.IsCCID(timelineHost) {
		requester, err := s.entity.Get(ctx, timelineHost)
		if err != nil {
			span.RecordError(err)
			return core.TimelineItem{}, err
		}
		timelineHost = requester.Domain
	}

	if !cdid.IsSeemsCDID(timelineID, 't') && timelineHost == s.config.FQDN && core.IsCCID(query[1]) {
		target, err := s.semanticid.Lookup(ctx, timelineID, query[1])
		if err != nil {
			span.RecordError(err)
			return core.TimelineItem{}, err
		}
		timelineID = target
	}

	item.TimelineID = timelineID

	author := item.Owner
	if item.Author != nil {
		author = *item.Author
	}

	if timelineHost != s.config.FQDN {
		span.RecordError(fmt.Errorf("Remote timeline is not supported"))
		return core.TimelineItem{}, fmt.Errorf("Program error: remote timeline is not supported")
	}

	// check if the user has write access to the timeline

	tl, err := s.GetTimeline(ctx, timeline)
	if err != nil {
		return core.TimelineItem{}, err
	}

	requesterEntity, err := s.entity.Get(ctx, author)
	if err != nil {
		span.RecordError(err)
	}

	var params map[string]any = make(map[string]any)
	if tl.PolicyParams != nil {
		json.Unmarshal([]byte(*tl.PolicyParams), &params)
	}

	result, err := s.policy.TestWithPolicyURL(
		ctx,
		tl.Policy,
		core.RequestContext{
			Self:      tl,
			Requester: requesterEntity,
			Params:    params,
		},
		"timeline.distribute",
	)
	if err != nil {
		span.RecordError(err)
	}

	writable := s.policy.Summerize([]core.PolicyEvalResult{result}, "timeline.distribute")

	if !writable {
		span.RecordError(fmt.Errorf("You don't have timeline.distribute access to %v", timelineID))
		span.SetAttributes(attribute.Int("result", int(result)))
		slog.InfoContext(
			ctx, "failed to post to timeline",
			slog.String("type", "audit"),
			slog.String("principal", author),
			slog.String("timeline", timelineID),
			slog.String("module", "timeline"),
		)
		return core.TimelineItem{}, fmt.Errorf("You don't have write access to %v", timelineID)
	}

	slog.DebugContext(
		ctx, fmt.Sprintf("post to local timeline: %v to %v", item.ResourceID, timelineID),
		slog.String("module", "timeline"),
	)

	// add to timeline
	created, err := s.repository.CreateItem(ctx, item)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create item", slog.String("error", err.Error()), slog.String("module", "timeline"))
		span.RecordError(err)
		return core.TimelineItem{}, err
	}

	return created, nil
}

func (s *service) RemoveItemsByResourceID(ctx context.Context, resourceID string) error {
	ctx, span := tracer.Start(ctx, "Timeline.Service.RemoveItemByResourceID")
	defer span.End()

	err := s.repository.DeleteItemByResourceID(ctx, resourceID)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (s *service) PublishEvent(ctx context.Context, event core.Event) error {
	ctx, span := tracer.Start(ctx, "Timeline.Service.PublishEvent")
	defer span.End()

	normalized, err := s.NormalizeTimelineID(ctx, event.Timeline)
	if err == nil {
		event.Timeline = normalized
	}

	return s.repository.PublishEvent(ctx, event)
}

func (s *service) Event(ctx context.Context, mode core.CommitMode, document, signature string) (core.Event, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.Event")
	defer span.End()

	var doc core.EventDocument
	err := json.Unmarshal([]byte(document), &doc)
	if err != nil {
		span.RecordError(err)
		return core.Event{}, err
	}

	event := core.Event{
		Timeline:  doc.Timeline,
		Item:      doc.Item,
		Document:  doc.Document,
		Signature: doc.Signature,
		Resource:  doc.Resource,
	}

	return event, s.repository.PublishEvent(ctx, event)
}

// Create updates timeline information
func (s *service) UpsertTimeline(ctx context.Context, mode core.CommitMode, document, signature string) (core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.UpsertTimline")
	defer span.End()

	var doc core.TimelineDocument[any]
	err := json.Unmarshal([]byte(document), &doc)
	if err != nil {
		return core.Timeline{}, err
	}

	// return existing timeline if semanticID exists
	if doc.SemanticID != "" {
		existingID, err := s.semanticid.Lookup(ctx, doc.SemanticID, doc.Signer)
		if err == nil { // なければなにもしない
			_, err := s.repository.GetTimeline(ctx, existingID) // 実在性チェック
			if err != nil {                                     // 実在しなければ掃除しておく
				s.semanticid.Delete(ctx, doc.SemanticID, doc.Signer)
			} else {
				if doc.ID == "" { // あるかつIDがない場合はセット
					doc.ID = existingID
				} else {
					if doc.ID != existingID { // あるかつIDが違う場合はエラー
						return core.Timeline{}, fmt.Errorf("SemanticID Mismatch: %s != %s", doc.ID, existingID)
					}
				}
			}
		}
	}

	signer, err := s.entity.Get(ctx, doc.Signer)
	if err != nil {
		span.RecordError(err)
		return core.Timeline{}, err
	}

	if doc.ID == "" { // Create
		hash := core.GetHash([]byte(document))
		hash10 := [10]byte{}
		copy(hash10[:], hash[:10])
		signedAt := doc.SignedAt
		doc.ID = cdid.New(hash10, signedAt).String()

		// check existence
		_, err := s.repository.GetTimeline(ctx, doc.ID)
		if err == nil {
			return core.Timeline{}, fmt.Errorf("Timeline already exists: %s", doc.ID)
		}

		policyResult, err := s.policy.TestWithPolicyURL(
			ctx,
			"",
			core.RequestContext{
				Requester: signer,
				Document:  doc,
			},
			"timeline.create",
		)
		if err != nil {
			return core.Timeline{}, err
		}

		result := s.policy.Summerize([]core.PolicyEvalResult{policyResult}, "timeline.create")
		if !result {
			return core.Timeline{}, fmt.Errorf("You don't have timeline.create access")
		}

	} else { // Update
		id, err := s.NormalizeTimelineID(ctx, doc.ID)
		if err != nil {
			return core.Timeline{}, err
		}
		split := strings.Split(id, "@")
		if len(split) >= 1 {
			if split[len(split)-1] != s.config.FQDN {
				return core.Timeline{}, fmt.Errorf("This timeline is not owned by this domain")
			}
			doc.ID = split[0]
		}

		existance, err := s.repository.GetTimeline(ctx, doc.ID)
		if err != nil {
			span.RecordError(err)
			return core.Timeline{}, err
		}

		doc.DomainOwned = existance.DomainOwned // make sure the domain owned is immutable

		var params map[string]any = make(map[string]any)
		if existance.PolicyParams != nil {
			json.Unmarshal([]byte(*existance.PolicyParams), &params)
		}

		policyResult, err := s.policy.TestWithPolicyURL(
			ctx,
			existance.Policy,
			core.RequestContext{
				Requester: signer,
				Self:      existance,
				Document:  doc,
				Params:    params,
			},
			"timeline.update",
		)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}

		result := s.policy.Summerize([]core.PolicyEvalResult{policyResult}, "timeline.update")
		if !result {
			return core.Timeline{}, fmt.Errorf("You don't have timeline.update access")
		}
	}

	var policyparams *string = nil
	if doc.PolicyParams != "" {
		policyparams = &doc.PolicyParams
	}

	saved, err := s.repository.UpsertTimeline(ctx, core.Timeline{
		ID:           doc.ID,
		Indexable:    doc.Indexable,
		Author:       doc.Signer,
		DomainOwned:  doc.DomainOwned,
		Schema:       doc.Schema,
		Policy:       doc.Policy,
		PolicyParams: policyparams,
		Document:     document,
		Signature:    signature,
	})

	if err != nil {
		return core.Timeline{}, err
	}

	if doc.SemanticID != "" {
		_, err = s.semanticid.Name(ctx, doc.SemanticID, doc.Signer, saved.ID, document, signature)
		if err != nil {
			return core.Timeline{}, err
		}
	}

	saved.ID = saved.ID + "@" + s.config.FQDN

	return saved, nil
}

// Get returns timeline information by ID
func (s *service) GetTimeline(ctx context.Context, key string) (core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.GetTimeline")
	defer span.End()

	split := strings.Split(key, "@")
	id := split[0]
	domain := split[len(split)-1]
	userid := split[len(split)-1]
	if len(split) == 3 {
		userid = split[1]
	}
	if len(split) >= 2 {
		if domain == s.config.FQDN {
			return s.repository.GetTimeline(ctx, id)
		} else {
			if cdid.IsSeemsCDID(split[0], 't') {
				timeline, err := s.repository.GetTimeline(ctx, id)
				if err == nil {
					return timeline, nil
				}
			}
			targetID, err := s.semanticid.Lookup(ctx, id, userid)
			if err != nil {
				return core.Timeline{}, err
			}
			return s.repository.GetTimeline(ctx, targetID)
		}
	} else {
		return s.repository.GetTimeline(ctx, key)
	}
}

// TimelineListBySchema returns timelineList by schema
func (s *service) ListTimelineBySchema(ctx context.Context, schema string) ([]core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.ListTimelineBySchema")
	defer span.End()

	timelines, err := s.repository.ListTimelineBySchema(ctx, schema)
	for i := 0; i < len(timelines); i++ {
		timelines[i].ID = timelines[i].ID + "@" + s.config.FQDN
	}
	return timelines, err
}

// TimelineListByAuthor returns timelineList by author
func (s *service) ListTimelineByAuthor(ctx context.Context, author string) ([]core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.ListTimelineByAuthor")
	defer span.End()

	timelines, err := s.repository.ListTimelineByAuthor(ctx, author)
	for i := 0; i < len(timelines); i++ {
		timelines[i].ID = timelines[i].ID + "@" + s.config.FQDN
	}
	return timelines, err
}

// GetItem returns timeline element by ID
func (s *service) GetItem(ctx context.Context, timeline string, id string) (core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.GetItem")
	defer span.End()

	return s.repository.GetItem(ctx, timeline, id)
}

// Retract removes timeline element by ID
func (s *service) Retract(ctx context.Context, mode core.CommitMode, document, signature string) (core.TimelineItem, []string, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.Retract")
	defer span.End()

	var doc core.RetractDocument
	err := json.Unmarshal([]byte(document), &doc)
	if err != nil {
		return core.TimelineItem{}, []string{}, err
	}

	existing, err := s.repository.GetItem(ctx, doc.Timeline, doc.Target)
	if err != nil {
		return core.TimelineItem{}, []string{}, err
	}

	signer, err := s.entity.Get(ctx, doc.Signer)
	if err != nil {
		span.RecordError(err)
		return core.TimelineItem{}, []string{}, err
	}

	timeline, err := s.repository.GetTimeline(ctx, doc.ID)
	if err != nil {
		span.RecordError(err)
		return core.TimelineItem{}, []string{}, err
	}

	var params map[string]any = make(map[string]any)
	if timeline.PolicyParams != nil {
		json.Unmarshal([]byte(*timeline.PolicyParams), &params)
	}

	policyResult, err := s.policy.TestWithPolicyURL(
		ctx,
		timeline.Policy,
		core.RequestContext{
			Requester: signer,
			Self:      timeline,
			Resource:  existing,
			Document:  doc,
			Params:    params,
		},
		"timeline.retract",
	)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}

	result := s.policy.Summerize([]core.PolicyEvalResult{policyResult}, "timeline.retract")
	if !result {
		return core.TimelineItem{}, []string{}, fmt.Errorf("You don't have timeline.retract access")
	}

	s.repository.DeleteItem(ctx, doc.Timeline, doc.Target)

	affected := []string{timeline.Author}
	if timeline.DomainOwned {
		affected = []string{s.config.FQDN}
	}

	return existing, affected, nil
}

// Delete deletes
func (s *service) DeleteTimeline(ctx context.Context, mode core.CommitMode, document string) (core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.DeleteTimeline")
	defer span.End()

	var doc core.DeleteDocument
	err := json.Unmarshal([]byte(document), &doc)
	if err != nil {
		span.RecordError(err)
		return core.Timeline{}, err
	}

	deleteTarget, err := s.repository.GetTimeline(ctx, doc.Target)
	if err != nil {
		span.RecordError(err)
		return core.Timeline{}, err
	}

	signer, err := s.entity.Get(ctx, doc.Signer)
	if err != nil {
		span.RecordError(err)
		return core.Timeline{}, err
	}

	var params map[string]any = make(map[string]any)
	if deleteTarget.PolicyParams != nil {
		json.Unmarshal([]byte(*deleteTarget.PolicyParams), &params)
	}

	policyResult, err := s.policy.TestWithPolicyURL(
		ctx,
		deleteTarget.Policy,
		core.RequestContext{
			Requester: signer,
			Self:      deleteTarget,
			Document:  doc,
		},
		"timeline.delete",
	)
	if err != nil {
		span.RecordError(err)
		return core.Timeline{}, err
	}

	result := s.policy.Summerize([]core.PolicyEvalResult{policyResult}, "timeline.delete")
	if !result {
		return core.Timeline{}, errors.New("policy failed")
	}

	err = s.repository.DeleteTimeline(ctx, doc.Target)
	if err != nil {
		span.RecordError(err)
		return core.Timeline{}, err
	}

	return deleteTarget, err
}

func (s *service) ListTimelineSubscriptions(ctx context.Context) (map[string]int64, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.ListTimelineSubscriptions")
	defer span.End()

	return s.repository.ListTimelineSubscriptions(ctx)
}

func (s *service) GetTimelineAutoDomain(ctx context.Context, timelineID string) (core.Timeline, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.getTimelineAutoDomain")
	defer span.End()

	normalized, err := s.NormalizeTimelineID(ctx, timelineID)
	if err != nil {
		return core.Timeline{}, err
	}

	key := normalized
	host := s.config.FQDN

	split := strings.Split(normalized, "@")
	if len(split) > 1 {
		key = split[0]
		host = split[len(split)-1]
	}

	if host == s.config.FQDN {
		return s.repository.GetTimeline(ctx, key)
	} else {
		return s.repository.GetTimelineFromRemote(ctx, host, key)
	}
}

func (s *service) Realtime(ctx context.Context, request <-chan []string, response chan<- core.Event) {

	atomic.AddInt64(&s.socketCounter, 1)
	defer atomic.AddInt64(&s.socketCounter, -1)

	var cancel context.CancelFunc
	events := make(chan core.Event)

	var mapper map[string]string

	for {
		select {
		case timelines := <-request:
			if cancel != nil {
				cancel()
			}

			normalized := make([]string, 0)
			mapper = make(map[string]string)
			for _, timeline := range timelines {
				normalizedTimeline, err := s.NormalizeTimelineID(ctx, timeline)
				if err != nil {
					slog.WarnContext(
						ctx,
						fmt.Sprintf("failed to normalize timeline: %s", timeline),
						slog.String("module", "timeline"),
					)
					continue
				}
				normalized = append(normalized, normalizedTimeline)
				mapper[normalizedTimeline] = timeline
			}

			var subctx context.Context
			subctx, cancel = context.WithCancel(ctx)
			go s.repository.Subscribe(subctx, normalized, events)
		case event := <-events:
			if mapper == nil {
				slog.WarnContext(ctx, "mapper is nil", slog.String("module", "timeline"))
				continue
			}
			event.Timeline = mapper[event.Timeline]
			response <- event
		case <-ctx.Done():
			if cancel != nil {
				cancel()
			}
			return
		}
	}
}

func (s *service) GetOwners(ctx context.Context, timelines []string) ([]string, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.GetOwners")
	defer span.End()

	var owners_map map[string]bool = make(map[string]bool)
	for _, timelineID := range timelines {
		timeline, err := s.GetTimeline(ctx, timelineID)
		if err != nil {
			continue
		}
		if timeline.DomainOwned {
			owners_map[s.config.FQDN] = true
		} else {
			owners_map[timeline.Author] = true
		}
	}

	owners := make([]string, 0)
	for owner := range owners_map {
		owners = append(owners, owner)
	}

	return owners, nil
}

func (s *service) Clean(ctx context.Context, ccid string) error {
	ctx, span := tracer.Start(ctx, "Timeline.Service.Clean")
	defer span.End()

	timelines, err := s.repository.ListTimelineByAuthorOwned(ctx, ccid)
	if err != nil {
		span.RecordError(err)
		return err
	}

	for _, timeline := range timelines {
		err := s.repository.DeleteTimeline(ctx, timeline.ID)
		if err != nil {
			span.RecordError(err)
			return err
		}
	}

	return nil
}

func (s *service) Query(ctx context.Context, timelineID, schema, owner, author string, since time.Time, limit int) ([]core.TimelineItem, error) {
	ctx, span := tracer.Start(ctx, "Timeline.Service.Query")
	defer span.End()

	normalized, err := s.NormalizeTimelineID(ctx, timelineID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	split := strings.Split(normalized, "@")
	host := split[len(split)-1]
	if host != s.config.FQDN {
		return nil, fmt.Errorf("Remote timeline is not supported")
	}

	id := split[0]

	items, err := s.repository.Query(ctx, id, schema, owner, author, since, limit)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return items, nil
}
