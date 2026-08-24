package model

import (
	"time"

	"gorm.io/gorm"
)

type McpServer struct {
	ID             uint           `gorm:"primaryKey"`
	AuthorID       uint           `gorm:"not null;index"`
	Name           string         `gorm:"size:100;not null"`
	Description    string         `gorm:"size:500"`
	RepoURL        string         `gorm:"size:255"`
	ToolsJSON      string         `gorm:"type:json"`
	Readme         string         `gorm:"type:mediumtext"`
	ZipURL         string         `gorm:"size:255"`
	ZipFilename    string         `gorm:"size:255"`
	FileSize       int64          `gorm:"not null;default:0"`
	Status         ResourceStatus `gorm:"size:16;not null;default:draft;index"`
	Views          int            `gorm:"not null;default:0"`
	Downloads      int            `gorm:"not null;default:0"`
	LikesCount     int            `gorm:"not null;default:0"`
	FavoritesCount int            `gorm:"not null;default:0"`
	CommentsCount  int            `gorm:"not null;default:0"`
	Pinned         bool           `gorm:"not null;default:false"`
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	Author *User `gorm:"foreignKey:AuthorID"`
}

type McpServerTag struct {
	ID          uint `gorm:"primaryKey"`
	McpServerID uint `gorm:"uniqueIndex:uniq_mcp_server_tag;not null"`
	TagID       uint `gorm:"uniqueIndex:uniq_mcp_server_tag;not null"`
	CreatedAt   time.Time
}
