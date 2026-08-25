package repo

import (
	"context"

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
	return r.findByID(r.db, id)
}

func (r *UserRepo) FindByIDWithContext(ctx context.Context, id uint) (*model.User, error) {
	return r.findByID(r.db.WithContext(ctx), id)
}

func (r *UserRepo) findByID(db *gorm.DB, id uint) (*model.User, error) {
	var u model.User
	if err := db.First(&u, id).Error; err != nil {
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

type UserQuery struct {
	Keyword  string
	Role     model.UserRole
	Page     int
	PageSize int
}

func (r *UserRepo) ListUsers(ctx context.Context, q UserQuery) ([]model.User, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.User{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		d = d.Where("email LIKE ? OR nickname LIKE ?", like, like)
	}
	if q.Role != "" {
		d = d.Where("role = ?", q.Role)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.User
	if err := d.Order("id asc").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *UserRepo) FindPublicByIDs(ctx context.Context, ids []uint) ([]model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []model.User
	if err := r.db.WithContext(ctx).Select("id, nickname, avatar_url").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
