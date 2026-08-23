package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

func TestIngestBatchEndpoint(t *testing.T) {
	_, db, handler, cleanup := setupTestServer(t)
	defer cleanup()

	now := time.Now().UTC().Truncate(time.Second)
	batch := models.IngestionBatch{
		Metadata: models.ClientMetadata{
			MachineID:     "macbook-pro-m3",
			Hostname:      "mbp.local",
			ClientVersion: "1.0.0",
			User:          "dev-user",
			SentAt:        now,
			BatchID:       "batch-001",
		},
		Sessions: []models.Session{
			{
				ID:          "claude_session_1",
				SessionID:   "s1",
				AgentName:   "claude_code",
				ProjectName: "project-x",
				FilePath:    "/logs/s1.jsonl",
				StartTime:   now,
				EndTime:     now.Add(10 * time.Minute),
				ModelRaw:    "claude-3-7-sonnet",
				InputTokens: 5000,
				OutputTokens: 1000,
				NetCostUSD:  0.08,
				Turns: []models.MessageTurn{
					{
						ID:          "turn_1",
						SessionID:   "claude_session_1",
						TurnIndex:   0,
						Timestamp:   now,
						Role:        "assistant",
						ModelName:   "claude-3-7-sonnet",
						InputTokens: 5000,
						OutputTokens: 1000,
						CostUSD:     0.08,
					},
				},
			},
		},
	}

	body, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("failed to marshal batch: %v", err)
	}

	// 1. Post to /api/v1/ingest
	req := newLocalRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.IngestionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.AcceptedSessions != 1 {
		t.Errorf("expected 1 accepted session, got %d", resp.AcceptedSessions)
	}
	if resp.AcceptedTurns != 1 {
		t.Errorf("expected 1 accepted turn, got %d", resp.AcceptedTurns)
	}

	// 2. Verify persisted in DB
	sess, err := db.GetSessionDetail(context.Background(), "claude_session_1")
	if err != nil {
		t.Fatalf("failed to retrieve ingested session from DB: %v", err)
	}
	if sess.MachineID != "macbook-pro-m3" {
		t.Errorf("expected MachineID 'macbook-pro-m3', got %q", sess.MachineID)
	}
	if len(sess.Turns) != 1 {
		t.Fatalf("expected 1 turn in DB, got %d", len(sess.Turns))
	}

	// 3. Post identical batch again (Idempotency test)
	req2 := newLocalRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200 on replay, got %d", rec2.Code)
	}

	// Turns should still be 1 (replaced, not duplicated)
	sessReplay, err := db.GetSessionDetail(context.Background(), "claude_session_1")
	if err != nil {
		t.Fatalf("failed to retrieve session after replay: %v", err)
	}
	if len(sessReplay.Turns) != 1 {
		t.Errorf("expected 1 turn after replay (idempotency), got %d", len(sessReplay.Turns))
	}
}

func TestIngestBatchMissingMachineID(t *testing.T) {
	_, _, handler, cleanup := setupTestServer(t)
	defer cleanup()

	batch := models.IngestionBatch{
		Metadata: models.ClientMetadata{
			// Missing MachineID
			Hostname: "mbp.local",
		},
		Sessions: []models.Session{},
	}

	body, _ := json.Marshal(batch)
	req := newLocalRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing machine ID, got %d", rec.Code)
	}
}
