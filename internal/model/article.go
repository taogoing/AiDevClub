package model

import (
	"time"

	"gorm.io/gorm"
)

type ArticleStatus string

const (
	ArticleStatusDraft     ArticleStatus = "draft"
	ArticleStatusPublished ArticleStatus = "published"
)

type Article struct {
	ID             uint          `gorm:"primaryKey"`
	AuthorID       uint          `gorm:"not null;index"`
	CategoryID     uint          `gorm:"not null;index"`
	Title          string        `gorm:"size:200;not null"`
	Summary        string        `gorm:"size:500"`
	Content        string        `gorm:"type:mediumtext"`
	Status         ArticleStatus `gorm:"size:16;not null;default:draft;index"`
	Views          int           `gorm:"not null;default:0"`
	LikesCount     int           `gorm:"not null;default:0"`
	FavoritesCount int           `gorm:"not null;default:0"`
	CommentsCount  int           `gorm:"not null;default:0"`
	Pinned         bool          `gorm:"not null;default:false"`
	Hidden         bool          `gorm:"not null;default:false"`
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	Author   *User     `gorm:"foreignKey:AuthorID"`
	Category *Category `gorm:"foreignKey:CategoryID"`
}

type ArticleTag struct {
	ID        uint `gorm:"primaryKey"`
	ArticleID uint `gorm:"uniqueIndex:uniq_article_tag;not null"`
	TagID     uint `gorm:"uniqueIndex:uniq_article_tag;not null"`
	CreatedAt time.Time
}
