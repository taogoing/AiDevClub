package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"gorm.io/gorm"
)

type NotificationDraft struct {
	UserID       uint            `json:"user_id"`
	Type         model.NotifType `json:"type"`
	Title        string          `json:"title"`
	Content      string          `json:"content"`
	ResourceType string          `json:"resource_type"`
	ResourceID   uint            `json:"resource_id"`
	ActorID      uint            `json:"actor_id"`
}

type notificationEventPayload struct {
	EventID string            `json:"event_id"`
	Data    NotificationDraft `json:"notification"`
}

type NotificationService struct {
	repo   *repo.NotificationRepo
	users  *repo.UserRepo
	mode   string
	outbox *repo.NotificationOutboxRepo
}

func NewNotificationService(r *repo.NotificationRepo, users *repo.UserRepo) *NotificationService {
	return &NotificationService{repo: r, users: users, mode: "sync"}
}

func (s *NotificationService) ConfigureMode(mode string, outbox *repo.NotificationOutboxRepo) {
	s.mode = mode
	s.outbox = outbox
}

func (s *NotificationService) IsOutboxMode() bool { return s.mode == "outbox" }

func newEventID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *NotificationService) EnqueueInTx(tx *gorm.DB, draft *NotificationDraft) error {
	if !s.IsOutboxMode() || draft == nil || draft.UserID == draft.ActorID {
		return nil
	}
	if s.outbox == nil {
		return fmt.Errorf("notification outbox repository is not configured")
	}
	eventID, err := newEventID()
	if err != nil {
		return fmt.Errorf("generate notification event id: %w", err)
	}
	payload, err := json.Marshal(notificationEventPayload{EventID: eventID, Data: *draft})
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}
	return s.outbox.Create(tx, &model.NotificationOutboxEvent{
		EventID: eventID, EventType: notificationEventType, Payload: payload,
		Status: model.NotificationOutboxPending,
	})
}

func (s *NotificationService) DeliverAfterCommit(ctx context.Context, draft *NotificationDraft) {
	if s.IsOutboxMode() || draft == nil {
		return
	}
	s.CreateBestEffort(ctx, draft.UserID, draft.Type, draft.Title, draft.Content, draft.ResourceType, draft.ResourceID, draft.ActorID)
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

// CreateBestEffort synchronously waits for notification persistence after the
// business transaction has committed. Notification failure is logged but must
// not turn a successful interaction into an HTTP failure.
func (s *NotificationService) CreateBestEffort(ctx context.Context, userID uint, notifType model.NotifType, title, content, resourceType string, resourceID, actorID uint) {
	if err := s.Create(ctx, userID, notifType, title, content, resourceType, resourceID, actorID); err != nil {
		slog.ErrorContext(ctx, "synchronous wait for notification failed; notification is best-effort after business success",
			"error", err, "user_id", userID, "type", notifType, "resource_type", resourceType, "resource_id", resourceID)
	}
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
