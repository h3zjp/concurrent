package notification

import (
	"context"

	"gorm.io/gorm"

	"github.com/totegamma/concurrent/core"
)

type Repo interface {
	Get(ctx context.Context, vendorID, owner string) (core.NotificationSubscription, error)
	Subscribe(ctx context.Context, notification core.NotificationSubscription) (core.NotificationSubscription, error)
	GetAllSubscriptions(ctx context.Context) ([]core.NotificationSubscription, error)
	Delete(ctx context.Context, vendorID, owner string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repo {
	return &repository{db}
}

func (r *repository) Subscribe(ctx context.Context, notification core.NotificationSubscription) (core.NotificationSubscription, error) {
	ctx, span := tracer.Start(ctx, "Notification.Repository.Subscribe")
	defer span.End()

	if err := r.db.WithContext(ctx).Save(&notification).Error; err != nil {
		return core.NotificationSubscription{}, err
	}

	return notification, nil
}

func (r *repository) GetAllSubscriptions(ctx context.Context) ([]core.NotificationSubscription, error) {
	ctx, span := tracer.Start(ctx, "Notification.Repository.GetAllSubscriptions")
	defer span.End()

	var notifications []core.NotificationSubscription
	err := r.db.Find(&notifications).Error
	if err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *repository) Delete(ctx context.Context, vendorID, owner string) error {
	ctx, span := tracer.Start(ctx, "Notification.Repository.Delete")
	defer span.End()

	err := r.db.WithContext(ctx).Where("vendor_id = ? AND owner = ?", vendorID, owner).Delete(&core.NotificationSubscription{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) Get(ctx context.Context, vendorID, owner string) (core.NotificationSubscription, error) {
	ctx, span := tracer.Start(ctx, "Notification.Repository.Get")
	defer span.End()

	var notification core.NotificationSubscription
	err := r.db.WithContext(ctx).Where("vendor_id = ? AND owner = ?", vendorID, owner).First(&notification).Error
	if err != nil {
		return core.NotificationSubscription{}, err
	}

	return notification, nil
}
