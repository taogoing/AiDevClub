package service

import (
	"context"
	"encoding/json"
	"time"

	"aidevclub/internal/repo"
	amqp "github.com/rabbitmq/amqp091-go"
)

const documentIndexExchange = "aidevclub.document-index"
const documentIndexQueue = "aidevclub.document-index"

type DocumentIndexMQ struct {
	url    string
	outbox *repo.DocumentIndexOutboxRepo
	ai     *AIAssistantService
	cancel context.CancelFunc
}

func NewDocumentIndexMQ(url string, outbox *repo.DocumentIndexOutboxRepo, ai *AIAssistantService) *DocumentIndexMQ {
	return &DocumentIndexMQ{url: url, outbox: outbox, ai: ai}
}
func (m *DocumentIndexMQ) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	go m.publish(ctx)
	go m.consume(ctx)
}
func (m *DocumentIndexMQ) Close() {
	if m.cancel != nil {
		m.cancel()
	}
}
func declareDocumentIndex(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(documentIndexExchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(documentIndexQueue, true, false, false, false, nil); err != nil {
		return err
	}
	return ch.QueueBind(documentIndexQueue, "index", documentIndexExchange, false, nil)
}
func (m *DocumentIndexMQ) publish(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.publishBatch(ctx)
		}
	}
}
func (m *DocumentIndexMQ) publishBatch(ctx context.Context) {
	conn, err := amqp.Dial(m.url)
	if err != nil {
		return
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return
	}
	defer ch.Close()
	if declareDocumentIndex(ch) != nil {
		return
	}
	events, err := m.outbox.Pending(ctx, 100)
	if err != nil {
		return
	}
	for _, e := range events {
		if err := ch.PublishWithContext(ctx, documentIndexExchange, "index", false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: e.EventID, Body: e.Payload}); err == nil {
			_ = m.outbox.MarkPublished(ctx, e.ID)
		}
	}
}
func (m *DocumentIndexMQ) consume(ctx context.Context) {
	for ctx.Err() == nil {
		conn, err := amqp.Dial(m.url)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		ch, err := conn.Channel()
		if err != nil {
			conn.Close()
			continue
		}
		if declareDocumentIndex(ch) != nil {
			ch.Close()
			conn.Close()
			continue
		}
		ds, err := ch.Consume(documentIndexQueue, "", false, false, false, false, nil)
		if err != nil {
			ch.Close()
			conn.Close()
			continue
		}
		for d := range ds {
			if m.process(ctx, d.Body) == nil {
				_ = d.Ack(false)
			} else {
				_ = d.Nack(false, true)
			}
		}
		ch.Close()
		conn.Close()
	}
}
func (m *DocumentIndexMQ) process(ctx context.Context, raw []byte) error {
	var e struct {
		Operation string `json:"Operation"`
		ArticleID uint   `json:"ArticleID"`
		Title     string `json:"Title"`
		Content   string `json:"Content"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return err
	}
	if e.Operation == "delete" {
		return m.ai.DeleteArticle(ctx, e.ArticleID)
	}
	return m.ai.IndexArticle(ctx, e.ArticleID, e.Title, e.Content)
}
