package association

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/totegamma/concurrent/x/core"
	"github.com/totegamma/concurrent/x/message"
	"github.com/totegamma/concurrent/x/stream"
	"github.com/totegamma/concurrent/x/util"
)

// Service is the interface for association service
type Service interface {
	PostAssociation(ctx context.Context, objectStr string, signature string, streams []string, targetType string) (core.Association, error)
	Get(ctx context.Context, id string) (core.Association, error)
	GetOwn(ctx context.Context, author string) ([]core.Association, error)
	Delete(ctx context.Context, id string) (core.Association, error)
}

type service struct {
	rdb     *redis.Client
	repo    Repository
	stream  stream.Service
	message message.Service
}

// NewService creates a new association service
func NewService(rdb *redis.Client, repo Repository, stream stream.Service, message message.Service) Service {
	return &service{rdb, repo, stream, message}
}

// PostAssociation creates a new association
// If targetType is messages, it also posts the association to the target message's streams
// returns the created association
func (s *service) PostAssociation(ctx context.Context, objectStr string, signature string, streams []string, targetType string) (core.Association, error) {
	ctx, span := tracer.Start(ctx, "ServicePostAssociation")
	defer span.End()

	var object SignedObject
	err := json.Unmarshal([]byte(objectStr), &object)
	if err != nil {
		span.RecordError(err)
		return core.Association{}, err
	}

	if err := util.VerifySignature(objectStr, object.Signer, signature); err != nil {
		span.RecordError(err)
		return core.Association{}, err
	}

	var content SignedObject
	err = json.Unmarshal([]byte(objectStr), &content)
	if err != nil {
		span.RecordError(err)
		return core.Association{}, err
	}

	contentString, err := json.Marshal(content.Body)
	if err != nil {
		span.RecordError(err)
		return core.Association{}, err
	}

	hash := sha256.Sum256(contentString)
	contentHash := hex.EncodeToString(hash[:])

	association := core.Association{
		Author:      object.Signer,
		Schema:      object.Schema,
		TargetID:    object.Target,
		TargetType:  targetType,
		Payload:     objectStr,
		Signature:   signature,
		Streams:     streams,
		ContentHash: contentHash,
	}

	created, err := s.repo.Create(ctx, association)
	if err != nil {
		span.RecordError(err)
		return created, err // TODO: if err is duplicate key error, server should return 409
	}

	if targetType != "messages" { // distribute is needed only when targetType is messages
		return created, nil
	}

	targetMessage, err := s.message.Get(ctx, created.TargetID)
	if err != nil {
		span.RecordError(err)
		return created, err
	}

	item := core.StreamItem{
		Type:     "association",
		ObjectID: created.ID,
		Owner:    targetMessage.Author,
		Author:   created.Author,
	}

	for _, stream := range association.Streams {
		err = s.stream.PostItem(ctx, stream, item, created)
		if err != nil {
			span.RecordError(err)
			log.Printf("fail to post stream: %v", err)
		}
	}

	for _, postto := range targetMessage.Streams {
		jsonstr, _ := json.Marshal(stream.Event{
			Stream: postto,
			Action: "create",
			Type:   "association",
			Item:   item,
			Body:   created,
		})
		err := s.rdb.Publish(context.Background(), postto, jsonstr).Err()
		if err != nil {
			span.RecordError(err)
			log.Printf("fail to publish message to Redis: %v", err)
		}
	}

	return created, nil
}

// Get returns an association by ID
func (s *service) Get(ctx context.Context, id string) (core.Association, error) {
	ctx, span := tracer.Start(ctx, "ServiceGet")
	defer span.End()

	return s.repo.Get(ctx, id)
}

// GetOwn returns associations by author
func (s *service) GetOwn(ctx context.Context, author string) ([]core.Association, error) {
	ctx, span := tracer.Start(ctx, "ServiceGetOwn")
	defer span.End()

	return s.repo.GetOwn(ctx, author)
}

// Delete deletes an association by ID
func (s *service) Delete(ctx context.Context, id string) (core.Association, error) {
	ctx, span := tracer.Start(ctx, "ServiceDelete")
	defer span.End()

	deleted, err := s.repo.Delete(ctx, id)
	if err != nil {
		span.RecordError(err)
		return core.Association{}, err
	}

	if deleted.TargetType != "messages" { // distribute is needed only when targetType is messages
		return deleted, nil
	}

	targetMessage, err := s.message.Get(ctx, deleted.TargetID)
	if err != nil {
		span.RecordError(err)
		return deleted, err
	}
	for _, posted := range targetMessage.Streams {
		jsonstr, _ := json.Marshal(stream.Event{
			Stream: posted,
			Type:   "association",
			Action: "delete",
			Body:   deleted,
		})
		err := s.rdb.Publish(context.Background(), posted, jsonstr).Err()
		if err != nil {
			log.Printf("fail to publish message to Redis: %v", err)
			span.RecordError(err)
			return deleted, err
		}
	}
	return deleted, nil
}
