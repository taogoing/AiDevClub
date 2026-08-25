package model

import "time"

type ReportReason string

const (
	ReportReasonSpam     ReportReason = "spam"
	ReportReasonAbuse    ReportReason = "abuse"
	ReportReasonCopyright ReportReason = "copyright"
	ReportReasonOther    ReportReason = "other"
)

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

type Report struct {
	ID           uint         `gorm:"primaryKey"`
	ReporterID   uint         `gorm:"not null"`
	TargetType   string       `gorm:"size:32;not null;index:idx_target"`
	TargetID     uint         `gorm:"not null;index:idx_target"`
	Reason       ReportReason `gorm:"size:32;not null"`
	Description  string       `gorm:"type:text"`
	Status       ReportStatus `gorm:"size:16;not null;default:pending;index:idx_status"`
	HandlerID    uint
	HandleResult string       `gorm:"type:text"`
	CreatedAt    time.Time
	ResolvedAt   *time.Time
}
