package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
	"github.com/redis/go-redis/v9"
)

var ErrQueueFull = errors.New("telemetry queue full")

type BatchStore interface {
	InsertTelemetryBatch(context.Context, []domain.Telemetry) ([]domain.Telemetry, int, error)
}
type Broadcaster interface{ Telemetry(domain.Telemetry) }
type Pipeline struct {
	store          BatchStore
	cache          *redis.Client
	hub            Broadcaster
	log            *slog.Logger
	queue          chan domain.Telemetry
	workers, batch int
	wg             sync.WaitGroup
}

func NewPipeline(store BatchStore, cache *redis.Client, hub Broadcaster, workers, queueSize, batch int, log *slog.Logger) *Pipeline {
	p := &Pipeline{store: store, cache: cache, hub: hub, log: log, queue: make(chan domain.Telemetry, queueSize), workers: workers, batch: batch}
	p.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go p.run(i)
	}
	return p
}
func (p *Pipeline) Enqueue(ctx context.Context, t domain.Telemetry) error {
	select {
	case p.queue <- t:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}
func (p *Pipeline) Close() { close(p.queue); p.wg.Wait() }
func (p *Pipeline) run(worker int) {
	defer p.wg.Done()
	buffer := make([]domain.Telemetry, 0, p.batch)
	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	flush := func() {
		if len(buffer) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		accepted, duplicates, err := p.store.InsertTelemetryBatch(ctx, buffer)
		cancel()
		if err != nil {
			p.log.Error("telemetry batch persistence failed", "worker", worker, "items", len(buffer), "error", err)
		} else {
			p.log.Debug("telemetry batch persisted", "worker", worker, "accepted", len(accepted), "duplicates", duplicates)
			for _, t := range accepted {
				p.cacheLatest(t)
				if p.hub != nil {
					p.hub.Telemetry(t)
				}
			}
		}
		buffer = buffer[:0]
	}
	for {
		select {
		case t, ok := <-p.queue:
			if !ok {
				flush()
				return
			}
			buffer = append(buffer, t)
			if len(buffer) >= p.batch {
				flush()
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(200 * time.Millisecond)
			}
		case <-timer.C:
			flush()
			timer.Reset(200 * time.Millisecond)
		}
	}
}

var latestScript = redis.NewScript(`local current=redis.call('GET',KEYS[1]);if current then local ok,decoded=pcall(cjson.decode,current);if ok and decoded['occurred_at_ms'] and tonumber(decoded['occurred_at_ms'])>tonumber(ARGV[2]) then return 0 end end;redis.call('SET',KEYS[1],ARGV[1],'PX',ARGV[3]);return 1`)

// PublishPersisted updates the latest cache and user stream for HTTPS batch items
// after their transaction commits. It never sends telemetry through RabbitMQ.
func (p *Pipeline) PublishPersisted(items []domain.Telemetry) {
	for _, t := range items {
		p.cacheLatest(t)
		if p.hub != nil {
			p.hub.Telemetry(t)
		}
	}
}

func (p *Pipeline) cacheLatest(t domain.Telemetry) {
	if p.cache == nil {
		return
	}
	raw, err := json.Marshal(t)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := latestScript.Run(ctx, p.cache, []string{"car:telemetry:latest:" + t.VIN}, raw, t.OccurredAtMS, int64((7*24*time.Hour)/time.Millisecond)).Err(); err != nil {
		p.log.Warn("latest telemetry cache failed", "vin", t.VIN, "error", err)
	}
}
func Latest(ctx context.Context, cache *redis.Client, vin string) (domain.Telemetry, error) {
	raw, err := cache.Get(ctx, "car:telemetry:latest:"+vin).Bytes()
	if err != nil {
		return domain.Telemetry{}, err
	}
	var t domain.Telemetry
	err = json.Unmarshal(raw, &t)
	return t, err
}
