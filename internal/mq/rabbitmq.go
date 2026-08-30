package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrUnavailable = errors.New("rabbitmq unavailable")

type OrderEvent struct {
	RequestID  string `json:"request_id"`
	UserID     uint64 `json:"user_id"`
	ActivityID uint64 `json:"activity_id"`
}

type Publisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	queue    string
	confirms chan amqp.Confirmation
	mu       sync.Mutex
}

func New(url, queue string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if _, err = ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare queue: %w", err)
	}
	if err = ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}
	return &Publisher{conn: conn, ch: ch, queue: queue, confirms: ch.NotifyPublish(make(chan amqp.Confirmation, 1))}, nil
}

func (p *Publisher) Publish(ctx context.Context, event OrderEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn.IsClosed() {
		return ErrUnavailable
	}
	if err := p.ch.PublishWithContext(ctx, "", p.queue, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Timestamp: time.Now(), Body: body}); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	select {
	case confirmation := <-p.confirms:
		if !confirmation.Ack {
			return fmt.Errorf("%w: broker rejected message", ErrUnavailable)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("%w: publisher confirm timeout", ErrUnavailable)
	}
}

func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

type Consumer struct {
	conn  *amqp.Connection
	ch    *amqp.Channel
	queue string
}

func NewConsumer(url, queue string, prefetch int) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if _, err = ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	if prefetch <= 0 {
		prefetch = 10
	}
	if err = ch.Qos(prefetch, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &Consumer{conn: conn, ch: ch, queue: queue}, nil
}

func (c *Consumer) Run(ctx context.Context, handle func(context.Context, OrderEvent) error) error {
	deliveries, err := c.ch.Consume(c.queue, "seckill-worker", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return ErrUnavailable
			}
			var event OrderEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				_ = d.Reject(false)
				continue
			}
			if err := handle(ctx, event); err != nil {
				_ = d.Nack(false, false)
				continue
			}
			if err := d.Ack(false); err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) Close() error {
	if c == nil {
		return nil
	}
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
