package repo

import (
	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(u *model.User) error { return r.db.Create(u).Error }

func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Update(u *model.User) error {
	return r.db.Save(u).Error
}

func (r *UserRepo) Delete(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}
