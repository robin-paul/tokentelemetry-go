package client

import (
	"context"
	"sync"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// BufferConfig configures batching parameters for the collector buffer.
type BufferConfig struct {
	BatchSize     int           `json:"batch_size"`
	FlushInterval time.Duration `json:"flush_interval"`
	MaxQueueSize  int           `json:"max_queue_size"`
}

// Buffer accumulates sessions and dispatches them in batches to the Hub.
type Buffer struct {
	client    *Client
	cfg       BufferConfig
	queue     chan *models.Session
	stopChan  chan struct{}
	wg        sync.WaitGroup
	running   bool
	mu        sync.Mutex
	onSent    func(resp *models.IngestionResponse, err error)
	onSentMu  sync.RWMutex
}

// NewBuffer instantiates a new asynchronous Batch Buffer.
func NewBuffer(client *Client, cfg BufferConfig) *Buffer {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 500 * time.Millisecond
	}
	if cfg.MaxQueueSize <= 0 {
		cfg.MaxQueueSize = 5000
	}

	return &Buffer{
		client:   client,
		cfg:      cfg,
		queue:    make(chan *models.Session, cfg.MaxQueueSize),
		stopChan: make(chan struct{}),
	}
}

// SetCallback attaches a notification listener for completed batch transmissions.
func (b *Buffer) SetCallback(onSent func(resp *models.IngestionResponse, err error)) {
	b.onSentMu.Lock()
	defer b.onSentMu.Unlock()
	b.onSent = onSent
}

// Enqueue adds a session to the outbound batch buffer queue.
func (b *Buffer) Enqueue(sess *models.Session) bool {
	if sess == nil {
		return false
	}
	select {
	case b.queue <- sess:
		return true
	default:
		return false
	}
}

// Start launches the background transmitter loop.
func (b *Buffer) Start(ctx context.Context) {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	b.mu.Unlock()

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.workerLoop(ctx)
	}()
}

func (b *Buffer) workerLoop(ctx context.Context) {
	ticker := time.NewTicker(b.cfg.FlushInterval)
	defer ticker.Stop()

	var batch []*models.Session

	flush := func() {
		if len(batch) == 0 {
			return
		}
		toSend := batch
		batch = nil

		sessValues := make([]models.Session, len(toSend))
		for i, s := range toSend {
			sessValues[i] = *s
		}

		resp, err := b.client.SendBatch(ctx, sessValues)
		b.onSentMu.RLock()
		cb := b.onSent
		b.onSentMu.RUnlock()
		if cb != nil {
			cb(resp, err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-b.stopChan:
			flush()
			return
		case sess := <-b.queue:
			batch = append(batch, sess)
			if len(batch) >= b.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Close gracefully flushes the buffer and stops the worker.
func (b *Buffer) Close(ctx context.Context) {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.running = false
	close(b.stopChan)
	b.mu.Unlock()

	b.wg.Wait()
}
