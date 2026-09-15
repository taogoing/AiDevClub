package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aidevclub/internal/model"
)

type NotificationOutboxRepo struct{ db *gorm.DB }

func NewNotificationOutboxRepo(db *gorm.DB) *NotificationOutboxRepo {
	return &NotificationOutboxRepo{db: db}
}

// Create must receive the business transaction so interaction and notification
// event persistence share the same MySQL commit boundary.
func (r *NotificationOutboxRepo) Create(tx *gorm.DB, event *model.NotificationOutboxEvent) error {
	if tx == nil {
		return fmt.Errorf("notification outbox create requires a transaction")
	}
	return tx.Create(event).Error
}

// ClaimNext uses SKIP LOCKED plus a time-bounded lease. Expired publishing rows
// are reclaimed after a crashed publisher, and the lease token fences stale workers.
func (r *NotificationOutboxRepo) ClaimNext(ctx context.Context, now time.Time, lease time.Duration, token string) (*model.NotificationOutboxEvent, error) {
	var claimed *model.NotificationOutboxEvent
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event model.NotificationOutboxEvent
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("(status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND locked_until <= ?)",
				model.NotificationOutboxPending, now, model.NotificationOutboxPublishing, now).
			Order("id ASC").First(&event).Error
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		lockedUntil := now.Add(lease)
		if err := tx.Model(&event).Updates(map[string]any{
			"status": model.NotificationOutboxPublishing, "locked_until": lockedUntil, "lease_token": token,
		}).Error; err != nil {
			return err
		}
		event.Status = model.NotificationOutboxPublishing
		event.LockedUntil = &lockedUntil
		event.LeaseToken = token
		claimed = &event
		return nil
	})
	return claimed, err
}

func (r *NotificationOutboxRepo) MarkPublished(ctx context.Context, id uint, token string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.NotificationOutboxEvent{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, model.NotificationOutboxPublishing, token).
		Updates(map[string]any{"status": model.NotificationOutboxPublished, "published_at": now, "locked_until": nil, "lease_token": "", "last_error": ""})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("notification outbox lease lost for event %d", id)
	}
	return nil
}

func (r *NotificationOutboxRepo) MarkRetry(ctx context.Context, id uint, token string, cause error, now time.Time) error {
	var event model.NotificationOutboxEvent
	if err := r.db.WithContext(ctx).Where("id = ? AND status = ? AND lease_token = ?", id, model.NotificationOutboxPublishing, token).First(&event).Error; err != nil {
		return err
	}
	delay := time.Second * time.Duration(1<<min(event.RetryCount, 8))
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	message := "unknown publish error"
	if cause != nil {
		message = cause.Error()
	}
	result := r.db.WithContext(ctx).Model(&event).
		Where("id = ? AND status = ? AND lease_token = ?", id, model.NotificationOutboxPublishing, token).
		Updates(map[string]any{
			"status": model.NotificationOutboxPending, "retry_count": event.RetryCount + 1,
			"next_retry_at": now.Add(delay), "last_error": strings.TrimSpace(message),
			"locked_until": nil, "lease_token": "",
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("notification outbox lease lost for event %d", id)
	}
	return nil
}
