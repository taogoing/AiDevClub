package model

import "time"

type Announcement struct {
	ID        uint      `gorm:"primaryKey"`
	Title     string    `gorm:"size:200;not null"`
	Content   string    `gorm:"type:text;not null"`
	AdminID   uint      `gorm:"not null"`
	CreatedAt time.Time `gorm:"index:idx_created_at"`
}
