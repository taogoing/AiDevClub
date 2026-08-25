package model

import (
	"time"

	"gorm.io/gorm"
)

type SkillLike struct {
	ID        uint `gorm:"primaryKey"`
	SkillID   uint `gorm:"uniqueIndex:uniq_skill_like;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_skill_like;not null"`
	CreatedAt time.Time
}

type SkillFavorite struct {
	ID        uint `gorm:"primaryKey"`
	SkillID   uint `gorm:"uniqueIndex:uniq_skill_fav;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_skill_fav;not null"`
	CreatedAt time.Time
}

type McpServerLike struct {
	ID          uint `gorm:"primaryKey"`
	McpServerID uint `gorm:"uniqueIndex:uniq_mcp_server_like;not null"`
	UserID      uint `gorm:"uniqueIndex:uniq_mcp_server_like;not null"`
	CreatedAt   time.Time
}

type McpServerFavorite struct {
	ID          uint `gorm:"primaryKey"`
	McpServerID uint `gorm:"uniqueIndex:uniq_mcp_server_fav;not null"`
	UserID      uint `gorm:"uniqueIndex:uniq_mcp_server_fav;not null"`
	CreatedAt   time.Time
}

type ResourceComment struct {
	ID           uint           `gorm:"primaryKey"`
	ResourceType string         `gorm:"size:16;not null;index:idx_resource"`
	ResourceID   uint           `gorm:"not null;index:idx_resource"`
	AuthorID     uint           `gorm:"not null;index"`
	ParentID     *uint          `gorm:"index"`
	ReplyToID    *uint          `gorm:"index"`
	Content      string         `gorm:"type:text;not null"`
	LikesCount   int            `gorm:"not null;default:0"`
	Hidden       bool           `gorm:"not null;default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	Author *User `gorm:"foreignKey:AuthorID"`
}

type ResourceCommentLike struct {
	ID        uint `gorm:"primaryKey"`
	CommentID uint `gorm:"uniqueIndex:uniq_res_comment_like;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_res_comment_like;not null"`
	CreatedAt time.Time
}
