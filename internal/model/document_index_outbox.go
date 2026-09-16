package model

import "time"

type DocumentIndexOutboxEvent struct {
	ID          uint   `gorm:"primaryKey"`
	EventID     string `gorm:"size:64;not null;uniqueIndex"`
	Operation   string `gorm:"size:16;not null"`
	ArticleID   uint   `gorm:"not null;index"`
	Payload     []byte `gorm:"type:json;not null"`
	PublishedAt *time.Time
	CreatedAt   time.Time `gorm:"index"`
}
