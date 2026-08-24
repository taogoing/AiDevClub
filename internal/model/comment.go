package model

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	ID         uint           `gorm:"primaryKey"`
	ArticleID  uint           `gorm:"not null;index"`
	AuthorID   uint           `gorm:"not null;index"`
	ParentID   *uint          `gorm:"index"`
	Content    string         `gorm:"type:text;not null"`
	LikesCount int            `gorm:"not null;default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
