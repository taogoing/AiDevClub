package service

import (
	"context"
	"errors"

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

func (s *TagService) AdminCreate(ctx context.Context, name, description string) (*model.Tag, error) {
	if name == "" {
		return nil, errors.New("标签名称不能为空")
	}
	return s.tags.AdminCreate(ctx, name, description)
}

func (s *TagService) AdminUpdate(ctx context.Context, id uint, name, description string) error {
	if name == "" {
		return errors.New("标签名称不能为空")
	}
	return s.tags.AdminUpdate(ctx, id, name, description)
}

func (s *TagService) Enable(ctx context.Context, id uint) error {
	return s.tags.Enable(ctx, id)
}

func (s *TagService) Disable(ctx context.Context, id uint) error {
	return s.tags.Disable(ctx, id)
}

func (s *TagService) AdminList(ctx context.Context, keyword, status string, page, pageSize int) ([]model.Tag, int64, error) {
	return s.tags.AdminList(ctx, keyword, status, page, pageSize)
}
