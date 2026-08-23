package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

func TestConfigDefaultsAndEnvOverrides(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.HubURL != "http://localhost:8000" {
		t.Fatalf("expected default HubURL http://localhost:8000, got %s", cfg.HubURL)
	}
	if cfg.BatchSize != 50 {
		t.Fatalf("expected default BatchSize 50, got %d", cfg.BatchSize)
	}

	// Test Get / Set
	if err := SetConfigValue(cfg, "hub_url", "http://test-hub:9000"); err != nil {
		t.Fatalf("failed to set hub_url: %v", err)
	}
	val, err := GetConfigValue(cfg, "hub_url")
	if err != nil || val != "http://test-hub:9000" {
		t.Fatalf("expected http://test-hub:9000, got %s (err: %v)", val, err)
	}

	if err := SetConfigValue(cfg, "batch_size", "100"); err != nil {
		t.Fatalf("failed to set batch_size: %v", err)
	}
	if cfg.BatchSize != 100 {
		t.Fatalf("expected batch_size 100, got %d", cfg.BatchSize)
	}

	// Test Env Overrides
	t.Setenv("TT_HUB_URL", "http://env-hub:8000")
	t.Setenv("TT_AUTH_TOKEN", "secret-env-token")
	t.Setenv("TT_MACHINE_ID", "machine-12345")
	t.Setenv("TT_LOG_LEVEL", "debug")
	ApplyEnvOverrides(cfg)

	if cfg.HubURL != "http://env-hub:8000" {
		t.Fatalf("expected env HubURL http://env-hub:8000, got %s", cfg.HubURL)
	}
	if cfg.AuthToken != "secret-env-token" {
		t.Fatalf("expected env AuthToken secret-env-token, got %s", cfg.AuthToken)
	}
	if cfg.MachineID != "machine-12345" {
		t.Fatalf("expected env MachineID machine-12345, got %s", cfg.MachineID)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected env LogLevel debug, got %s", cfg.LogLevel)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.HubURL = "http://saved-hub:8080"
	cfg.AuthToken = "saved-token"
	cfg.BatchSize = 25

	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.HubURL != "http://saved-hub:8080" {
		t.Fatalf("expected http://saved-hub:8080, got %s", loaded.HubURL)
	}
	if loaded.AuthToken != "saved-token" {
		t.Fatalf("expected saved-token, got %s", loaded.AuthToken)
	}
	if loaded.BatchSize != 25 {
		t.Fatalf("expected batch_size 25, got %d", loaded.BatchSize)
	}
}

func TestPipelineSendSyntheticAndPingHub(t *testing.T) {
	receivedBatch := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"version": "v2.0.0-test",
			})
		case "/api/v1/ingest":
			receivedBatch = true
			var batch models.IngestionBatch
			if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(models.IngestionResponse{
				Status:           "success",
				BatchID:          batch.Metadata.BatchID,
				AcceptedSessions: len(batch.Sessions),
				AcceptedTurns:    2,
				ServerTime:       time.Now().UTC(),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.HubURL = server.URL
	cfg.AuthToken = "test-token"

	sink := NewSilentSink()
	pipeline, err := NewPipeline(cfg, sink)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	ctx := context.Background()

	// Test PingHub
	health, err := pipeline.PingHub(ctx)
	if err != nil {
		t.Fatalf("ping hub failed: %v", err)
	}
	if health.Status != "ONLINE" {
		t.Fatalf("expected status ONLINE, got %s", health.Status)
	}
	if health.ServerVersion != "v2.0.0-test" {
		t.Fatalf("expected version v2.0.0-test, got %s", health.ServerVersion)
	}

	// Test SendSynthetic
	resp, err := pipeline.SendSynthetic(ctx, "claude_code", "pipeline-test", "claude-3-7-sonnet")
	if err != nil {
		t.Fatalf("send synthetic failed: %v", err)
	}
	if !receivedBatch {
		t.Fatalf("expected hub server to receive batch")
	}
	if resp.AcceptedSessions != 1 {
		t.Fatalf("expected 1 accepted session, got %d", resp.AcceptedSessions)
	}
}

func TestPipelineScanOnce(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptDir := filepath.Join(tmpDir, ".claude", "projects", "proj1")
	if err := os.MkdirAll(transcriptDir, 0755); err != nil {
		t.Fatalf("failed to create transcript dir: %v", err)
	}

	// Create a mock claude jsonl file
	filePath := filepath.Join(transcriptDir, "session.jsonl")
	content := `{"type":"user","timestamp":"2026-08-23T12:00:00Z","sessionId":"sess-1"}
{"type":"assistant","timestamp":"2026-08-23T12:00:05Z","sessionId":"sess-1","message":{"id":"msg-1","model":"claude-3-7-sonnet-20250219","usage":{"input_tokens":1000,"output_tokens":200,"cache_read_input_tokens":500,"cache_creation_input_tokens":0},"content":[{"type":"tool_use","name":"view_file"}]}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test transcript: %v", err)
	}

	cfg := DefaultConfig()
	cfg.ScanRoots = []string{transcriptDir}
	sink := NewSilentSink()

	pipeline, err := NewPipeline(cfg, sink)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	ctx := context.Background()
	summary, err := pipeline.ScanOnce(ctx, []string{transcriptDir}, true)
	if err != nil {
		t.Fatalf("scan once failed: %v", err)
	}

	if summary.TotalFiles != 1 {
		t.Fatalf("expected 1 file scanned, got %d", summary.TotalFiles)
	}
	if summary.ParsedSessions != 1 {
		t.Fatalf("expected 1 session parsed, got %d", summary.ParsedSessions)
	}
	if summary.TotalInputTokens != 1000 {
		t.Fatalf("expected 1000 input tokens, got %d", summary.TotalInputTokens)
	}
}
