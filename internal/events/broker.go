package events

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	defaultPingInterval = 15 * time.Second
	clientBufferSize    = 64
)

// Broker maintains thread-safe client SSE connections and broadcasts events.
type Broker struct {
	mu           sync.RWMutex
	subscribers  map[chan []byte]struct{}
	register     chan chan []byte
	unregister   chan chan []byte
	broadcast    chan []byte
	stopChan     chan struct{}
	pingInterval time.Duration
	running      bool
}

// NewBroker constructs an initialized Broker.
func NewBroker(pingInterval time.Duration) *Broker {
	if pingInterval <= 0 {
		pingInterval = defaultPingInterval
	}
	return &Broker{
		subscribers:  make(map[chan []byte]struct{}),
		register:     make(chan chan []byte),
		unregister:   make(chan chan []byte),
		broadcast:    make(chan []byte, 256),
		stopChan:     make(chan struct{}),
		pingInterval: pingInterval,
	}
}

// Start runs the event dispatcher loop and 15s keepalive ping ticker.
func (b *Broker) Start(ctx context.Context) {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	b.mu.Unlock()

	ticker := time.NewTicker(b.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.Stop()
			return
		case <-b.stopChan:
			return
		case ch := <-b.register:
			b.mu.Lock()
			b.subscribers[ch] = struct{}{}
			b.mu.Unlock()
		case ch := <-b.unregister:
			b.mu.Lock()
			if _, exists := b.subscribers[ch]; exists {
				delete(b.subscribers, ch)
				close(ch)
			}
			b.mu.Unlock()
		case msg := <-b.broadcast:
			b.mu.RLock()
			for ch := range b.subscribers {
				select {
				case ch <- msg:
				default:
					// Slow consumer: drop message to avoid blocking the broker
				}
			}
			b.mu.RUnlock()
		case <-ticker.C:
			// SSE comment keepalive heartbeat
			pingMsg := []byte(": ping\n\n")
			b.mu.RLock()
			for ch := range b.subscribers {
				select {
				case ch <- pingMsg:
				default:
				}
			}
			b.mu.RUnlock()
		}
	}
}

// Stop closes all client subscriber channels and stops the broker.
func (b *Broker) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.running {
		return
	}
	b.running = false
	close(b.stopChan)
	for ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, ch)
	}
}

// Subscribe registers a new buffered client channel.
func (b *Broker) Subscribe() chan []byte {
	ch := make(chan []byte, clientBufferSize)
	b.register <- ch
	return ch
}

// Unsubscribe removes a client channel.
func (b *Broker) Unsubscribe(ch chan []byte) {
	b.unregister <- ch
}

// SubscriberCount returns the current count of active subscribers.
func (b *Broker) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// Broadcast dispatches an EventPayload to all connected subscribers.
func (b *Broker) Broadcast(event EventPayload) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	sseBytes, err := event.ToSSE()
	if err != nil {
		return fmt.Errorf("failed to format SSE event: %w", err)
	}
	b.BroadcastRaw(sseBytes)
	return nil
}

// BroadcastRaw sends raw bytes to all subscribers.
func (b *Broker) BroadcastRaw(msg []byte) {
	b.mu.RLock()
	running := b.running
	b.mu.RUnlock()
	if !running {
		return
	}
	select {
	case b.broadcast <- msg:
	default:
	}
}

// ServeHTTP implements http.Handler for Server-Sent Events clients.
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send initial connection comment
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	clientChan := b.Subscribe()
	defer b.Unsubscribe(clientChan)

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			_, err := w.Write(msg)
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
