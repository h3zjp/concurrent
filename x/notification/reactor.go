package notification

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"time"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/totegamma/concurrent/core"
)

type reactor struct {
	service  core.NotificationService
	timeline core.TimelineService
	opts     webpush.Options
}

func NewReactor(service core.NotificationService, timeline core.TimelineService, opts webpush.Options) Reactor {
	return &reactor{
		service:  service,
		timeline: timeline,
		opts:     opts,
	}
}

type Reactor interface {
	Start(ctx context.Context)
}

type Worker struct {
	MDate   time.Time
	Routine context.CancelFunc
}

func (r *reactor) Start(ctx context.Context) {
	slog.Info("starting reactor")

	ticker10 := time.NewTicker(10 * time.Second)
	workers := make(map[string]Worker)

	go func() {
		for ; true; <-ticker10.C {
			slog.Info("checking subscriptions")

			subscriptions, err := r.service.GetAllSubscriptions(ctx)
			if err != nil {
				slog.Error("error getting subscriptions", slog.String("error", err.Error()))
				continue
			}

			for _, sub := range subscriptions {

				subID := sub.VendorID + sub.Owner
				existingWorker, ok := workers[subID]
				if ok {
					if existingWorker.MDate == sub.MDate {
						slog.Info("worker already running", slog.String("vendorID", sub.VendorID), slog.String("owner", sub.Owner))
						continue
					} else {
						existingWorker.Routine()
						delete(workers, subID)
					}
				}

				slog.Info("starting worker", slog.String("vendorID", sub.VendorID), slog.String("owner", sub.Owner))

				workerctx, cancel := context.WithCancel(ctx)
				workers[subID] = Worker{
					MDate:   sub.MDate,
					Routine: cancel,
				}

				go func(ctx context.Context, sub core.NotificationSubscription) {

					slog.Info("worker started", slog.String("vendorID", sub.VendorID), slog.String("owner", sub.Owner))

					var subscription webpush.Subscription
					json.Unmarshal([]byte(sub.Subscription), &subscription)

					request := make(chan []string)
					realtime := make(chan core.Event)

					go r.timeline.Realtime(ctx, request, realtime)

					request <- sub.Timelines

					for {
						select {
						case <-ctx.Done():
							close(request)
							close(realtime)
							return
						case event := <-realtime:

							var doc core.DocumentBase[any]
							err := json.Unmarshal([]byte(event.Document), &doc)
							if err != nil {
								slog.Error("error unmarshalling document", slog.String("error", err.Error()))
								continue
							}

							if !slices.Contains(sub.Schemas, doc.Schema) {
								slog.Info("schema not in subscription", slog.String("schema", doc.Schema))
								continue
							}

							// Send Notification
							resp, err := webpush.SendNotification([]byte(event.Document), &subscription, &r.opts)
							if err != nil {
								slog.Error("error sending notification", slog.String("error", err.Error()))
								continue
							}
							defer resp.Body.Close()
						}
					}
				}(workerctx, sub)

			}

			var validSubs []string
			for _, sub := range subscriptions {
				validSubs = append(validSubs, sub.VendorID+sub.Owner)
			}

			for id, worker := range workers {
				if !slices.Contains(validSubs, id) {
					slog.Info("stopping worker", slog.String("id", id))
					worker.Routine()
					delete(workers, id)
				}
			}
		}
	}()
}
