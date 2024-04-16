package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/totegamma/concurrent/x/ack"
	"github.com/totegamma/concurrent/x/association"
	"github.com/totegamma/concurrent/x/core"
	"github.com/totegamma/concurrent/x/entity"
	"github.com/totegamma/concurrent/x/key"
	"github.com/totegamma/concurrent/x/message"
	"github.com/totegamma/concurrent/x/profile"
	"github.com/totegamma/concurrent/x/subscription"
	"github.com/totegamma/concurrent/x/timeline"
)

type Service interface {
	Commit(ctx context.Context, document, signature, option string) (any, error)
}

type service struct {
	key          key.Service
	entity       entity.Service
	message      message.Service
	association  association.Service
	profile      profile.Service
	timeline     timeline.Service
	ack          ack.Service
	subscription subscription.Service
}

func NewService(
	key key.Service,
	entity entity.Service,
	message message.Service,
	association association.Service,
	profile profile.Service,
	timeline timeline.Service,
	ack ack.Service,
	subscription subscription.Service,
) Service {
	return &service{
		key:          key,
		entity:       entity,
		message:      message,
		association:  association,
		profile:      profile,
		timeline:     timeline,
		ack:          ack,
		subscription: subscription,
	}
}

func (s *service) Commit(ctx context.Context, document string, signature string, option string) (any, error) {
	ctx, span := tracer.Start(ctx, "Store.Service.Commit")
	defer span.End()

	var base core.DocumentBase[any]
	err := json.Unmarshal([]byte(document), &base)
	if err != nil {
		return nil, err
	}

	keys, ok := ctx.Value(core.RequesterKeychainKey).([]core.Key)
	if !ok {
		keys = []core.Key{}
	}

	err = s.key.ValidateDocument(ctx, document, signature, keys)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	switch base.Type {
	case "message":
		return s.message.Create(ctx, document, signature)
	case "association":
		return s.association.Create(ctx, document, signature)
	case "profile":
		return s.profile.Upsert(ctx, document, signature)
	case "affiliation":
		return s.entity.Affiliation(ctx, document, signature, option)
	case "tombstone":
		return s.entity.Tombstone(ctx, document, signature)
	case "timeline":
		return s.timeline.UpsertTimeline(ctx, document, signature)
	case "ack", "unack":
		return nil, s.ack.Ack(ctx, document, signature)
	case "subscription":
		return s.subscription.CreateSubscription(ctx, document, signature)
	case "subscribe":
		return s.subscription.Subscribe(ctx, document, signature)
	case "unsubscribe":
		return s.subscription.Unsubscribe(ctx, document)
	case "delete":
		var doc core.DeleteDocument
		err := json.Unmarshal([]byte(document), &doc)
		if err != nil {
			return nil, err
		}
		typ := doc.Target[0]
		switch typ {
		case 'm': // message
			return s.message.Delete(ctx, document, signature)
		case 'a': // association
			return s.association.Delete(ctx, document, signature)
		case 'p': // profile
			return s.profile.Delete(ctx, document)
		case 't': // timeline
			return s.timeline.DeleteTimeline(ctx, document)
		default:
			return nil, fmt.Errorf("unknown document type: %s", string(typ))
		}
	}
	return nil, fmt.Errorf("unknown document type: %s", base.Type)
}