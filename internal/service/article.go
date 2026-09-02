package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var (
	ErrArticleNotFound = platform.NewBizError(http.StatusNotFound, platform.CodeArticleNotFound, "文章不存在或不可见")
	ErrCommentNotFound = platform.NewBizError(http.StatusNotFound, platform.CodeCommentNotFound, "评论不存在")
	ErrTagNotFound     = platform.NewBizError(http.StatusNotFound, platform.CodeTagNotFound, "标签不存在")
	ErrForbidden       = platform.NewBizError(http.StatusForbidden, platform.CodeForbidden, "无权限")
	ErrBadParam        = platform.NewBizError(http.StatusBadRequest, platform.CodeBizError, "参数不合法")
)

type ArticleService struct {
	articles   *repo.ArticleRepo
	tags       *repo.TagRepo
	inter      *repo.InteractionRepo
	cfg        *platform.Config
	notifSvc   *NotificationService
	rankingSvc *RankingService
}

func NewArticleService(articles *repo.ArticleRepo, tags *repo.TagRepo, inter *repo.InteractionRepo, cfg *platform.Config, notifSvc *NotificationService, rankingSvc *RankingService) *ArticleService {
	return &ArticleService{articles: articles, tags: tags, inter: inter, cfg: cfg, notifSvc: notifSvc, rankingSvc: rankingSvc}
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
		var tag model.Tag
		err := tx.WithContext(ctx).First(&tag, id).Error
		t := &tag
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
		var tag model.Tag
		err := tx.WithContext(ctx).Where("name = ?", name).First(&tag).Error
		t := &tag
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t = &model.Tag{Name: name, Enabled: true}
			err = tx.WithContext(ctx).Create(t).Error
			if err != nil {
				if platform.IsDuplicateEntry(err) {
					err = tx.WithContext(ctx).Where("name = ?", name).First(t).Error
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
	a := &model.Article{
		AuthorID: userID,
		Title:    in.Title,
		Summary:  in.Summary,
		Content:  in.Content,
		Status:   in.Status,
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
	return a, err
}

func (s *ArticleService) Update(ctx context.Context, userID, articleID uint, in CreateArticleInput) (*model.Article, error) {
	if in.Title == "" || in.Content == "" {
		return nil, ErrBadParam
	}
	if len(in.Title) > 200 {
		return nil, ErrBadParam
	}
	if err := s.validateStatus(in.Status); err != nil {
		return nil, err
	}
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil {
		return nil, ErrArticleNotFound
	}
	if a.AuthorID != userID {
		return nil, ErrForbidden
	}
	oldTags, err := s.articles.FindArticleTags(nil, articleID)
	if err != nil {
		return nil, err
	}

	a.Title = in.Title
	a.Summary = in.Summary
	a.Content = in.Content
	if a.Status != in.Status {
		a.Status = in.Status
		if in.Status == model.ArticleStatusPublished && a.PublishedAt == nil {
			now := time.Now()
			a.PublishedAt = &now
		}
	}

	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.articles.Update(tx, a); err != nil {
			return err
		}
		newTags, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		oldSet := map[uint]bool{}
		for _, id := range oldTags {
			oldSet[id] = true
		}
		newSet := map[uint]bool{}
		for _, id := range newTags {
			newSet[id] = true
		}
		for _, id := range oldTags {
			if !newSet[id] {
				if err := s.tags.IncrUsage(tx, id, -1); err != nil {
					return err
				}
			}
		}
		for _, id := range newTags {
			if !oldSet[id] {
				if err := s.tags.IncrUsage(tx, id, 1); err != nil {
					return err
				}
			}
		}
		return s.articles.SetArticleTags(tx, articleID, newTags)
	})
	return a, err
}

func (s *ArticleService) Delete(ctx context.Context, userID, articleID uint) error {
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil {
		return ErrArticleNotFound
	}
	if a.AuthorID != userID {
		return ErrForbidden
	}
	return s.articles.DB().Transaction(func(tx *gorm.DB) error {
		tagIDs, err := s.articles.FindArticleTags(tx, articleID)
		if err != nil {
			return err
		}
		if err := s.articles.Delete(tx, articleID); err != nil {
			return err
		}
		for _, id := range tagIDs {
			if err := s.tags.IncrUsage(tx, id, -1); err != nil {
				return err
			}
		}
		return s.articles.SetArticleTags(tx, articleID, nil)
	})
}

func (s *ArticleService) List(ctx context.Context, q ListQuery) (*ArticleListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = s.cfg.DefaultPageSize
	}
	if q.PageSize > s.cfg.MaxPageSize {
		q.PageSize = s.cfg.MaxPageSize
	}
	switch q.Sort {
	case "hot", "pinned":
	default:
		q.Sort = "latest"
	}

	if q.Sort == "hot" && q.TagID == nil && q.AuthorID == nil && q.Keyword == "" && s.rankingSvc != nil {
		articles, total, err := s.rankingSvc.ListArticleHot(ctx, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		return &ArticleListResult{
			List:     articles,
			Total:    total,
			Page:     q.Page,
			PageSize: q.PageSize,
		}, nil
	}

	rq := repo.ArticleQuery{
		Page: q.Page, PageSize: q.PageSize,
		TagID:   q.TagID,
		Keyword: q.Keyword, AuthorID: q.AuthorID, Sort: q.Sort,
	}

	list, total, err := s.articles.List(ctx, rq)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	tagMap, err := s.articles.TagsForArticles(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := &ArticleListResult{
		List:  make([]ArticleSummary, 0, len(list)),
		Total: total, Page: q.Page, PageSize: q.PageSize,
	}
	for _, a := range list {
		out.List = append(out.List, s.summaryOf(a, tagMap[a.ID]))
	}
	return out, nil
}

func (s *ArticleService) summaryOf(a model.Article, tags []model.Tag) ArticleSummary {
	sm := ArticleSummary{
		ID: a.ID, Title: a.Title, Summary: a.Summary,
		Tags:  []TagBrief{},
		Views: a.Views, LikesCount: a.LikesCount,
		FavoritesCount: a.FavoritesCount, CommentsCount: a.CommentsCount,
		Status: string(a.Status), PublishedAt: a.PublishedAt, Pinned: a.Pinned,
		Author: AuthorBrief{ID: a.AuthorID},
	}
	if a.Author != nil {
		sm.Author = AuthorBrief{ID: a.Author.ID, Nickname: a.Author.Nickname, AvatarURL: a.Author.AvatarURL}
	}
	for _, t := range tags {
		sm.Tags = append(sm.Tags, TagBrief{ID: t.ID, Name: t.Name})
	}
	return sm
}

func (s *ArticleService) ToggleLike(ctx context.Context, userID, articleID uint) (bool, int, error) {
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil || a.Status != model.ArticleStatusPublished {
		return false, 0, ErrArticleNotFound
	}
	var liked bool
	var newCount int
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.inter.ToggleArticleLike(tx, userID, articleID)
		if err != nil {
			return err
		}
		delta := 1
		if !liked {
			delta = -1
		}
		if err := s.articles.IncrCount(tx, articleID, "likes_count", delta); err != nil {
			return err
		}
		newCount = a.LikesCount + delta
		return nil
	})
	if err == nil {
		if s.rankingSvc != nil {
			go func() {
				updated, _ := s.articles.FindByID(nil, articleID)
				if updated != nil {
					_ = s.rankingSvc.UpdateArticleHotScore(context.Background(), updated)
				}
			}()
		}
		if liked {
			go func() {
				_ = s.notifSvc.Create(context.Background(), a.AuthorID, model.NotifTypeLikeArticle, "点赞", "有人赞了你的文章", "article", articleID, userID)
			}()
		}
	}
	return liked, newCount, err
}

func (s *ArticleService) ToggleFavorite(ctx context.Context, userID, articleID uint) (bool, int, error) {
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil || a.Status != model.ArticleStatusPublished {
		return false, 0, ErrArticleNotFound
	}
	var favorited bool
	var newCount int
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		favorited, err = s.inter.ToggleArticleFavorite(tx, userID, articleID)
		if err != nil {
			return err
		}
		delta := 1
		if !favorited {
			delta = -1
		}
		if err := s.articles.IncrCount(tx, articleID, "favorites_count", delta); err != nil {
			return err
		}
		newCount = a.FavoritesCount + delta
		return nil
	})
	if err == nil {
		if s.rankingSvc != nil {
			go func() {
				updated, _ := s.articles.FindByID(nil, articleID)
				if updated != nil {
					_ = s.rankingSvc.UpdateArticleHotScore(context.Background(), updated)
				}
			}()
		}
	}
	return favorited, newCount, err
}

func (s *ArticleService) Get(ctx context.Context, userID, articleID uint) (*ArticleDetail, error) {
	return s.detail(ctx, userID, articleID, true, true)
}

func (s *ArticleService) Read(ctx context.Context, userID, articleID uint) (*ArticleDetail, error) {
	return s.detail(ctx, userID, articleID, false, false)
}

func (s *ArticleService) canView(a *model.Article, userID uint) bool {
	if userID > 0 && a.AuthorID == userID {
		return true
	}
	return a.Status == model.ArticleStatusPublished && !a.Hidden
}

func (s *ArticleService) detail(ctx context.Context, userID, articleID uint, trackView, loadInteractions bool) (*ArticleDetail, error) {
	a, err := s.articles.FindByIDWithContext(ctx, articleID)
	if err != nil {
		return nil, ErrArticleNotFound
	}
	if !s.canView(a, userID) {
		return nil, ErrArticleNotFound
	}
	if trackView && a.Status == model.ArticleStatusPublished {
		_ = s.articles.IncrViews(ctx, articleID)
		a.Views++
	}
	tagMap, err := s.articles.TagsForArticles(ctx, []uint{articleID})
	if err != nil {
		return nil, err
	}
	sm := s.summaryOf(*a, tagMap[a.ID])
	d := &ArticleDetail{ArticleSummary: sm, Content: a.Content}
	if loadInteractions && userID > 0 {
		if d.Liked, err = s.inter.ArticleLiked(ctx, userID, articleID); err != nil {
			return nil, err
		}
		if d.Favorited, err = s.inter.ArticleFavorited(ctx, userID, articleID); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (s *ArticleService) ListOwned(ctx context.Context, userID uint, status string, page, pageSize int) (*ArticleListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = s.cfg.DefaultPageSize
	}
	if pageSize > s.cfg.MaxPageSize {
		pageSize = s.cfg.MaxPageSize
	}
	list, total, err := s.articles.ListOwned(ctx, userID, status, page, pageSize)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(list))
	for _, article := range list {
		ids = append(ids, article.ID)
	}
	tagMap, err := s.articles.TagsForArticles(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := &ArticleListResult{List: make([]ArticleSummary, 0, len(list)), Total: total, Page: page, PageSize: pageSize}
	for _, article := range list {
		out.List = append(out.List, s.summaryOf(article, tagMap[article.ID]))
	}
	return out, nil
}
