package e2e_test

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// getFreePort binds to port 0 to allocate a free ephemeral TCP port.
func getFreePort(t *testing.T) int {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to resolve TCP addr: %v", err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("failed to listen on ephemeral port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// buildBinary compiles the static CGO-free binary if not already present or out of date.
func buildBinary(t *testing.T, binPath string) {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	cmd := exec.Command("go", "build", "-ldflags=-s -w -X main.Version=1.0.0-e2e -X main.Commit=e2etest", "-o", binPath, "./cmd/tokentelemetry")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to compile tokentelemetry binary: %v\nOutput: %s", err, string(out))
	}
}

// TestEndToEndIntegration runs a complete black-box verification of the compiled binary.
func TestEndToEndIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "tokentelemetry_e2e_*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binPath := filepath.Join(tmpDir, "tokentelemetry")
	dbPath := filepath.Join(tmpDir, "e2e_tokentelemetry.db")
	logsRoot := filepath.Join(tmpDir, "logs")

	// 1. Build Single Static CGO-Free Binary
	buildBinary(t, binPath)

	// Verify binary exists and reports version
	versionCmd := exec.Command(binPath, "--version")
	verOut, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute --version: %v\nOutput: %s", err, string(verOut))
	}
	if !strings.Contains(string(verOut), "tokentelemetry version 1.0.0-e2e") {
		t.Fatalf("unexpected version output: %s", string(verOut))
	}

	// 2. Prepare Sample Agent Transcripts (Claude Code & Antigravity)
	claudeDir := filepath.Join(logsRoot, ".claude", "projects", "ecommerce-app")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create claude log dir: %v", err)
	}

	antigravityDir := filepath.Join(logsRoot, ".gemini", "antigravity-cli", "brain", "task_01", ".system_generated", "logs")
	if err := os.MkdirAll(antigravityDir, 0755); err != nil {
		t.Fatalf("failed to create antigravity log dir: %v", err)
	}

	claudeSessionFile := filepath.Join(claudeDir, "sess_e2e_01.jsonl")
	claudeTranscript := `{"type":"user","sessionId":"claude-sess-e2e","timestamp":"2026-06-15T10:00:00Z","message":{"content":"Implement user checkout"}}
{"type":"assistant","sessionId":"claude-sess-e2e","timestamp":"2026-06-15T10:00:05Z","message":{"model":"claude-3-7-sonnet","usage":{"input_tokens":4000,"output_tokens":850,"cache_read_input_tokens":1500,"cache_creation_input_tokens":500},"content":[{"type":"tool_use","name":"view_file","input":{"path":"checkout.go"}}]}}
`
	if err := os.WriteFile(claudeSessionFile, []byte(claudeTranscript), 0644); err != nil {
		t.Fatalf("failed to write sample claude session: %v", err)
	}

	// 3. Launch the Server Subprocess
	port := getFreePort(t)
	serverCmd := exec.Command(binPath,
		"-port", fmt.Sprintf("%d", port),
		"-db", dbPath,
		"-scan-dir", logsRoot,
		"-auth-token", "e2e-secret-token",
	)

	serverOut, err := os.Create(filepath.Join(tmpDir, "server.log"))
	if err != nil {
		t.Fatalf("failed to create server log: %v", err)
	}
	defer serverOut.Close()
	serverCmd.Stdout = serverOut
	serverCmd.Stderr = serverOut

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("failed to start server subprocess: %v", err)
	}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- serverCmd.Wait()
	}()

	// Ensure cleanup on test exit
	defer func() {
		if serverCmd.Process != nil {
			_ = serverCmd.Process.Signal(syscall.SIGTERM)
			select {
			case <-serverDone:
			case <-time.After(3 * time.Second):
				_ = serverCmd.Process.Kill()
			}
		}
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 5 * time.Second}

	// 4. Poll Health Endpoint until Ready
	var ready bool
	for i := 0; i < 40; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := client.Get(baseURL + "/healthz")
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			ready = true
			break
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}
	if !ready {
		t.Fatalf("server failed to become healthy within 4s. See logs at %s", filepath.Join(tmpDir, "server.log"))
	}

	// 5. Verify Static Embedded Astro Web Interface & Client Island Routes
	t.Run("WebAssets_and_SPA_Fallback", func(t *testing.T) {
		// A. Root index.html
		resp, err := client.Get(baseURL + "/")
		if err != nil {
			t.Fatalf("GET / failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for /, got %d", resp.StatusCode)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		if !strings.Contains(bodyStr, "TokenTelemetry") || !strings.Contains(bodyStr, "<!DOCTYPE html>") {
			t.Errorf("expected Astro index HTML, got: %s", bodyStr[:min(len(bodyStr), 300)])
		}

		// B. Analytics Page
		resp, err = client.Get(baseURL + "/analytics")
		if err != nil {
			t.Fatalf("GET /analytics failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for /analytics, got %d", resp.StatusCode)
		}

		// C. Dynamic SPA Fallback Route (/sessions/:id)
		resp, err = client.Get(baseURL + "/sessions/claude-sess-e2e")
		if err != nil {
			t.Fatalf("GET /sessions/claude-sess-e2e failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for SPA route, got %d", resp.StatusCode)
		}
	})

	// 6. Connect Real-time SSE Stream Subscriber
	sseCtx, cancelSSE := context.WithCancel(context.Background())
	defer cancelSSE()

	sseReq, err := http.NewRequestWithContext(sseCtx, "GET", baseURL+"/api/events", nil)
	if err != nil {
		t.Fatalf("failed to create SSE request: %v", err)
	}
	sseResp, err := http.DefaultClient.Do(sseReq)
	if err != nil {
		t.Fatalf("failed to connect SSE stream: %v", err)
	}
	defer sseResp.Body.Close()

	if sseResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for SSE stream, got %d", sseResp.StatusCode)
	}
	if !strings.Contains(sseResp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %s", sseResp.Header.Get("Content-Type"))
	}

	var (
		sseEventsMu sync.Mutex
		sseEvents   []string
	)
	go func() {
		reader := bufio.NewReader(sseResp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:") {
				sseEventsMu.Lock()
				sseEvents = append(sseEvents, line)
				sseEventsMu.Unlock()
			}
		}
	}()

	// 7. Write a new agent transcript to trigger live fsnotify watcher & reconciler
	agFile := filepath.Join(antigravityDir, "transcript.jsonl")
	agTranscript := `{"type":"USER_INPUT","created_at":"2026-06-15T11:00:00Z","content":"Optimize database query"}
{"type":"PLANNER_RESPONSE","created_at":"2026-06-15T11:00:05Z","content":"Analyzing indexes","thinking":"Checking SQLite WAL pragmas","tool_calls":[{"name":"view_file","arguments":{"path":"db.go"}}],"token_usage":{"prompt_tokens":2500,"completion_tokens":400,"total_tokens":2900,"cached_tokens":1000}}
`
	if err := os.WriteFile(agFile, []byte(agTranscript), 0644); err != nil {
		t.Fatalf("failed to write antigravity transcript: %v", err)
	}

	// 8. Wait for Scanner and Ingestion to Process Transcripts
	var sessionsList []map[string]interface{}
	var ingested bool
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := client.Get(baseURL + "/api/sessions")
		if err == nil && resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			var arr []map[string]interface{}
			if err := json.Unmarshal(body, &arr); err == nil && len(arr) >= 2 {
				sessionsList = arr
				ingested = true
				break
			}

			var result struct {
				Sessions []map[string]interface{} `json:"sessions"`
			}
			if err := json.Unmarshal(body, &result); err == nil && len(result.Sessions) >= 2 {
				sessionsList = result.Sessions
				ingested = true
				break
			}
		}
	}
	if !ingested {
		logBytes, _ := os.ReadFile(filepath.Join(tmpDir, "server.log"))
		t.Fatalf("timed out waiting for 2 sessions to be scanned. Got: %v\nServer log:\n%s", sessionsList, string(logBytes))
	}

	// 9. Verify REST API Endpoints & Pricing Data
	t.Run("REST_API_Verification", func(t *testing.T) {
		// A. /api/sessions list
		if len(sessionsList) < 2 {
			t.Fatalf("expected at least 2 sessions, got %d", len(sessionsList))
		}

		var claudeFound, agFound bool
		for _, s := range sessionsList {
			agentName := s["agent_name"].(string)
			grossCost := s["gross_cost_usd"].(float64)
			inputTokens := s["input_tokens"].(float64)
			outputTokens := s["output_tokens"].(float64)

			if agentName == "claude_code" {
				claudeFound = true
				if inputTokens != 4000 || outputTokens != 850 {
					t.Errorf("claude tokens mismatch: input=%f, output=%f", inputTokens, outputTokens)
				}
				if grossCost <= 0 {
					t.Errorf("expected positive gross cost for Claude, got %f", grossCost)
				}
			}
			if agentName == "antigravity" {
				agFound = true
				if inputTokens < 2500 || outputTokens != 400 {
					t.Errorf("antigravity tokens mismatch: input=%f, output=%f", inputTokens, outputTokens)
				}
			}
		}
		if !claudeFound || !agFound {
			t.Errorf("missing expected agent session in results: claude=%v, ag=%v", claudeFound, agFound)
		}

		// B. /api/stats
		resp, err := client.Get(baseURL + "/api/stats")
		if err != nil {
			t.Fatalf("GET /api/stats failed: %v", err)
		}
		defer resp.Body.Close()
		var stats map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			t.Fatalf("failed to decode stats: %v", err)
		}
		if totalSessions, ok := stats["total_sessions"].(float64); !ok || totalSessions < 2 {
			t.Errorf("expected total_sessions >= 2, got %v", stats["total_sessions"])
		}

		// C. /api/leaderboard
		resp, err = client.Get(baseURL + "/api/leaderboard")
		if err != nil {
			t.Fatalf("GET /api/leaderboard failed: %v", err)
		}
		defer resp.Body.Close()
		var leaderboard struct {
			Models []map[string]interface{} `json:"models"`
			Agents []map[string]interface{} `json:"agents"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&leaderboard); err != nil {
			t.Fatalf("failed to decode leaderboard: %v", err)
		}
		if len(leaderboard.Models) == 0 && len(leaderboard.Agents) == 0 {
			t.Errorf("expected non-empty leaderboard")
		}

		// D. /api/pricing
		resp, err = client.Get(baseURL + "/api/pricing")
		if err != nil {
			t.Fatalf("GET /api/pricing failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for /api/pricing, got %d", resp.StatusCode)
		}
	})

	// 10. Verify Real-time SSE Events Emitted
	t.Run("SSE_Broadcast_Verification", func(t *testing.T) {
		time.Sleep(300 * time.Millisecond)
		sseEventsMu.Lock()
		eventsCopy := make([]string, len(sseEvents))
		copy(eventsCopy, sseEvents)
		sseEventsMu.Unlock()

		var hasSessionEvent bool
		for _, ev := range eventsCopy {
			if strings.Contains(ev, "session.created") || strings.Contains(ev, "session.updated") ||
				strings.Contains(ev, "session.new") || strings.Contains(ev, "session.update") {
				hasSessionEvent = true
				break
			}
		}
		if !hasSessionEvent {
			t.Errorf("expected session event in SSE stream, got events: %v", eventsCopy)
		}
	})

	// 11. Verify Pure-Go SQLite Database On Disk
	t.Run("SQLite_WAL_Persistence_Verification", func(t *testing.T) {
		db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)")
		if err != nil {
			t.Fatalf("failed to open SQLite db directly: %v", err)
		}
		defer db.Close()

		var journalMode string
		err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
		if err != nil {
			t.Fatalf("failed to query journal_mode: %v", err)
		}
		if !strings.EqualFold(journalMode, "wal") {
			t.Errorf("expected journal_mode WAL, got %s", journalMode)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sessions;").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query sessions count: %v", err)
		}
		if count < 2 {
			t.Errorf("expected at least 2 sessions in SQLite, got %d", count)
		}

		var turnsCount int
		err = db.QueryRow("SELECT COUNT(*) FROM message_turns;").Scan(&turnsCount)
		if err != nil {
			t.Fatalf("failed to query message_turns count: %v", err)
		}
		if turnsCount < 2 {
			t.Errorf("expected at least 2 message turns in SQLite, got %d", turnsCount)
		}

		var checkpointsCount int
		err = db.QueryRow("SELECT COUNT(*) FROM scanner_checkpoints;").Scan(&checkpointsCount)
		if err != nil {
			t.Fatalf("failed to query scanner_checkpoints count: %v", err)
		}
		if checkpointsCount < 2 {
			t.Errorf("expected at least 2 checkpoints in SQLite, got %d", checkpointsCount)
		}
	})

	// 12. Verify Clean Graceful Shutdown on SIGTERM
	t.Run("Graceful_Shutdown", func(t *testing.T) {
		cancelSSE()
		_ = sseResp.Body.Close()
		if serverCmd.Process != nil {
			_ = serverCmd.Process.Signal(syscall.SIGTERM)
			select {
			case err := <-serverDone:
				if err != nil && !strings.Contains(err.Error(), "signal: terminated") && !strings.Contains(err.Error(), "exit status 0") {
					t.Logf("server exited with: %v", err)
				}
			case <-time.After(4 * time.Second):
				t.Errorf("server failed to shutdown gracefully within 4s")
				_ = serverCmd.Process.Kill()
			}
		}
	})
}
