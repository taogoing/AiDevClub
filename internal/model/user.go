package model

import "time"

type User struct {
	ID           uint       `gorm:"primaryKey"`
	Email        string     `gorm:"size:191;uniqueIndex;not null"`
	PasswordHash string     `gorm:"size:255;not null"`
	Nickname     string     `gorm:"size:64;not null"`
	AvatarURL    string     `gorm:"size:255;not null"`
	Bio          string     `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time `gorm:"index"`
}
