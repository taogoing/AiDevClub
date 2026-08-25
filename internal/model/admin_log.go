package model

import "time"

type AdminLogAction string

const (
	AdminLogActionApproveResource    AdminLogAction = "approve_resource"
	AdminLogActionRejectResource     AdminLogAction = "reject_resource"
	AdminLogActionHideContent        AdminLogAction = "hide_content"
	AdminLogActionUnhideContent      AdminLogAction = "unhide_content"
	AdminLogActionCreateTag          AdminLogAction = "create_tag"
	AdminLogActionUpdateTag          AdminLogAction = "update_tag"
	AdminLogActionCreateAnnouncement AdminLogAction = "create_announcement"
	AdminLogActionResolveReport      AdminLogAction = "resolve_report"
)

type AdminLog struct {
	ID         uint           `gorm:"primaryKey"`
	AdminID    uint           `gorm:"not null;index:idx_admin"`
	Action     AdminLogAction `gorm:"size:32;not null;index:idx_action"`
	TargetType string         `gorm:"size:32"`
	TargetID   uint
	Detail     string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"index:idx_created_at"`
}
