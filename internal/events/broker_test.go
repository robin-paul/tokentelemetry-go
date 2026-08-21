package events

import (
	"bufio"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

func TestBrokerSubscribeAndBroadcast(t *testing.T) {
	broker := NewBroker(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go broker.Start(ctx)
	// Allow broker loop to start
	time.Sleep(10 * time.Millisecond)

	ch1 := broker.Subscribe()
	ch2 := broker.Subscribe()

	// Wait for registration
	time.Sleep(20 * time.Millisecond)

	if count := broker.SubscriberCount(); count != 2 {
		t.Fatalf("expected 2 subscribers, got %d", count)
	}

	payload := EventPayload{
		Type:      EventSessionCreated,
		Timestamp: time.Now().UTC(),
		Data: SessionEventData{
			Session: models.Session{
				ID:        "test-sess-1",
				AgentName: "claude",
			},
		},
	}

	err := broker.Broadcast(payload)
	if err != nil {
		t.Fatalf("unexpected broadcast error: %v", err)
	}

	select {
	case msg := <-ch1:
		if !strings.Contains(string(msg), "session.created") {
			t.Errorf("expected session.created event, got %s", string(msg))
		}
		if !strings.Contains(string(msg), "test-sess-1") {
			t.Errorf("expected test-sess-1 in data, got %s", string(msg))
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for message on ch1")
	}

	select {
	case msg := <-ch2:
		if !strings.Contains(string(msg), "session.created") {
			t.Errorf("expected session.created event, got %s", string(msg))
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for message on ch2")
	}

	// Test Unsubscribe
	broker.Unsubscribe(ch1)
	time.Sleep(20 * time.Millisecond)

	if count := broker.SubscriberCount(); count != 1 {
		t.Fatalf("expected 1 subscriber after unsubscribe, got %d", count)
	}

	broker.Stop()
}

func TestBrokerKeepalivePing(t *testing.T) {
	broker := NewBroker(30 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go broker.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	ch := broker.Subscribe()
	time.Sleep(10 * time.Millisecond)

	select {
	case msg := <-ch:
		if string(msg) != ": ping\n\n" {
			t.Errorf("expected keepalive ping comment, got %s", string(msg))
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for keepalive ping")
	}

	broker.Stop()
}

func TestBrokerHTTPStreaming(t *testing.T) {
	broker := NewBroker(100 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go broker.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	reqCtx, reqCancel := context.WithCancel(context.Background())
	defer reqCancel()

	req := httptest.NewRequest("GET", "/events", nil).WithContext(reqCtx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		broker.ServeHTTP(w, req)
		close(done)
	}()

	// Wait for connection to establish
	time.Sleep(30 * time.Millisecond)

	err := broker.Broadcast(EventPayload{
		Type: EventScanProgress,
		Data: ScanProgressData{
			TotalFiles:     10,
			ProcessedFiles: 5,
			CurrentFile:    "/tmp/test.jsonl",
		},
	})
	if err != nil {
		t.Fatalf("broadcast error: %v", err)
	}

	time.Sleep(30 * time.Millisecond)
	reqCancel()
	<-done

	res := w.Result()
	if res.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", res.Header.Get("Content-Type"))
	}

	body := w.Body.String()
	scanner := bufio.NewScanner(strings.NewReader(body))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if !strings.Contains(body, ": connected") {
		t.Errorf("expected initial : connected comment in body: %s", body)
	}
	if !strings.Contains(body, "event: scan.progress") {
		t.Errorf("expected event: scan.progress in body: %s", body)
	}
	if !strings.Contains(body, "processed_files") {
		t.Errorf("expected processed_files in body: %s", body)
	}

	broker.Stop()
}
