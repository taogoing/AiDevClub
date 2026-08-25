package model

import "time"

type NotifType string

const (
	NotifTypeCommentArticle      NotifType = "comment_article"
	NotifTypeReplyComment        NotifType = "reply_comment"
	NotifTypeLikeArticle         NotifType = "like_article"
	NotifTypeLikeSkill           NotifType = "like_skill"
	NotifTypeLikeMcpServer       NotifType = "like_mcp_server"
	NotifTypeLikeComment         NotifType = "like_comment"
	NotifTypeLikeResourceComment NotifType = "like_resource_comment"
	NotifTypeResourceApproved    NotifType = "resource_approved"
	NotifTypeResourceRejected    NotifType = "resource_rejected"
	NotifTypeReportResolved      NotifType = "report_resolved"
	NotifTypeAnnouncement        NotifType = "announcement"
)

type Notification struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"not null;index:idx_user_read"`
	Type         NotifType `gorm:"size:32;not null"`
	Title        string    `gorm:"size:200;not null"`
	Content      string    `gorm:"type:text"`
	ResourceType string    `gorm:"size:32"`
	ResourceID   uint
	ActorID      uint
	IsRead       bool      `gorm:"not null;default:false;index:idx_user_read"`
	CreatedAt    time.Time `gorm:"index:idx_created_at"`
}
