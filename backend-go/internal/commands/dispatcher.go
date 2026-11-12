package commands

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
	"github.com/car-system/car-system/backend-go/internal/events"
	"github.com/car-system/car-system/backend-go/internal/realtime"
	"github.com/car-system/car-system/backend-go/internal/store"
)

type Publisher interface {
	PublishCommand(context.Context, domain.Command) error
}
type dispatchJob struct {
	eventID string
	command domain.Command
	result  chan error
}
type Dispatcher struct {
	store     *store.Store
	publisher Publisher
	hub       *realtime.Hub
	log       *slog.Logger
	mu        sync.Mutex
	lanes     map[string]chan dispatchJob
}

func NewDispatcher(db *store.Store, publisher Publisher, hub *realtime.Hub, log *slog.Logger) *Dispatcher {
	return &Dispatcher{store: db, publisher: publisher, hub: hub, log: log, lanes: map[string]chan dispatchJob{}}
}
func (d *Dispatcher) Dispatch(ctx context.Context, eventID string, body []byte) error {
	d.log.Debug("command event received", "event_id", eventID)
	var c domain.Command
	if err := events.DecodeCommand(body, &c); err != nil {
		return nil
	}
	processed, err := d.store.DispatchProcessed(ctx, eventID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}
	d.mu.Lock()
	lane := d.lanes[c.VIN]
	if lane == nil {
		lane = make(chan dispatchJob, 128)
		d.lanes[c.VIN] = lane
		go d.run(c.VIN, lane)
	}
	d.mu.Unlock()
	job := dispatchJob{eventID: eventID, command: c, result: make(chan error, 1)}
	select {
	case lane <- job:
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("VIN dispatch lane full")
	}
	select {
	case err := <-job.result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (d *Dispatcher) run(vin string, lane <-chan dispatchJob) {
	for job := range lane {
		job.result <- d.process(job)
	}
}
func (d *Dispatcher) process(job dispatchJob) error {
	d.log.Debug("dispatch process started", "event_id", job.eventID, "command_id", job.command.CommandID)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	processed, err := d.store.DispatchProcessed(ctx, job.eventID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}
	current, err := d.store.CommandByID(ctx, job.command.CommandID)
	if err != nil {
		return err
	}
	d.log.Debug("dispatch command loaded", "command_id", current.CommandID, "status", current.Status)
	if current.ExpiresAtMS <= time.Now().UnixMilli() {
		updated, changed, err := d.store.ApplyCommandStatus(ctx, current.CommandID, "expired", &domain.CommandError{Code: "COMMAND_EXPIRED", Message: "command expired before dispatch", Retryable: false}, time.Now().UnixMilli())
		if err == nil && changed {
			d.hub.Command(updated)
		}
		return err
	}
	// Persist the dispatch attempt before exposing it to a fast device. The
	// RabbitMQ delivery is still acknowledged only after MQTT publication.
	sent, changed, err := d.store.ApplyCommandStatus(ctx, current.CommandID, "sent", nil, time.Now().UnixMilli())
	if err != nil {
		return err
	}
	if changed {
		d.hub.Command(sent)
	}
	d.log.Debug("command marked sent", "command_id", current.CommandID)
	d.log.Debug("publishing MQTT command", "command_id", current.CommandID, "vin", current.VIN)
	if err := d.publisher.PublishCommand(ctx, current); err != nil {
		return err
	}
	d.log.Debug("MQTT command published", "command_id", current.CommandID)
	inserted, err := d.store.RecordDispatch(ctx, job.eventID, current.CommandID, time.Now().UnixMilli())
	if err != nil {
		return err
	}
	if inserted {
		updated, err := d.store.CommandByID(ctx, current.CommandID)
		if err == nil {
			d.hub.Command(updated)
		}
	}
	return nil
}
func (d *Dispatcher) ExpireLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			commands, err := d.store.ExpireCommands(ctx, time.Now().UnixMilli(), 100)
			if err != nil {
				d.log.Warn("command expiry sweep failed", "error", err)
			}
			for _, c := range commands {
				d.hub.Command(c)
			}
		}
	}
}
