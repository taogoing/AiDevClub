package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

const notificationEventType = "notification.created"

type NotificationOutboxPublisher interface {
	Publish(context.Context, *model.NotificationOutboxEvent) error
}

type NotificationOutboxWorker struct {
	repo      *repo.NotificationOutboxRepo
	publisher NotificationOutboxPublisher
	interval  time.Duration
	lease     time.Duration
	now       func() time.Time
}

func NewNotificationOutboxWorker(outbox *repo.NotificationOutboxRepo, publisher NotificationOutboxPublisher, interval, lease time.Duration) *NotificationOutboxWorker {
	return &NotificationOutboxWorker{repo: outbox, publisher: publisher, interval: interval, lease: lease, now: time.Now}
}

func (w *NotificationOutboxWorker) Run(ctx context.Context) error {
	if _, err := w.RunBatch(ctx, 100); err != nil {
		return err
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := w.RunBatch(ctx, 100); err != nil {
				return err
			}
		}
	}
}

func (w *NotificationOutboxWorker) RunOnce(ctx context.Context) error {
	_, err := w.RunBatch(ctx, 1)
	return err
}

// RunBatch publishes up to limit eligible events without waiting for another poll interval.
func (w *NotificationOutboxWorker) RunBatch(ctx context.Context, limit int) (int, error) {
	if limit < 1 {
		return 0, nil
	}
	processed := 0
	for processed < limit {
		count, err := w.runOne(ctx)
		if err != nil {
			return processed, err
		}
		if count == 0 {
			break
		}
		processed += count
	}
	return processed, nil
}

func (w *NotificationOutboxWorker) runOne(ctx context.Context) (int, error) {
	token, err := newEventID()
	if err != nil {
		return 0, err
	}
	now := w.now()
	event, err := w.repo.ClaimNext(ctx, now, w.lease, token)
	if err != nil {
		return 0, err
	}
	if event == nil {
		return 0, nil
	}
	if err := w.publisher.Publish(ctx, event); err != nil {
		if markErr := w.repo.MarkRetry(ctx, event.ID, token, err, w.now()); markErr != nil {
			return 0, fmt.Errorf("publish outbox event %s: %v; record retry: %w", event.EventID, err, markErr)
		}
		return 0, fmt.Errorf("publish outbox event %s: %w", event.EventID, err)
	}
	if err := w.repo.MarkPublished(ctx, event.ID, token, w.now()); err != nil {
		return 0, fmt.Errorf("mark outbox event %s published: %w", event.EventID, err)
	}
	return 1, nil
}

type NotificationEventConsumer struct{ notifications *repo.NotificationRepo }

func NewNotificationEventConsumer(notifications *repo.NotificationRepo) *NotificationEventConsumer {
	return &NotificationEventConsumer{notifications: notifications}
}

func (c *NotificationEventConsumer) Process(ctx context.Context, payload []byte) error {
	var event notificationEventPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode notification event: %w", err)
	}
	if event.EventID == "" || event.Data.UserID == 0 || event.Data.Type == "" || event.Data.Title == "" {
		return fmt.Errorf("notification event is missing event id or required data")
	}
	n := &model.Notification{
		EventID: &event.EventID, UserID: event.Data.UserID, Type: event.Data.Type,
		Title: event.Data.Title, Content: event.Data.Content,
		ResourceType: event.Data.ResourceType, ResourceID: event.Data.ResourceID, ActorID: event.Data.ActorID,
	}
	return c.notifications.CreateFromOutboxEvent(ctx, n)
}

func processBeforeAck(ctx context.Context, payload []byte, process func(context.Context, []byte) error, ack func() error) error {
	if err := process(ctx, payload); err != nil {
		return err
	}
	return ack()
}
