package service

import (
	"context"
	"strings"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type CategoryService struct{ cats *repo.CategoryRepo }

func NewCategoryService(cats *repo.CategoryRepo) *CategoryService {
	return &CategoryService{cats: cats}
}

func (s *CategoryService) List(ctx context.Context) ([]model.Category, error) {
	return s.cats.List(ctx)
}

func (s *CategoryService) ListForMCP(ctx context.Context, keyword string, limit int) ([]model.Category, error) {
	if limit <= 0 {
		limit = 50
	}
	list, err := s.cats.List(ctx)
	if err != nil {
		return nil, err
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	filtered := make([]model.Category, 0, len(list))
	for _, category := range list {
		if keyword != "" && !strings.Contains(strings.ToLower(category.Name), keyword) && !strings.Contains(strings.ToLower(category.Slug), keyword) {
			continue
		}
		filtered = append(filtered, category)
		if len(filtered) == limit {
			break
		}
	}
	return filtered, nil
}
