package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/api"
	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/robin-paul/tokentelemetry-go/internal/events"
	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

func TestEndToEndCLItoHubStreaming(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "hub_e2e.db")

	// 1. Initialize Hub Server with SQLite
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	pe, err := pricing.NewEngine()
	if err != nil {
		t.Fatalf("failed to create pricing engine: %v", err)
	}

	broker := events.NewBroker(5 * time.Second)
	go broker.Start(ctx)
	defer broker.Stop()

	scannerEngine := scanner.NewEngine(db, pe, scanner.Config{})

	authToken := "secret-e2e-token-12345"
	apiServer := api.NewServer(db, pe, scannerEngine, broker, api.Config{
		AuthToken:  authToken,
		Version:    "2.0.0-e2e",
		Commit:     "test",
		WebHandler: http.NotFoundHandler(),
	})

	hubServer := httptest.NewServer(apiServer.Router())
	defer hubServer.Close()

	// 2. Initialize Collector Pipeline pointing at Hub Server
	collectorCfg := &collector.Config{
		HubURL:       hubServer.URL,
		AuthToken:    authToken,
		MachineID:    "e2e-worker-node-1",
		LogLevel:     "debug",
		BatchSize:    10,
		FlushMS:      100,
		MaxRetries:   3,
		TimeoutSec:   5,
	}

	silentSink := collector.NewSilentSink()
	pipeline, err := collector.NewPipeline(collectorCfg, silentSink)
	if err != nil {
		t.Fatalf("failed to initialize collector pipeline: %v", err)
	}

	// 3. Test PingHub Connectivity
	health, err := pipeline.PingHub(ctx)
	if err != nil {
		t.Fatalf("ping hub failed: %v", err)
	}
	if health.Status != "ONLINE" {
		t.Fatalf("expected hub status ONLINE, got %s (err: %s)", health.Status, health.Error)
	}

	// 4. Test SendSynthetic Session
	resp, err := pipeline.SendSynthetic(ctx, "claude_code", "e2e-synthetic-proj", "claude-3-7-sonnet")
	if err != nil {
		t.Fatalf("send synthetic failed: %v", err)
	}
	if resp.Status != "success" || resp.AcceptedSessions != 1 || resp.AcceptedTurns != 2 {
		t.Fatalf("unexpected synthetic ingestion response: %+v", resp)
	}

	// 5. Query Hub REST API: Verify Session Ingested in Hub DB
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, hubServer.URL+"/api/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)
	sessionsResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to query /api/sessions: %v", err)
	}
	defer sessionsResp.Body.Close()

	if sessionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /api/sessions, got %d", sessionsResp.StatusCode)
	}

	var sessionList []map[string]interface{}
	if err := json.NewDecoder(sessionsResp.Body).Decode(&sessionList); err != nil {
		t.Fatalf("failed to decode sessions JSON: %v", err)
	}

	if len(sessionList) != 1 {
		t.Fatalf("expected 1 session on hub, got %d", len(sessionList))
	}
	if sessionList[0]["agent_name"] != "claude_code" {
		t.Fatalf("expected agent claude_code, got %v", sessionList[0]["agent_name"])
	}
	if sessionList[0]["machine_id"] != "e2e-worker-node-1" {
		t.Fatalf("expected machine_id e2e-worker-node-1, got %v", sessionList[0]["machine_id"])
	}

	// 6. Test ScanOnce with Real Log File
	transcriptDir := filepath.Join(tmpDir, ".claude", "projects", "real-e2e-project")
	if err := os.MkdirAll(transcriptDir, 0755); err != nil {
		t.Fatalf("failed to create transcript dir: %v", err)
	}

	sampleFile := filepath.Join(transcriptDir, "real_session.jsonl")
	content := `{"type":"user","timestamp":"2026-08-23T14:00:00Z","sessionId":"sess-e2e-real-1"}
{"type":"assistant","timestamp":"2026-08-23T14:00:10Z","sessionId":"sess-e2e-real-1","message":{"id":"msg-e2e-1","model":"claude-3-7-sonnet-20250219","usage":{"input_tokens":15000,"output_tokens":3200,"cache_read_input_tokens":8000,"cache_creation_input_tokens":0},"content":[{"type":"tool_use","name":"write_to_file"}]}}
`
	if err := os.WriteFile(sampleFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test log file: %v", err)
	}

	scanSummary, err := pipeline.ScanOnce(ctx, []string{transcriptDir}, false)
	if err != nil {
		t.Fatalf("scan once failed: %v", err)
	}

	if scanSummary.ParsedSessions != 1 || scanSummary.AcceptedSessions != 1 {
		t.Fatalf("expected 1 session parsed and accepted, got parsed=%d accepted=%d errors=%v",
			scanSummary.ParsedSessions, scanSummary.AcceptedSessions, scanSummary.Errors)
	}

	// 7. Verify Hub Daily Summary Rollup
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, hubServer.URL+"/api/stats/daily", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)
	dailyResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to query /api/stats/daily: %v", err)
	}
	defer dailyResp.Body.Close()

	var dailySummaries []map[string]interface{}
	if err := json.NewDecoder(dailyResp.Body).Decode(&dailySummaries); err != nil {
		t.Fatalf("failed to decode daily summary JSON: %v", err)
	}

	if len(dailySummaries) == 0 {
		t.Fatalf("expected daily summaries rollup on hub")
	}

	// 8. Test Auth Rejection with Invalid Token for Remote Caller
	remoteAuthMW := api.RemoteAuthMiddleware(authToken)
	remoteHandler := remoteAuthMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recUnauth := httptest.NewRecorder()
	reqUnauth := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", nil)
	reqUnauth.RemoteAddr = "192.168.1.100:54321" // Non-loopback IP
	reqUnauth.Header.Set("Authorization", "Bearer invalid-token")
	remoteHandler.ServeHTTP(recUnauth, reqUnauth)

	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for remote request with invalid token, got %d", recUnauth.Code)
	}

	recAuth := httptest.NewRecorder()
	reqAuth := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", nil)
	reqAuth.RemoteAddr = "192.168.1.100:54321" // Non-loopback IP
	reqAuth.Header.Set("Authorization", "Bearer "+authToken)
	remoteHandler.ServeHTTP(recAuth, reqAuth)

	if recAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for remote request with valid token, got %d", recAuth.Code)
	}

	fmt.Println("E2E CLI-to-Hub streaming and authentication test passed successfully.")
}
