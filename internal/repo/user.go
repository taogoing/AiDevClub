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

func (r *UserRepo) FindByIDs(ids []uint) ([]model.User, error) {
	var users []model.User
	if len(ids) == 0 {
		return users, nil
	}
	if err := r.db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepo) UpdateRole(id uint, role model.UserRole) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("role", role).Error
}

func (r *UserRepo) AllUserIDs() ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&model.User{}).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *UserRepo) Count() (int64, error) {
	var total int64
	if err := r.db.Model(&model.User{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
