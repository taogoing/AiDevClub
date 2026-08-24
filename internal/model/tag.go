package model

import "time"

type Tag struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"size:64;uniqueIndex;not null"`
	UsageCount int    `gorm:"not null;default:0"`
	Enabled    bool   `gorm:"not null;default:true"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
