package service

import (
	"context"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type TagService struct{ tags *repo.TagRepo }

func NewTagService(tags *repo.TagRepo) *TagService { return &TagService{tags: tags} }

func (s *TagService) List(ctx context.Context, keyword string, hot bool, limit int) ([]model.Tag, error) {
	if limit <= 0 {
		limit = 50
	}
	if hot {
		return s.tags.ListHot(ctx, limit)
	}
	return s.tags.List(ctx, keyword, limit)
}
