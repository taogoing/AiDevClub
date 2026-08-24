package repo

import "gorm.io/gorm"

type InteractionRepo struct{ db *gorm.DB }

func NewInteractionRepo(db *gorm.DB) *InteractionRepo { return &InteractionRepo{db: db} }
