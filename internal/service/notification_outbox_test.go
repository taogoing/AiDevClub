package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
	amqp "github.com/rabbitmq/amqp091-go"
)

type fakeOutboxPublisher struct {
	calls int
	err   error
}

func (p *fakeOutboxPublisher) Publish(context.Context, *model.NotificationOutboxEvent) error {
	p.calls++
	return p.err
}

func TestOutboxPublisherRetriesThenMarksPublished(t *testing.T) {
	db := testutil.NewTestDB(t)
	outbox := repo.NewNotificationOutboxRepo(db)
	payload, err := json.Marshal(notificationEventPayload{EventID: "retry-event", Data: NotificationDraft{UserID: 3, Type: model.NotifTypeLikeArticle, Title: "点赞"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := outbox.Create(db, &model.NotificationOutboxEvent{EventID: "retry-event", EventType: notificationEventType, Payload: payload, Status: model.NotificationOutboxPending}); err != nil {
		t.Fatal(err)
	}
	publisher := &fakeOutboxPublisher{err: errors.New("broker unavailable")}
	worker := NewNotificationOutboxWorker(outbox, publisher, time.Millisecond, time.Minute)
	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("publish failure was not returned")
	}
	var event model.NotificationOutboxEvent
	if err := db.First(&event, "event_id = ?", "retry-event").Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != model.NotificationOutboxPending || event.RetryCount != 1 || event.LastError == "" {
		t.Fatalf("event after failed publish = %+v", event)
	}
	if event.NextRetryAt == nil || !event.NextRetryAt.After(time.Now()) {
		t.Fatalf("retry is not scheduled in the future: %v", event.NextRetryAt)
	}
	if err := db.Model(&event).Update("next_retry_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	publisher.err = nil
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&event, "event_id = ?", "retry-event").Error; err != nil {
		t.Fatal(err)
	}
	if publisher.calls != 2 || event.Status != model.NotificationOutboxPublished || event.PublishedAt == nil {
		t.Fatalf("publisher calls=%d, event=%+v", publisher.calls, event)
	}
}

func TestOutboxPublisherBatchDrainsAvailableEvents(t *testing.T) {
	db := testutil.NewTestDB(t)
	outbox := repo.NewNotificationOutboxRepo(db)
	for _, eventID := range []string{"batch-one", "batch-two", "batch-three"} {
		payload, err := json.Marshal(notificationEventPayload{EventID: eventID, Data: NotificationDraft{UserID: 3, Type: model.NotifTypeLikeArticle, Title: "点赞"}})
		if err != nil {
			t.Fatal(err)
		}
		if err := outbox.Create(db, &model.NotificationOutboxEvent{EventID: eventID, EventType: notificationEventType, Payload: payload, Status: model.NotificationOutboxPending}); err != nil {
			t.Fatal(err)
		}
	}
	publisher := &fakeOutboxPublisher{}
	worker := NewNotificationOutboxWorker(outbox, publisher, time.Millisecond, time.Minute)
	processed, err := worker.RunBatch(context.Background(), 10)
	if err != nil || processed != 3 || publisher.calls != 3 {
		t.Fatalf("RunBatch = (%d, %v), publisher calls=%d; want (3, nil), 3", processed, err, publisher.calls)
	}
	var pending int64
	if err := db.Model(&model.NotificationOutboxEvent{}).Where("status = ?", model.NotificationOutboxPending).Count(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Fatalf("pending events = %d, want 0", pending)
	}
}

func TestNotificationConsumerAcksOnlyAfterSuccessfulProcessing(t *testing.T) {
	calls := []string{}
	ack := func() error { calls = append(calls, "ack"); return nil }
	process := func(context.Context, []byte) error { calls = append(calls, "process"); return nil }
	if err := processBeforeAck(context.Background(), nil, process, ack); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "process" || calls[1] != "ack" {
		t.Fatalf("order = %v, want process then ack", calls)
	}
	calls = nil
	process = func(context.Context, []byte) error {
		calls = append(calls, "process")
		return errors.New("database unavailable")
	}
	if err := processBeforeAck(context.Background(), nil, process, ack); err == nil {
		t.Fatal("consumer processing failure was not returned")
	}
	if len(calls) != 1 || calls[0] != "process" {
		t.Fatalf("failed processing was ACKed: %v", calls)
	}
}

func TestAwaitPublishConfirmationRejectsUnroutableMandatoryMessage(t *testing.T) {
	confirms := make(chan amqp.Confirmation, 1)
	returns := make(chan amqp.Return, 1)
	returns <- amqp.Return{MessageId: "event-1", ReplyCode: 312, ReplyText: "NO_ROUTE"}
	confirms <- amqp.Confirmation{Ack: true}
	if err := awaitPublishConfirmation(context.Background(), confirms, returns, "event-1"); err == nil {
		t.Fatal("mandatory unroutable publish was accepted from its ACK alone")
	}
}

func TestAwaitPublishConfirmationAcceptsRoutedMessage(t *testing.T) {
	confirms := make(chan amqp.Confirmation, 1)
	returns := make(chan amqp.Return, 1)
	confirms <- amqp.Confirmation{Ack: true}
	if err := awaitPublishConfirmation(context.Background(), confirms, returns, "event-2"); err != nil {
		t.Fatalf("routed publish confirmation: %v", err)
	}
}
