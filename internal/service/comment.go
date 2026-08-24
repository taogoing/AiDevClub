package service

import (
	"context"
	"net/http"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var ErrBadParent = platform.NewBizError(http.StatusBadRequest, platform.CodeBizError, "父评论不合法")

type CommentService struct {
	comments *repo.CommentRepo
	articles *repo.ArticleRepo
	inter    *repo.InteractionRepo
	users    *repo.UserRepo
}

func NewCommentService(comments *repo.CommentRepo, articles *repo.ArticleRepo, inter *repo.InteractionRepo, users *repo.UserRepo) *CommentService {
	return &CommentService{comments: comments, articles: articles, inter: inter, users: users}
}

func (s *CommentService) Create(ctx context.Context, userID, articleID uint, content string, parentID *uint) (*model.Comment, error) {
	if content == "" {
		return nil, ErrBadParam
	}
	if len(content) > 2000 {
		return nil, ErrBadParam
	}
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil || a.Status != model.ArticleStatusPublished {
		return nil, ErrArticleNotFound
	}
	if parentID != nil {
		p, err := s.comments.FindByID(nil, *parentID)
		if err != nil || p.ArticleID != articleID {
			return nil, ErrBadParent
		}
		if p.ParentID != nil {
			parentID = p.ParentID
		}
	}
	c := &model.Comment{ArticleID: articleID, AuthorID: userID, ParentID: parentID, Content: content}
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Create(tx, c); err != nil {
			return err
		}
		return s.articles.IncrCount(tx, articleID, "comments_count", 1)
	})
	return c, err
}

func (s *CommentService) List(ctx context.Context, articleID uint) ([]CommentItem, error) {
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil || a.Status != model.ArticleStatusPublished {
		return nil, ErrArticleNotFound
	}
	comments, err := s.comments.ListByArticle(nil, articleID)
	if err != nil {
		return nil, err
	}
	items := assembleComments(comments)
	ids := map[uint]bool{}
	var collect func(items []CommentItem)
	collect = func(items []CommentItem) {
		for i := range items {
			ids[items[i].AuthorID] = true
			collect(items[i].Replies)
		}
	}
	collect(items)
	keys := make([]uint, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	users, err := s.users.FindByIDs(keys)
	if err != nil {
		return nil, err
	}
	byID := map[uint]model.User{}
	for _, u := range users {
		byID[u.ID] = u
	}
	var fill func(items []CommentItem)
	fill = func(items []CommentItem) {
		for i := range items {
			if u, ok := byID[items[i].AuthorID]; ok {
				items[i].Author = AuthorBrief{ID: u.ID, Nickname: u.Nickname, AvatarURL: u.AvatarURL}
			}
			fill(items[i].Replies)
		}
	}
	fill(items)
	return items, nil
}

func assembleComments(comments []model.Comment) []CommentItem {
	roots := map[uint]*CommentItem{}
	var order []*CommentItem
	for i := range comments {
		c := &comments[i]
		if c.ParentID == nil {
			it := newCommentItem(c)
			roots[c.ID] = &it
			order = append(order, &it)
		}
	}
	var orphans []CommentItem
	for i := range comments {
		c := &comments[i]
		if c.ParentID != nil {
			it := newCommentItem(c)
			if root, ok := roots[*c.ParentID]; ok {
				root.Replies = append(root.Replies, it)
			} else {
				orphans = append(orphans, it)
			}
		}
	}
	result := make([]CommentItem, 0, len(order)+len(orphans))
	for _, p := range order {
		result = append(result, *p)
	}
	result = append(result, orphans...)
	return result
}

func newCommentItem(c *model.Comment) CommentItem {
	return CommentItem{
		ID: c.ID, ArticleID: c.ArticleID, AuthorID: c.AuthorID,
		Content: c.Content, LikesCount: c.LikesCount, CreatedAt: c.CreatedAt,
		Author: AuthorBrief{ID: c.AuthorID}, Replies: []CommentItem{},
	}
}

func (s *CommentService) Delete(ctx context.Context, userID, commentID uint) error {
	c, err := s.comments.FindByID(nil, commentID)
	if err != nil {
		return ErrCommentNotFound
	}
	a, err := s.articles.FindByID(nil, c.ArticleID)
	if err != nil {
		return ErrArticleNotFound
	}
	if c.AuthorID != userID && a.AuthorID != userID {
		return ErrForbidden
	}
	return s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Delete(tx, commentID); err != nil {
			return err
		}
		return s.articles.IncrCount(tx, c.ArticleID, "comments_count", -1)
	})
}

func (s *CommentService) ToggleLike(ctx context.Context, userID, commentID uint) (bool, int, error) {
	c, err := s.comments.FindByID(nil, commentID)
	if err != nil {
		return false, 0, ErrCommentNotFound
	}
	var liked bool
	var newCount int
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.inter.ToggleCommentLike(tx, userID, commentID)
		if err != nil {
			return err
		}
		delta := 1
		if !liked {
			delta = -1
		}
		if err := s.comments.IncrLikes(tx, commentID, delta); err != nil {
			return err
		}
		newCount = c.LikesCount + delta
		return nil
	})
	return liked, newCount, err
}
