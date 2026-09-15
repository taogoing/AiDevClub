package repo

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestNotificationOutboxRowRollsBackWithBusinessTransaction(t *testing.T) {
	db := testutil.NewTestDB(t)
	outbox := NewNotificationOutboxRepo(db)
	payload, _ := json.Marshal(map[string]any{"user_id": uint(2), "type": "like_article"})
	errRollback := errors.New("force rollback")
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.ArticleLike{ArticleID: 1, UserID: 2}).Error; err != nil {
			return err
		}
		if err := outbox.Create(tx, &model.NotificationOutboxEvent{
			EventID: "test-atomic-event", EventType: "notification.created", Payload: payload,
			Status: model.NotificationOutboxPending,
		}); err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("transaction error = %v", err)
	}
	var likeCount, eventCount int64
	if err := db.Model(&model.ArticleLike{}).Count(&likeCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.NotificationOutboxEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatal(err)
	}
	if likeCount != 0 || eventCount != 0 {
		t.Fatalf("rollback left like=%d outbox=%d", likeCount, eventCount)
	}
}

func TestNotificationOutboxClaimUsesLeaseAndReclaimsAfterExpiry(t *testing.T) {
	db := testutil.NewTestDB(t)
	outbox := NewNotificationOutboxRepo(db)
	payload, _ := json.Marshal(map[string]any{"event_id": "lease-event"})
	if err := outbox.Create(db, &model.NotificationOutboxEvent{
		EventID: "lease-event", EventType: "notification.created", Payload: payload,
		Status: model.NotificationOutboxPending,
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	first, err := outbox.ClaimNext(context.Background(), now, time.Minute, "lease-one")
	if err != nil || first == nil {
		t.Fatalf("first claim = %+v, %v", first, err)
	}
	second, err := outbox.ClaimNext(context.Background(), now.Add(time.Second), time.Minute, "lease-two")
	if err != nil || second != nil {
		t.Fatalf("live lease was claimed twice: event=%+v err=%v", second, err)
	}
	reclaimed, err := outbox.ClaimNext(context.Background(), now.Add(2*time.Minute), time.Minute, "lease-two")
	if err != nil || reclaimed == nil || reclaimed.ID != first.ID || reclaimed.LeaseToken != "lease-two" {
		t.Fatalf("expired lease was not reclaimed: event=%+v err=%v", reclaimed, err)
	}
}

func TestNotificationCreateFromOutboxEventIsIdempotent(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewNotificationRepo(db)
	eventID := "event-idempotent"
	notification := &model.Notification{
		EventID: &eventID, UserID: 8, Type: model.NotifTypeLikeArticle,
		Title: "点赞", Content: "有人赞了你的文章", ResourceType: "article", ResourceID: 9, ActorID: 10,
	}
	if err := repo.CreateFromOutboxEvent(context.Background(), notification); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateFromOutboxEvent(context.Background(), notification); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.Notification{}).Where("event_id = ?", eventID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("notifications with event id = %d, want 1", count)
	}
}
