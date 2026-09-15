package model

import "time"

type NotificationOutboxStatus string

const (
	NotificationOutboxPending    NotificationOutboxStatus = "pending"
	NotificationOutboxPublishing NotificationOutboxStatus = "publishing"
	NotificationOutboxPublished  NotificationOutboxStatus = "published"
)

type NotificationOutboxEvent struct {
	ID          uint                     `gorm:"primaryKey"`
	EventID     string                   `gorm:"size:64;not null;uniqueIndex:uniq_notification_outbox_event_id"`
	EventType   string                   `gorm:"size:96;not null;index:idx_notification_outbox_pending,priority:1"`
	Payload     []byte                   `gorm:"type:json;not null"`
	Status      NotificationOutboxStatus `gorm:"size:16;not null;default:pending;index:idx_notification_outbox_pending,priority:2"`
	RetryCount  int                      `gorm:"not null;default:0"`
	NextRetryAt *time.Time               `gorm:"index:idx_notification_outbox_pending,priority:3"`
	LockedUntil *time.Time               `gorm:"index"`
	LeaseToken  string                   `gorm:"size:64"`
	LastError   string                   `gorm:"type:text"`
	CreatedAt   time.Time                `gorm:"index"`
	PublishedAt *time.Time
}
