package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

var defaultCategories = []model.Category{
	{Name: "Go", Slug: "go", SortOrder: 1},
	{Name: "后端", Slug: "backend", SortOrder: 2},
	{Name: "前端", Slug: "frontend", SortOrder: 3},
	{Name: "AI", Slug: "ai", SortOrder: 4},
	{Name: "Agent", Slug: "agent", SortOrder: 5},
	{Name: "数据库", Slug: "database", SortOrder: 6},
	{Name: "DevOps", Slug: "devops", SortOrder: 7},
}

type CategoryRepo struct{ db *gorm.DB }

func NewCategoryRepo(db *gorm.DB) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) List(ctx context.Context) ([]model.Category, error) {
	var list []model.Category
	err := r.db.WithContext(ctx).Order("sort_order asc, id asc").Find(&list).Error
	return list, err
}

func (r *CategoryRepo) FindByID(ctx context.Context, id uint) (*model.Category, error) {
	var c model.Category
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepo) Seed(ctx context.Context) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Category{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&defaultCategories).Error
}

// SeedForce 强制重新初始化分类（清空现有数据后插入默认分类）
func (r *CategoryRepo) SeedForce(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清空分类表
		if err := tx.Unscoped().Delete(&model.Category{}, "1 = 1").Error; err != nil {
			return err
		}
		// 重新插入默认分类
		return tx.Create(&defaultCategories).Error
	})
}
