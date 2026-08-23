package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

func TestCobraRootHelp(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running --help, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "TokenTelemetry Collector (tt)") {
		t.Fatalf("expected help text to contain collector summary, got: %s", out)
	}
	if !strings.Contains(out, "watch") || !strings.Contains(out, "scan") || !strings.Contains(out, "config") {
		t.Fatalf("expected subcommands to be listed in help, got: %s", out)
	}
}

func TestCobraConfigCommands(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom-config.yaml")

	// 1. Config Set
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"--config", configPath, "config", "set", "hub_url", "http://myhub:8080"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to run config set: %v", err)
	}

	// 2. Config Get
	cmd = NewRootCmd()
	getBuf := new(bytes.Buffer)
	cmd.SetOut(getBuf)
	cmd.SetArgs([]string{"--config", configPath, "config", "get", "hub_url"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to run config get: %v", err)
	}
	if !strings.Contains(getBuf.String(), "http://myhub:8080") {
		t.Fatalf("expected http://myhub:8080 in get output, got: %s", getBuf.String())
	}

	// 3. Config Path
	cmd = NewRootCmd()
	pathBuf := new(bytes.Buffer)
	cmd.SetOut(pathBuf)
	cmd.SetArgs([]string{"--config", configPath, "config", "path"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to run config path: %v", err)
	}
	if !strings.Contains(pathBuf.String(), configPath) {
		t.Fatalf("expected %s in path output, got: %s", configPath, pathBuf.String())
	}
}

func TestCobraStatusAndSendWithMockHub(t *testing.T) {
	receivedBatch := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"version": "v2.0.0-cli-test",
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

	// 1. Test tt status
	cmd := NewRootCmd()
	statusBuf := new(bytes.Buffer)
	cmd.SetOut(statusBuf)
	cmd.SetArgs([]string{"--hub", server.URL, "status"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to run tt status: %v", err)
	}

	// 2. Test tt send (synthetic)
	cmd = NewRootCmd()
	sendBuf := new(bytes.Buffer)
	cmd.SetOut(sendBuf)
	cmd.SetArgs([]string{"--hub", server.URL, "send", "--synthetic", "--agent", "claude_code", "--project", "test-p"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to run tt send: %v", err)
	}

	if !receivedBatch {
		t.Fatalf("expected server to receive ingestion batch")
	}
}

func TestCobraScanSweep(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptDir := filepath.Join(tmpDir, ".claude", "projects", "sample-project")
	if err := os.MkdirAll(transcriptDir, 0755); err != nil {
		t.Fatalf("failed to create sample transcript directory: %v", err)
	}

	sessionFile := filepath.Join(transcriptDir, "session.jsonl")
	content := `{"type":"user","timestamp":"2026-08-23T10:00:00Z","sessionId":"sess-cli-1"}
{"type":"assistant","timestamp":"2026-08-23T10:00:05Z","sessionId":"sess-cli-1","message":{"id":"m-1","model":"claude-3-7-sonnet-20250219","usage":{"input_tokens":5000,"output_tokens":800,"cache_read_input_tokens":2000,"cache_creation_input_tokens":0},"content":[{"type":"tool_use","name":"view_file"}]}}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write mock session: %v", err)
	}

	// 1. Dry run scan
	cmd := NewRootCmd()
	scanBuf := new(bytes.Buffer)
	cmd.SetOut(scanBuf)
	cmd.SetArgs([]string{"scan", "--dry-run", transcriptDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to execute tt scan --dry-run: %v", err)
	}

	// 2. JSON scan output
	cmd = NewRootCmd()
	jsonBuf := new(bytes.Buffer)
	cmd.SetOut(jsonBuf)
	cmd.SetArgs([]string{"scan", "--dry-run", "--json", transcriptDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to execute tt scan --json: %v", err)
	}
}

func TestCobraWatchGracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"watch", "--daemon", tmpDir})
	err := cmd.ExecuteContext(ctx)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("expected clean shutdown on context cancellation, got: %v", err)
	}
}

func TestCobraSessionsCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Setup sample Antigravity session
	agDir := filepath.Join(tmpDir, ".gemini", "antigravity-cli", "brain", "session-ag-1", ".system_generated", "logs")
	if err := os.MkdirAll(agDir, 0755); err != nil {
		t.Fatalf("failed to create antigravity dir: %v", err)
	}
	agContent := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","created_at":"2026-08-23T10:00:00Z","content":"<USER_SETTINGS_CHANGE>\nThe user changed setting ` + "`Model Selection`" + ` from None to Gemini 3.7 Flash (High).\n</USER_SETTINGS_CHANGE>"}
{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","created_at":"2026-08-23T10:00:05Z","tool_calls":[{"name":"view_file"}],"metrics":{"input_tokens":3000,"output_tokens":600,"cache_read_tokens":1000}}
`
	if err := os.WriteFile(filepath.Join(agDir, "transcript.jsonl"), []byte(agContent), 0644); err != nil {
		t.Fatalf("failed to write antigravity session: %v", err)
	}

	// 2. Test tt sessions --plain
	cmd := NewRootCmd()
	plainBuf := new(bytes.Buffer)
	cmd.SetOut(plainBuf)
	cmd.SetArgs([]string{"sessions", "--plain", "--harness", "antigravity", tmpDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to run tt sessions --plain: %v", err)
	}
	plainOut := plainBuf.String()
	if !strings.Contains(plainOut, "TokenTelemetry Discovered Sessions") {
		t.Fatalf("expected header in plain output, got: %s", plainOut)
	}
	if !strings.Contains(plainOut, "antigravity") {
		t.Fatalf("expected antigravity harness in plain output, got: %s", plainOut)
	}
	if !strings.Contains(plainOut, "gemini-3.7-flash") {
		t.Fatalf("expected gemini-3.7-flash in plain output, got: %s", plainOut)
	}

	// 3. Test tt sessions --json
	cmd = NewRootCmd()
	jsonBuf := new(bytes.Buffer)
	cmd.SetOut(jsonBuf)
	cmd.SetArgs([]string{"sessions", "--json", "--limit", "5", tmpDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to run tt sessions --json: %v", err)
	}

	var jsonSessions []*models.Session
	if err := json.Unmarshal(jsonBuf.Bytes(), &jsonSessions); err != nil {
		t.Fatalf("failed to decode JSON sessions output: %v, raw: %s", err, jsonBuf.String())
	}
	if len(jsonSessions) != 1 {
		t.Fatalf("expected 1 session in JSON output, got %d", len(jsonSessions))
	}
	if jsonSessions[0].ModelRaw != "gemini-3.7-flash" {
		t.Fatalf("expected model gemini-3.7-flash in JSON output, got %s", jsonSessions[0].ModelRaw)
	}
}

