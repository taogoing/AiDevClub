package service

import (
	"context"
	"net/http"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var (
	ErrResCommentNotFound = platform.NewBizError(http.StatusNotFound, platform.CodeResCommentNotFound, "资源评论不存在")
	ErrResCommentBadParent = platform.NewBizError(http.StatusBadRequest, platform.CodeBizError, "父评论不合法")
	ErrResourceNotFound    = platform.NewBizError(http.StatusNotFound, platform.CodeBizError, "资源不存在或不可见")
	ErrResCommentForbidden = platform.NewBizError(http.StatusForbidden, platform.CodeForbidden, "无权限")
)

type ResourceCommentService struct {
	comments  *repo.ResourceCommentRepo
	skills    *repo.SkillRepo
	mcpServers *repo.McpServerRepo
	inter     *repo.InteractionRepo
	users     *repo.UserRepo
}

func NewResourceCommentService(comments *repo.ResourceCommentRepo, skills *repo.SkillRepo, mcpServers *repo.McpServerRepo, inter *repo.InteractionRepo, users *repo.UserRepo) *ResourceCommentService {
	return &ResourceCommentService{comments: comments, skills: skills, mcpServers: mcpServers, inter: inter, users: users}
}

func (s *ResourceCommentService) getDB() *gorm.DB {
	return s.comments.DB()
}

func (s *ResourceCommentService) checkResourcePublished(resourceType string, resourceID uint) (uint, error) {
	switch resourceType {
	case "skill":
		sk, err := s.skills.FindByID(nil, resourceID)
		if err != nil || sk.Status != model.ResourceStatusPublished {
			return 0, ErrResourceNotFound
		}
		return sk.AuthorID, nil
	case "mcp_server":
		sv, err := s.mcpServers.FindByID(nil, resourceID)
		if err != nil || sv.Status != model.ResourceStatusPublished {
			return 0, ErrResourceNotFound
		}
		return sv.AuthorID, nil
	default:
		return 0, ErrResourceNotFound
	}
}

func (s *ResourceCommentService) incrCommentsCount(resourceType string, resourceID uint, delta int) error {
	switch resourceType {
	case "skill":
		return s.skills.IncrCount(nil, resourceID, "comments_count", delta)
	case "mcp_server":
		return s.mcpServers.IncrCount(nil, resourceID, "comments_count", delta)
	default:
		return nil
	}
}

func (s *ResourceCommentService) Create(ctx context.Context, userID uint, resourceType string, resourceID uint, content string, parentID *uint) (*model.ResourceComment, error) {
	if content == "" || len(content) > 2000 {
		return nil, ErrBadParam
	}
	_, err := s.checkResourcePublished(resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	if parentID != nil {
		p, err := s.comments.FindByID(nil, *parentID)
		if err != nil || p.ResourceType != resourceType || p.ResourceID != resourceID {
			return nil, ErrResCommentBadParent
		}
		if p.ParentID != nil {
			parentID = p.ParentID
		}
	}
	c := &model.ResourceComment{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		AuthorID:     userID,
		ParentID:     parentID,
		Content:      content,
	}
	err = s.getDB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Create(tx, c); err != nil {
			return err
		}
		return s.incrCommentsCount(resourceType, resourceID, 1)
	})
	return c, err
}

func (s *ResourceCommentService) List(ctx context.Context, resourceType string, resourceID uint) ([]ResourceCommentItem, error) {
	if _, err := s.checkResourcePublished(resourceType, resourceID); err != nil {
		return nil, err
	}
	comments, err := s.comments.ListByResource(ctx, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	items := assembleResComments(comments)
	ids := map[uint]bool{}
	var collect func(items []ResourceCommentItem)
	collect = func(items []ResourceCommentItem) {
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
	var fill func(items []ResourceCommentItem)
	fill = func(items []ResourceCommentItem) {
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

func assembleResComments(comments []model.ResourceComment) []ResourceCommentItem {
	roots := map[uint]*ResourceCommentItem{}
	var order []*ResourceCommentItem
	for i := range comments {
		c := &comments[i]
		if c.ParentID == nil {
			it := newResCommentItem(c)
			roots[c.ID] = &it
			order = append(order, &it)
		}
	}
	var orphans []ResourceCommentItem
	for i := range comments {
		c := &comments[i]
		if c.ParentID != nil {
			it := newResCommentItem(c)
			if root, ok := roots[*c.ParentID]; ok {
				root.Replies = append(root.Replies, it)
			} else {
				orphans = append(orphans, it)
			}
		}
	}
	result := make([]ResourceCommentItem, 0, len(order)+len(orphans))
	for _, p := range order {
		result = append(result, *p)
	}
	result = append(result, orphans...)
	return result
}

func newResCommentItem(c *model.ResourceComment) ResourceCommentItem {
	return ResourceCommentItem{
		ID: c.ID, ResourceID: c.ResourceID, AuthorID: c.AuthorID,
		Content: c.Content, LikesCount: c.LikesCount, CreatedAt: c.CreatedAt,
		Author: AuthorBrief{ID: c.AuthorID}, Replies: []ResourceCommentItem{},
	}
}

func (s *ResourceCommentService) Delete(ctx context.Context, userID, commentID uint) error {
	c, err := s.comments.FindByID(nil, commentID)
	if err != nil {
		return ErrResCommentNotFound
	}
	resourceAuthorID, err := s.checkResourcePublished(c.ResourceType, c.ResourceID)
	if err != nil {
		return err
	}
	if c.AuthorID != userID && resourceAuthorID != userID {
		return ErrResCommentForbidden
	}
	return s.getDB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Delete(tx, commentID); err != nil {
			return err
		}
		return s.incrCommentsCount(c.ResourceType, c.ResourceID, -1)
	})
}

func (s *ResourceCommentService) ToggleLike(ctx context.Context, userID, commentID uint) (bool, int, error) {
	c, err := s.comments.FindByID(nil, commentID)
	if err != nil {
		return false, 0, ErrResCommentNotFound
	}
	var liked bool
	var newCount int
	err = s.getDB().Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.inter.ToggleResourceCommentLike(tx, userID, commentID)
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
