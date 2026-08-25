package model

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

type User struct {
	ID           uint           `gorm:"primaryKey"`
	Email        string         `gorm:"size:191;uniqueIndex;not null"`
	PasswordHash string         `gorm:"size:255;not null"`
	Nickname     string         `gorm:"size:64;not null"`
	AvatarURL    string         `gorm:"size:255;not null"`
	Bio          string         `gorm:"type:text"`
	Role         UserRole       `gorm:"size:16;not null;default:user"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
