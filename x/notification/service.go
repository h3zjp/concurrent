package notification

import (
	"context"

	"github.com/totegamma/concurrent/core"
)

type service struct {
	repo Repo
}

func NewService(repo Repo) core.NotificationService {
	return &service{
		repo,
	}
}

func (s *service) Subscribe(ctx context.Context, subscription core.NotificationSubscription) (core.NotificationSubscription, error) {
	ctx, span := tracer.Start(ctx, "Notification.Service.Subscribe")
	defer span.End()

	subscription, err := s.repo.Subscribe(ctx, subscription)
	if err != nil {
		return core.NotificationSubscription{}, err
	}

	return subscription, nil
}

func (s *service) GetAllSubscriptions(ctx context.Context) ([]core.NotificationSubscription, error) {
	ctx, span := tracer.Start(ctx, "Notification.Service.GetAllSubscriptions")
	defer span.End()

	subscriptions, err := s.repo.GetAllSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (s *service) Delete(ctx context.Context, vendorID, owner string) error {
	ctx, span := tracer.Start(ctx, "Notification.Service.Delete")
	defer span.End()

	err := s.repo.Delete(ctx, vendorID, owner)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) Get(ctx context.Context, vendorID, owner string) (core.NotificationSubscription, error) {
	ctx, span := tracer.Start(ctx, "Notification.Service.Get")
	defer span.End()

	subscription, err := s.repo.Get(ctx, vendorID, owner)
	if err != nil {
		return core.NotificationSubscription{}, err
	}

	return subscription, nil
}
