package model

import (
	"time"

	"gorm.io/gorm"
)

type ResourceStatus string

const (
	ResourceStatusDraft         ResourceStatus = "draft"
	ResourceStatusPendingReview ResourceStatus = "pending_review"
	ResourceStatusPublished     ResourceStatus = "published"
	ResourceStatusRejected      ResourceStatus = "rejected"
	ResourceStatusArchived      ResourceStatus = "archived"
)

type Skill struct {
	ID             uint           `gorm:"primaryKey"`
	AuthorID       uint           `gorm:"not null;index"`
	Name           string         `gorm:"size:100;not null"`
	Description    string         `gorm:"size:500"`
	RepoURL        string         `gorm:"size:255"`
	ZipURL         string         `gorm:"size:255"`
	ZipFilename    string         `gorm:"size:255"`
	FileSize       int64          `gorm:"not null;default:0"`
	SkillMD        string         `gorm:"type:mediumtext"`
	Status         ResourceStatus `gorm:"size:16;not null;default:draft;index"`
	Views          int            `gorm:"not null;default:0"`
	Downloads      int            `gorm:"not null;default:0"`
	LikesCount     int            `gorm:"not null;default:0"`
	FavoritesCount int            `gorm:"not null;default:0"`
	CommentsCount  int            `gorm:"not null;default:0"`
	Pinned         bool           `gorm:"not null;default:false"`
	Hidden         bool           `gorm:"not null;default:false"`
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	Author *User `gorm:"foreignKey:AuthorID"`
}

type SkillTag struct {
	ID        uint `gorm:"primaryKey"`
	SkillID   uint `gorm:"uniqueIndex:uniq_skill_tag;not null"`
	TagID     uint `gorm:"uniqueIndex:uniq_skill_tag;not null"`
	CreatedAt time.Time
}
