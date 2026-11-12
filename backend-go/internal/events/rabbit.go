package events

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/car-system/car-system/backend-go/internal/store"
	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "car.events"
const Queue = "vehicle.commands"
const DLX = "car.dlx"

type Bus struct {
	conn      *amqp.Connection
	publisher *amqp.Channel
	healthy   atomic.Bool
	log       *slog.Logger
}

func Dial(rawURL, caFile string, log *slog.Logger) (*Bus, error) {
	ca, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("invalid CA")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := parsed.Hostname()
	conn, err := amqp.DialTLS(rawURL, &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: host})
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	b := &Bus{conn: conn, publisher: ch, log: log}
	if err := b.topology(ch); err != nil {
		b.Close()
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		b.Close()
		return nil, err
	}
	b.healthy.Store(true)
	go func() {
		closeErr := <-conn.NotifyClose(make(chan *amqp.Error, 1))
		if closeErr != nil {
			log.Error("RabbitMQ connection closed", "error", closeErr)
		}
		b.healthy.Store(false)
	}()
	return b, nil
}
func (b *Bus) topology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(DLX, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	dead, err := ch.QueueDeclare("vehicle.commands.dlq", true, false, false, false, amqp.Table{"x-queue-type": "quorum"})
	if err != nil {
		return err
	}
	if err := ch.QueueBind(dead.Name, "", DLX, false, nil); err != nil {
		return err
	}
	q, err := ch.QueueDeclare(Queue, true, false, false, false, amqp.Table{"x-queue-type": "quorum", "x-dead-letter-exchange": DLX, "x-delivery-limit": 5})
	if err != nil {
		return err
	}
	return ch.QueueBind(q.Name, "command.created", Exchange, false, nil)
}
func (b *Bus) Healthy() bool { return b.healthy.Load() }
func (b *Bus) Close() error {
	b.healthy.Store(false)
	if b.publisher != nil {
		_ = b.publisher.Close()
	}
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}
func (b *Bus) Publish(ctx context.Context, event store.OutboxEvent) error {
	confirmation, err := b.publisher.PublishWithDeferredConfirmWithContext(ctx, Exchange, "command.created", true, false, amqp.Publishing{MessageId: event.ID, Type: event.Type, ContentType: "application/json", DeliveryMode: amqp.Persistent, Timestamp: time.UnixMilli(event.CreatedAtMS), Body: event.Payload})
	if err != nil {
		return err
	}
	if confirmation == nil {
		return fmt.Errorf("RabbitMQ publisher confirms are not enabled")
	}
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	acked, err := confirmation.WaitContext(waitCtx)
	if err != nil {
		return fmt.Errorf("publisher confirm: %w", err)
	}
	if !acked {
		return fmt.Errorf("RabbitMQ publisher nack")
	}
	return nil
}
func (b *Bus) Relay(ctx context.Context, db *store.Store) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !b.Healthy() {
				continue
			}
			publishCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := db.PublishOutbox(publishCtx, 50, b.Publish)
			cancel()
			if err != nil && ctx.Err() == nil {
				b.log.Error("outbox relay failed", "error", err)
			}
		}
	}
}

type Handler interface {
	Dispatch(context.Context, string, []byte) error
}

func (b *Bus) Consume(ctx context.Context, handler Handler) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err := ch.Qos(32, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume(Queue, "car-command-dispatcher", false, false, false, false, nil)
	if err != nil {
		return err
	}
	if err := ch.Confirm(false); err != nil {
		return err
	}
	sem := make(chan struct{}, 32)
	errCh := make(chan error, 1)
	retryMu := &sync.Mutex{}
	var workers sync.WaitGroup
	defer workers.Wait()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("consumer channel closed")
			}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer func() { <-sem }()
				if err := b.consumeDelivery(ctx, ch, retryMu, handler, d); err != nil {
					select {
					case errCh <- err:
					default:
					}
				}
			}()
		}
	}
}

func (b *Bus) consumeDelivery(ctx context.Context, ch *amqp.Channel, retryMu *sync.Mutex, handler Handler, d amqp.Delivery) error {
	processCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	err := handler.Dispatch(processCtx, d.MessageId, d.Body)
	cancel()
	if err == nil {
		return d.Ack(false)
	}
	b.log.Warn("command dispatch failed", "message_id", d.MessageId, "error", err)
	retry := int32(0)
	if value, ok := d.Headers["x-retry-count"].(int32); ok {
		retry = value
	}
	if retry >= 3 {
		return d.Nack(false, false)
	}
	headers := amqp.Table{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = retry + 1
	// A confirmation listener belongs to one publish; serialize retries so
	// concurrent VIN workers cannot consume each other's confirmations.
	retryMu.Lock()
	publishCtx, publishCancel := context.WithTimeout(ctx, 5*time.Second)
	confirmation, publishErr := ch.PublishWithDeferredConfirmWithContext(publishCtx, Exchange, "command.created", true, false, amqp.Publishing{MessageId: d.MessageId, Type: d.Type, ContentType: d.ContentType, DeliveryMode: amqp.Persistent, Timestamp: d.Timestamp, Headers: headers, Body: d.Body})
	if publishErr == nil && confirmation == nil {
		publishErr = fmt.Errorf("retry publisher confirms are not enabled")
	}
	if publishErr == nil {
		acked, waitErr := confirmation.WaitContext(publishCtx)
		if waitErr != nil {
			publishErr = fmt.Errorf("retry confirm: %w", waitErr)
		} else if !acked {
			publishErr = fmt.Errorf("retry nack")
		}
	}
	publishCancel()
	retryMu.Unlock()
	if publishErr != nil {
		return d.Nack(false, true)
	}
	return d.Ack(false)
}
func DecodeCommand(body []byte, target any) error {
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode command event: %w", err)
	}
	return nil
}
