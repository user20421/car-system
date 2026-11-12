package telemetry

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
)

type fakeBatchStore struct {
	mu      sync.Mutex
	batches [][]domain.Telemetry
}

func (f *fakeBatchStore) InsertTelemetryBatch(_ context.Context, items []domain.Telemetry) ([]domain.Telemetry, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copyItems := append([]domain.Telemetry(nil), items...)
	f.batches = append(f.batches, copyItems)
	return copyItems, 0, nil
}

type fakeHub struct {
	mu    sync.Mutex
	items []domain.Telemetry
}

func (f *fakeHub) Telemetry(t domain.Telemetry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = append(f.items, t)
}
func TestBoundedBatchPipelineFlushesOnClose(t *testing.T) {
	store := &fakeBatchStore{}
	hub := &fakeHub{}
	p := NewPipeline(store, nil, hub, 2, 8, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for i := 0; i < 5; i++ {
		item := domain.Telemetry{MessageID: "m", VIN: "LTEST000000000001", OccurredAtMS: int64(i + 1)}
		if err := p.Enqueue(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}
	p.Close()
	store.mu.Lock()
	defer store.mu.Unlock()
	count := 0
	for _, batch := range store.batches {
		count += len(batch)
	}
	if count != 5 {
		t.Fatalf("persisted %d items, want 5", count)
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if len(hub.items) != 5 {
		t.Fatalf("broadcast %d items, want 5", len(hub.items))
	}
}
func TestLatestScriptDoesNotRunWithoutRedis(t *testing.T) {
	p := NewPipeline(&fakeBatchStore{}, nil, nil, 1, 1, 1, slog.New(slog.NewTextHandler(io.Discard, nil)))
	done := make(chan struct{})
	go func() { p.PublishPersisted([]domain.Telemetry{{VIN: "LTEST000000000001"}}); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cache-free pipeline blocked")
	}
}
