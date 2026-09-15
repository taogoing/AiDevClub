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
	ErrResCommentNotFound  = platform.NewBizError(http.StatusNotFound, platform.CodeResCommentNotFound, "资源评论不存在")
	ErrResCommentBadParent = platform.NewBizError(http.StatusBadRequest, platform.CodeBizError, "父评论不合法")
	ErrResourceNotFound    = platform.NewBizError(http.StatusNotFound, platform.CodeBizError, "资源不存在或不可见")
	ErrResCommentForbidden = platform.NewBizError(http.StatusForbidden, platform.CodeForbidden, "无权限")
)

type ResourceCommentService struct {
	comments       *repo.ResourceCommentRepo
	skills         *repo.SkillRepo
	mcpServers     *repo.McpServerRepo
	inter          *repo.InteractionRepo
	users          *repo.UserRepo
	notifSvc       *NotificationService
	contentRanking *ContentRankingService
}

func NewResourceCommentService(comments *repo.ResourceCommentRepo, skills *repo.SkillRepo, mcpServers *repo.McpServerRepo, inter *repo.InteractionRepo, users *repo.UserRepo, notifSvc *NotificationService, contentRanking *ContentRankingService) *ResourceCommentService {
	return &ResourceCommentService{comments: comments, skills: skills, mcpServers: mcpServers, inter: inter, users: users, notifSvc: notifSvc, contentRanking: contentRanking}
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

func (s *ResourceCommentService) incrCommentsCount(tx *gorm.DB, resourceType string, resourceID uint, delta int) error {
	switch resourceType {
	case "skill":
		return s.skills.IncrCount(tx, resourceID, "comments_count", delta)
	case "mcp_server":
		return s.mcpServers.IncrCount(tx, resourceID, "comments_count", delta)
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
	replyToID := parentID
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
		ReplyToID:    replyToID,
		Content:      content,
	}
	notification := s.resourceCommentNotificationDraft(userID, resourceType, resourceID, replyToID, content)
	err = s.getDB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Create(tx, c); err != nil {
			return err
		}
		if err := s.incrCommentsCount(tx, resourceType, resourceID, 1); err != nil {
			return err
		}
		return s.notifSvc.EnqueueInTx(tx, notification)
	})
	if err == nil {
		if s.contentRanking != nil {
			if ct, ok := rankedResourceType(resourceType); ok {
				_ = s.contentRanking.AddScore(ctx, ct, resourceID, 3)
			}
		}
		s.notifSvc.DeliverAfterCommit(ctx, notification)
	}
	return c, err
}

func (s *ResourceCommentService) resourceCommentNotificationDraft(userID uint, resourceType string, resourceID uint, replyToID *uint, content string) *NotificationDraft {
	if replyToID != nil {
		rc, err := s.comments.FindByID(nil, *replyToID)
		if err == nil && rc.AuthorID != userID {
			return &NotificationDraft{UserID: rc.AuthorID, Type: model.NotifTypeReplyComment, Title: "新回复", Content: content, ResourceType: resourceType, ResourceID: resourceID, ActorID: userID}
		}
		return nil
	}
	authorID, err := s.checkResourcePublished(resourceType, resourceID)
	if err != nil {
		return nil
	}
	if authorID != userID {
		return &NotificationDraft{UserID: authorID, Type: model.NotifTypeCommentArticle, Title: "新评论", Content: content, ResourceType: resourceType, ResourceID: resourceID, ActorID: userID}
	}
	return nil
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
	err = s.getDB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Delete(tx, commentID); err != nil {
			return err
		}
		return s.incrCommentsCount(tx, c.ResourceType, c.ResourceID, -1)
	})
	if err == nil && s.contentRanking != nil {
		if ct, ok := rankedResourceType(c.ResourceType); ok {
			_ = s.contentRanking.AddScore(ctx, ct, c.ResourceID, -3)
		}
	}
	return err
}

func (s *ResourceCommentService) ToggleLike(ctx context.Context, userID, commentID uint) (bool, int, error) {
	c, err := s.comments.FindByID(nil, commentID)
	if err != nil {
		return false, 0, ErrResCommentNotFound
	}
	var liked bool
	var newCount int
	notification := &NotificationDraft{UserID: c.AuthorID, Type: model.NotifTypeLikeResourceComment, Title: "点赞", Content: "有人赞了你的评论", ResourceType: c.ResourceType, ResourceID: commentID, ActorID: userID}
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
		if liked {
			return s.notifSvc.EnqueueInTx(tx, notification)
		}
		return nil
	})
	if err == nil && liked {
		s.notifSvc.DeliverAfterCommit(ctx, notification)
	}
	return liked, newCount, err
}
