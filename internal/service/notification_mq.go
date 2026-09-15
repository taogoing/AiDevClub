package service

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

const (
	notificationExchange     = "aidevclub.notifications"
	notificationDeadExchange = "aidevclub.notifications.dead"
	notificationQueue        = "aidevclub.notifications"
	notificationDeadQueue    = "aidevclub.notifications.dead"
	notificationRoute        = "notification.created"
	notificationRetryHeader  = "x-notification-retry"
	notificationMaxRetries   = 8
)

type NotificationMQ struct {
	url       string
	outbox    *repo.NotificationOutboxRepo
	notifs    *repo.NotificationRepo
	interval  time.Duration
	lease     time.Duration
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	startOnce sync.Once
}

func NewNotificationMQ(url string, outbox *repo.NotificationOutboxRepo, notifications *repo.NotificationRepo, interval, lease time.Duration) *NotificationMQ {
	return &NotificationMQ{url: url, outbox: outbox, notifs: notifications, interval: interval, lease: lease}
}

func (m *NotificationMQ) Start(parent context.Context) {
	m.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		m.cancel = cancel
		m.wg.Add(2)
		go func() { defer m.wg.Done(); m.publisherLoop(ctx) }()
		go func() { defer m.wg.Done(); m.consumerLoop(ctx) }()
	})
}

func (m *NotificationMQ) Close() {
	if m.cancel != nil {
		m.cancel()
	}
	done := make(chan struct{})
	go func() { m.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		slog.Warn("notification workers did not stop before shutdown timeout")
	}
}

func (m *NotificationMQ) publisherLoop(ctx context.Context) {
	for ctx.Err() == nil {
		conn, ch, err := m.connectChannel()
		if err != nil {
			slog.WarnContext(ctx, "notification outbox publisher cannot connect to RabbitMQ; events remain in MySQL", "error", err)
			if !sleepContext(ctx, 2*time.Second) {
				return
			}
			continue
		}
		if err := ch.Confirm(false); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			if !sleepContext(ctx, 2*time.Second) {
				return
			}
			continue
		}
		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
		returns := ch.NotifyReturn(make(chan amqp.Return, 1))
		publisher := &rabbitOutboxPublisher{channel: ch, confirms: confirms, returns: returns}
		worker := NewNotificationOutboxWorker(m.outbox, publisher, m.interval, m.lease)
		err = m.runPublisherConnection(ctx, worker, conn, ch)
		_ = ch.Close()
		_ = conn.Close()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.WarnContext(ctx, "notification outbox publisher reconnecting", "error", err)
		}
		if !sleepContext(ctx, 2*time.Second) {
			return
		}
	}
}

func (m *NotificationMQ) runPublisherConnection(ctx context.Context, worker *NotificationOutboxWorker, conn *amqp.Connection, ch *amqp.Channel) error {
	closed := conn.NotifyClose(make(chan *amqp.Error, 1))
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		for {
			processed, err := worker.RunBatch(ctx, 100)
			if err != nil {
				return err
			}
			if processed < 100 {
				break
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case err := <-closed:
			if err != nil {
				return err
			}
			return fmt.Errorf("RabbitMQ publisher connection closed")
		case <-ticker.C:
			if ch.IsClosed() {
				return fmt.Errorf("RabbitMQ publisher channel closed")
			}
		}
	}
}

func (m *NotificationMQ) consumerLoop(ctx context.Context) {
	consumer := NewNotificationEventConsumer(m.notifs)
	for ctx.Err() == nil {
		conn, ch, err := m.connectChannel()
		if err != nil {
			slog.WarnContext(ctx, "notification consumer cannot connect to RabbitMQ", "error", err)
			if !sleepContext(ctx, 2*time.Second) {
				return
			}
			continue
		}
		if err := declareNotificationTopology(ch); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			if !sleepContext(ctx, 2*time.Second) {
				return
			}
			continue
		}
		if err := ch.Confirm(false); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			if !sleepContext(ctx, 2*time.Second) {
				return
			}
			continue
		}
		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 16))
		returns := ch.NotifyReturn(make(chan amqp.Return, 16))
		if err := ch.Qos(16, 0, false); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			if !sleepContext(ctx, 2*time.Second) {
				return
			}
			continue
		}
		deliveries, err := ch.Consume(notificationQueue, "", false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			_ = conn.Close()
			if !sleepContext(ctx, 2*time.Second) {
				return
			}
			continue
		}
		err = m.consume(ctx, consumer, ch, confirms, returns, deliveries, conn)
		_ = ch.Close()
		_ = conn.Close()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.WarnContext(ctx, "notification consumer reconnecting", "error", err)
		}
		if !sleepContext(ctx, 2*time.Second) {
			return
		}
	}
}

func (m *NotificationMQ) consume(ctx context.Context, consumer *NotificationEventConsumer, ch *amqp.Channel, confirms <-chan amqp.Confirmation, returns <-chan amqp.Return, deliveries <-chan amqp.Delivery, conn *amqp.Connection) error {
	closed := conn.NotifyClose(make(chan *amqp.Error, 1))
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-closed:
			if err != nil {
				return err
			}
			return fmt.Errorf("RabbitMQ consumer connection closed")
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("RabbitMQ delivery channel closed")
			}
			if err := processBeforeAck(ctx, delivery.Body, consumer.Process, func() error { return delivery.Ack(false) }); err == nil {
				continue
			} else {
				if handleErr := m.retryOrDeadLetter(ctx, ch, confirms, returns, delivery, err); handleErr != nil {
					_ = delivery.Nack(false, true)
					return handleErr
				}
			}
		}
	}
}

func (m *NotificationMQ) retryOrDeadLetter(ctx context.Context, ch *amqp.Channel, confirms <-chan amqp.Confirmation, returns <-chan amqp.Return, delivery amqp.Delivery, processErr error) error {
	attempt := notificationRetryCount(delivery.Headers[notificationRetryHeader]) + 1
	headers := amqp.Table{notificationRetryHeader: int32(attempt), "x-notification-last-error": processErr.Error()}
	routingKey := "retry.1s"
	if attempt > notificationMaxRetries {
		if err := publishConfirmed(ctx, ch, confirms, returns, notificationDeadExchange, "dead", delivery.Body, delivery.MessageId, headers); err != nil {
			return fmt.Errorf("publish notification to dead-letter queue after %v: %w", processErr, err)
		}
		return delivery.Ack(false)
	}
	if attempt >= 3 {
		routingKey = "retry.5s"
	}
	if attempt >= 5 {
		routingKey = "retry.30s"
	}
	if err := publishConfirmed(ctx, ch, confirms, returns, notificationExchange, routingKey, delivery.Body, delivery.MessageId, headers); err != nil {
		return fmt.Errorf("schedule notification retry after %v: %w", processErr, err)
	}
	return delivery.Ack(false)
}

func notificationRetryCount(value any) int {
	switch n := value.(type) {
	case int32:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	case uint64:
		return int(n)
	default:
		return 0
	}
}

func (m *NotificationMQ) connectChannel() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.DialConfig(m.url, amqp.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 5*time.Second)
		},
	})
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := declareNotificationTopology(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, err
	}
	return conn, ch, nil
}

func declareNotificationTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(notificationExchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(notificationDeadExchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(notificationQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(notificationQueue, notificationRoute, notificationExchange, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(notificationDeadQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(notificationDeadQueue, "dead", notificationDeadExchange, false, nil); err != nil {
		return err
	}
	for _, retry := range []struct {
		key string
		ttl int32
	}{{"retry.1s", 1000}, {"retry.5s", 5000}, {"retry.30s", 30000}} {
		args := amqp.Table{"x-message-ttl": retry.ttl, "x-dead-letter-exchange": notificationExchange, "x-dead-letter-routing-key": notificationRoute}
		name := "aidevclub.notifications." + retry.key
		if _, err := ch.QueueDeclare(name, true, false, false, false, args); err != nil {
			return err
		}
		if err := ch.QueueBind(name, retry.key, notificationExchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}

type rabbitOutboxPublisher struct {
	channel  *amqp.Channel
	confirms <-chan amqp.Confirmation
	returns  <-chan amqp.Return
}

func (p *rabbitOutboxPublisher) Publish(ctx context.Context, event *model.NotificationOutboxEvent) error {
	return publishConfirmed(ctx, p.channel, p.confirms, p.returns, notificationExchange, notificationRoute, event.Payload, event.EventID, nil)
}

func publishConfirmed(ctx context.Context, ch *amqp.Channel, confirms <-chan amqp.Confirmation, returns <-chan amqp.Return, exchange, routingKey string, body []byte, messageID string, headers amqp.Table) error {
	if err := ch.PublishWithContext(ctx, exchange, routingKey, true, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: messageID, Timestamp: time.Now(), Headers: headers, Body: body,
	}); err != nil {
		return err
	}
	return awaitPublishConfirmation(ctx, confirms, returns, messageID)
}

func awaitPublishConfirmation(ctx context.Context, confirms <-chan amqp.Confirmation, returns <-chan amqp.Return, messageID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case returned, ok := <-returns:
		if !ok {
			return fmt.Errorf("RabbitMQ publisher return channel closed")
		}
		// RabbitMQ sends Basic.Return before the publisher confirm for a mandatory,
		// unroutable message. Consume that confirm too so it cannot be mistaken for
		// the next message's confirmation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case confirmation, ok := <-confirms:
			if !ok {
				return fmt.Errorf("RabbitMQ publisher confirmation channel closed")
			}
			if !confirmation.Ack {
				return fmt.Errorf("RabbitMQ negatively acknowledged %s", messageID)
			}
			return fmt.Errorf("RabbitMQ returned unroutable message %s: %s", messageID, returned.ReplyText)
		}
	case confirmation, ok := <-confirms:
		if !ok {
			return fmt.Errorf("RabbitMQ publisher confirmation channel closed")
		}
		if !confirmation.Ack {
			return fmt.Errorf("RabbitMQ negatively acknowledged %s", messageID)
		}
		// The AMQP reader dispatches a Basic.Return before its matching confirm.
		// Both channels are buffered, so also inspect a return that arrived in the
		// same frame batch before accepting the ACK.
		select {
		case returned, ok := <-returns:
			if !ok {
				return fmt.Errorf("RabbitMQ publisher return channel closed")
			}
			return fmt.Errorf("RabbitMQ returned unroutable message %s: %s", messageID, returned.ReplyText)
		default:
		}
		return nil
	}
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
