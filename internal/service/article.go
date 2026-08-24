package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var (
	ErrArticleNotFound  = platform.NewBizError(http.StatusNotFound, 40402, "文章不存在或不可见")
	ErrCommentNotFound  = platform.NewBizError(http.StatusNotFound, 40403, "评论不存在")
	ErrCategoryNotFound = platform.NewBizError(http.StatusNotFound, 40404, "分类不存在")
	ErrTagNotFound      = platform.NewBizError(http.StatusNotFound, 40405, "标签不存在")
	ErrForbidden        = platform.NewBizError(http.StatusForbidden, 40301, "无权限")
	ErrBadParam         = platform.NewBizError(http.StatusBadRequest, 40002, "参数不合法")
)

type ArticleService struct {
	articles *repo.ArticleRepo
	tags     *repo.TagRepo
	cats     *repo.CategoryRepo
	inter    *repo.InteractionRepo
	rdb      *redis.Client
	cfg      *platform.Config
}

func NewArticleService(articles *repo.ArticleRepo, tags *repo.TagRepo, cats *repo.CategoryRepo, inter *repo.InteractionRepo, rdb *redis.Client, cfg *platform.Config) *ArticleService {
	return &ArticleService{articles: articles, tags: tags, cats: cats, inter: inter, rdb: rdb, cfg: cfg}
}

func (s *ArticleService) ImageDir() string     { return s.cfg.ArticleImageDir }
func (s *ArticleService) MaxImageBytes() int64 { return s.cfg.MaxArticleImageBytes }

func (s *ArticleService) validateStatus(st model.ArticleStatus) error {
	switch st {
	case model.ArticleStatusDraft, model.ArticleStatusPublished:
		return nil
	}
	return ErrBadParam
}

func (s *ArticleService) ResolveTagSet(ctx context.Context, tx *gorm.DB, tagIDs []uint, tagNames []string) ([]uint, error) {
	set := map[uint]bool{}
	var out []uint
	for _, id := range tagIDs {
		t, err := s.tags.FindByID(ctx, id)
		if err != nil || !t.Enabled {
			return nil, ErrTagNotFound
		}
		if !set[id] {
			set[id] = true
			out = append(out, id)
		}
	}
	for _, name := range tagNames {
		if name == "" {
			continue
		}
		t, err := s.tags.FindByName(ctx, name)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t, err = s.tags.Create(ctx, name)
			if err != nil {
				if platform.IsDuplicateEntry(err) {
					t, err = s.tags.FindByName(ctx, name)
				} else {
					return nil, err
				}
			}
		} else if err != nil {
			return nil, err
		}
		if !t.Enabled {
			continue
		}
		if !set[t.ID] {
			set[t.ID] = true
			out = append(out, t.ID)
		}
	}
	return out, nil
}

func (s *ArticleService) Create(ctx context.Context, userID uint, in CreateArticleInput) (*model.Article, error) {
	if in.Title == "" || in.Content == "" {
		return nil, ErrBadParam
	}
	if len(in.Title) > 200 {
		return nil, ErrBadParam
	}
	if err := s.validateStatus(in.Status); err != nil {
		return nil, err
	}
	if _, err := s.cats.FindByID(ctx, in.CategoryID); err != nil {
		return nil, ErrCategoryNotFound
	}
	a := &model.Article{
		AuthorID:   userID,
		CategoryID: in.CategoryID,
		Title:      in.Title,
		Summary:    in.Summary,
		Content:    in.Content,
		Status:     in.Status,
	}
	if in.Status == model.ArticleStatusPublished {
		now := time.Now()
		a.PublishedAt = &now
	}

	err := s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.articles.Create(tx, a); err != nil {
			return err
		}
		tagIDs, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		if len(tagIDs) > 0 {
			if err := s.articles.SetArticleTags(tx, a.ID, tagIDs); err != nil {
				return err
			}
			for _, id := range tagIDs {
				if err := s.tags.IncrUsage(tx, id, 1); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}
