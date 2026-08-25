package service

import (
	"context"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type NotificationService struct {
	repo  *repo.NotificationRepo
	users *repo.UserRepo
}

func NewNotificationService(r *repo.NotificationRepo, users *repo.UserRepo) *NotificationService {
	return &NotificationService{repo: r, users: users}
}

func (s *NotificationService) Create(ctx context.Context, userID uint, notifType model.NotifType, title, content, resourceType string, resourceID, actorID uint) error {
	if userID == actorID {
		return nil
	}
	n := &model.Notification{
		UserID:       userID,
		Type:         notifType,
		Title:        title,
		Content:      content,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ActorID:      actorID,
	}
	return s.repo.Create(n)
}

func (s *NotificationService) CreateBatchForAllUsers(ctx context.Context, notifType model.NotifType, title, content string, actorID uint) error {
	ids, err := s.users.AllUserIDs()
	if err != nil {
		return err
	}
	var batch []*model.Notification
	for _, uid := range ids {
		if uid == actorID {
			continue
		}
		batch = append(batch, &model.Notification{
			UserID:  uid,
			Type:    notifType,
			Title:   title,
			Content: content,
			ActorID: actorID,
		})
	}
	return s.repo.CreateBatch(batch)
}

func (s *NotificationService) List(ctx context.Context, userID uint, notifType string, unreadOnly bool, page, pageSize int) (*NotificationListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	list, total, err := s.repo.List(ctx, userID, notifType, unreadOnly, page, pageSize)
	if err != nil {
		return nil, err
	}
	actorIDs := make([]uint, 0, len(list))
	for _, n := range list {
		if n.ActorID > 0 {
			actorIDs = append(actorIDs, n.ActorID)
		}
	}
	actorMap := map[uint]AuthorBrief{}
	if len(actorIDs) > 0 {
		users, err := s.users.FindByIDs(actorIDs)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			actorMap[u.ID] = AuthorBrief{ID: u.ID, Nickname: u.Nickname, AvatarURL: u.AvatarURL}
		}
	}
	out := &NotificationListResult{
		List:     make([]NotificationItem, 0, len(list)),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
	for _, n := range list {
		item := NotificationItem{
			ID:           n.ID,
			Type:         string(n.Type),
			Title:        n.Title,
			Content:      n.Content,
			ResourceType: n.ResourceType,
			ResourceID:   n.ResourceID,
			IsRead:       n.IsRead,
			CreatedAt:    n.CreatedAt,
		}
		if a, ok := actorMap[n.ActorID]; ok {
			item.Actor = a
		}
		out.List = append(out.List, item)
	}
	return out, nil
}

func (s *NotificationService) UnreadCount(ctx context.Context, userID uint) (int64, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, userID, notifID uint) error {
	return s.repo.MarkRead(ctx, userID, notifID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uint) error {
	return s.repo.MarkAllRead(ctx, userID)
}
