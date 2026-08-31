package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type TagService struct {
	tags *repo.TagRepo
	rdb  *redis.Client
}

func NewTagService(tags *repo.TagRepo, rdb *redis.Client) *TagService {
	return &TagService{tags: tags, rdb: rdb}
}

func (s *TagService) List(ctx context.Context, prefix string, hot bool, limit int) ([]model.Tag, error) {
	if limit <= 0 {
		limit = 50
	}

	if hot && prefix == "" && s.rdb != nil {
		key := fmt.Sprintf("hot:tags:%d", limit)
		if v, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var tags []model.Tag
			if json.Unmarshal(v, &tags) == nil {
				return tags, nil
			}
		}

		tags, err := s.tags.ListHot(ctx, limit)
		if err != nil {
			return nil, err
		}

		if b, err := json.Marshal(tags); err == nil {
			_ = s.rdb.Set(ctx, key, b, 300*time.Second).Err()
		}

		return tags, nil
	}

	if hot {
		return s.tags.ListHot(ctx, limit)
	}
	return s.tags.List(ctx, prefix, limit)
}

func (s *TagService) ListForMCP(ctx context.Context, keyword string, limit int) ([]model.Tag, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.tags.List(ctx, strings.TrimSpace(keyword), limit)
}

func (s *TagService) AdminCreate(ctx context.Context, name, description string) (*model.Tag, error) {
	if name == "" {
		return nil, errors.New("标签名称不能为空")
	}
	return s.tags.AdminCreate(ctx, name, description)
}

func (s *TagService) AdminUpdate(ctx context.Context, id uint, updates map[string]interface{}) error {
	if name, ok := updates["name"]; ok {
		if name == "" {
			return errors.New("标签名称不能为空")
		}
	}
	return s.tags.AdminPatch(ctx, id, updates)
}

func (s *TagService) AdminDelete(ctx context.Context, id uint) error {
	count, err := s.tags.CountTagUsage(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该标签正在被 %d 个内容引用，无法删除", count)
	}
	return s.tags.AdminDelete(ctx, id)
}

func (s *TagService) AdminList(ctx context.Context, keyword, status string, page, pageSize int) ([]model.Tag, int64, error) {
	return s.tags.AdminList(ctx, keyword, status, page, pageSize)
}
