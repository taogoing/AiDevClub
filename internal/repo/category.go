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
	{Name: "AI/LLM", Slug: "ai-llm", SortOrder: 4},
	{Name: "DevOps", Slug: "devops", SortOrder: 5},
	{Name: "数据库", Slug: "database", SortOrder: 6},
	{Name: "移动端", Slug: "mobile", SortOrder: 7},
	{Name: "安全", Slug: "security", SortOrder: 8},
	{Name: "其他", Slug: "other", SortOrder: 9},
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
