package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

func TestClientSendBatchSuccess(t *testing.T) {
	var receivedHeaderMachineID string
	var receivedAuth string
	var receivedBatch models.IngestionBatch

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ingest" {
			http.NotFound(w, r)
			return
		}
		receivedHeaderMachineID = r.Header.Get("X-TT-Machine-ID")
		receivedAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&receivedBatch)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models.IngestionResponse{
			Status:           "success",
			BatchID:          receivedBatch.Metadata.BatchID,
			AcceptedSessions: len(receivedBatch.Sessions),
			AcceptedTurns:    1,
			ServerTime:       time.Now().UTC(),
		})
	}))
	defer ts.Close()

	c := NewClient(Config{
		HubURL:        ts.URL,
		AuthToken:     "test-token-xyz",
		MachineID:     "machine-123",
		Hostname:      "test-host",
		ClientVersion: "1.0.0",
		User:          "test-user",
	})

	sessions := []models.Session{
		{
			ID:          "s1",
			SessionID:   "s1",
			AgentName:   "claude_code",
			ProjectName: "test-proj",
		},
	}

	resp, err := c.SendBatch(context.Background(), sessions)
	if err != nil {
		t.Fatalf("SendBatch failed: %v", err)
	}

	if resp.AcceptedSessions != 1 {
		t.Errorf("expected 1 accepted session, got %d", resp.AcceptedSessions)
	}
	if receivedHeaderMachineID != "machine-123" {
		t.Errorf("expected header X-TT-Machine-ID 'machine-123', got %q", receivedHeaderMachineID)
	}
	if receivedAuth != "Bearer test-token-xyz" {
		t.Errorf("expected Authorization 'Bearer test-token-xyz', got %q", receivedAuth)
	}
	if receivedBatch.Metadata.MachineID != "machine-123" {
		t.Errorf("expected body metadata MachineID 'machine-123', got %q", receivedBatch.Metadata.MachineID)
	}
}

func TestClientRetryWithJitter(t *testing.T) {
	var attempts int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models.IngestionResponse{
			Status:           "success",
			AcceptedSessions: 1,
		})
	}))
	defer ts.Close()

	c := NewClient(Config{
		HubURL:         ts.URL,
		MachineID:      "m1",
		MaxRetries:     4,
		BaseRetryDelay: 10 * time.Millisecond,
		MaxRetryDelay:  50 * time.Millisecond,
	})

	sessions := []models.Session{
		{ID: "s1"},
	}

	resp, err := c.SendBatch(context.Background(), sessions)
	if err != nil {
		t.Fatalf("SendBatch failed on retry: %v", err)
	}

	if resp.AcceptedSessions != 1 {
		t.Errorf("expected 1 accepted session, got %d", resp.AcceptedSessions)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestBufferFlushOnSizeAndInterval(t *testing.T) {
	var receivedBatches int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedBatches, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models.IngestionResponse{
			Status: "success",
		})
	}))
	defer ts.Close()

	c := NewClient(Config{
		HubURL:    ts.URL,
		MachineID: "m1",
	})

	buf := NewBuffer(c, BufferConfig{
		BatchSize:     2,
		FlushInterval: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf.Start(ctx)

	// 1. Enqueue 2 items (triggers flush by size)
	buf.Enqueue(&models.Session{ID: "s1"})
	buf.Enqueue(&models.Session{ID: "s2"})

	// Give short time for batch size flush
	time.Sleep(30 * time.Millisecond)
	if atomic.LoadInt32(&receivedBatches) < 1 {
		t.Errorf("expected at least 1 batch sent by size trigger, got %d", atomic.LoadInt32(&receivedBatches))
	}

	// 2. Enqueue 1 item (triggers flush by ticker interval)
	buf.Enqueue(&models.Session{ID: "s3"})
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&receivedBatches) < 2 {
		t.Errorf("expected at least 2 batches sent after interval trigger, got %d", atomic.LoadInt32(&receivedBatches))
	}

	buf.Close(context.Background())
}
