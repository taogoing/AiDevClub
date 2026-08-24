package service

import (
	"context"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type CategoryService struct{ cats *repo.CategoryRepo }

func NewCategoryService(cats *repo.CategoryRepo) *CategoryService { return &CategoryService{cats: cats} }

func (s *CategoryService) List(ctx context.Context) ([]model.Category, error) {
	return s.cats.List(ctx)
}
