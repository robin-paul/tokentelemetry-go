package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/events"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

func newLocalRequest(method, url string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.RemoteAddr = "127.0.0.1:54321"
	return req
}

func setupTestServer(t *testing.T) (*Server, *store.DB, http.Handler, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	pe, err := pricing.NewEngine()
	if err != nil {
		t.Fatalf("failed to init pricing engine: %v", err)
	}

	broker := events.NewBroker(100 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)

	cfg := Config{
		AuthToken:      "secret-token-123",
		AllowedOrigins: []string{"http://localhost:3000"},
		Version:        "1.2.3",
		Commit:         "testcommit",
	}

	srv := NewServer(db, pe, nil, broker, cfg)
	router := srv.Router()

	cleanup := func() {
		cancel()
		broker.Stop()
		_ = db.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return srv, db, router, cleanup
}

func seedTestData(t *testing.T, db *store.DB) {
	t.Helper()
	ctx := context.Background()

	childSess := &models.Session{
		ID:              "child-sess-1",
		SessionID:       "child-sess-1",
		AgentName:       "research",
		ProjectName:     "test-project",
		FilePath:        "/path/to/child.json",
		StartTime:       time.Now().Add(-30 * time.Minute).UTC(),
		EndTime:         time.Now().UTC(),
		DurationSeconds: 1800,
		ModelRaw:        "claude-sonnet-4-6",
		ModelResolved:   "claude-sonnet-4-6",
		InputTokens:     1500,
		OutputTokens:    500,
		Status:          "completed",
		IsSubagent:      true,
		ParentSessionID: "sess-test-1",
		SubagentType:    "research",
	}
	if err := db.SaveSessionWithTurnsAndSubagents(ctx, childSess); err != nil {
		t.Fatalf("failed to seed child session: %v", err)
	}

	sess := &models.Session{
		ID:                  "sess-test-1",
		SessionID:           "sess-test-1",
		AgentName:           "claude",
		ProjectName:         "test-project",
		FilePath:            "/path/to/session.json",
		StartTime:           time.Now().Add(-1 * time.Hour).UTC(),
		EndTime:             time.Now().UTC(),
		DurationSeconds:     3600,
		ModelRaw:            "claude-sonnet-4-6",
		ModelResolved:       "claude-sonnet-4-6",
		InputTokens:         10000,
		OutputTokens:        2000,
		CacheReadTokens:     5000,
		CacheCreationTokens: 1000,
		GrossCostUSD:        0.06,
		NetCostUSD:          0.045,
		ElectricityCostUSD:  0.0,
		Status:              "completed",
		Turns: []models.MessageTurn{
			{
				ID:           "turn-1",
				SessionID:    "sess-test-1",
				TurnIndex:    0,
				Role:         "user",
				InputTokens:  5000,
				OutputTokens: 0,
			},
			{
				ID:           "turn-2",
				SessionID:    "sess-test-1",
				TurnIndex:    1,
				Role:         "assistant",
				ModelName:    "claude-sonnet-4-6",
				InputTokens:  5000,
				OutputTokens: 2000,
				ToolsInvoked: []string{"view_file", "run_command"},
			},
		},
		SubagentRuns: []models.SubagentRun{
			{
				ID:              "sub-run-1",
				ParentSessionID: "sess-test-1",
				ChildSessionID:  "child-sess-1",
				AgentType:       "research",
				Tokens:          1500,
				CostUSD:         0.005,
				CreatedAt:       time.Now().UTC(),
			},
		},
	}

	if err := db.SaveSessionWithTurnsAndSubagents(ctx, sess); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}

	dateStr := sess.StartTime.Format("2006-01-02")
	if err := db.RollupDailySummariesForDate(ctx, dateStr); err != nil {
		t.Fatalf("failed to rollup summaries: %v", err)
	}
}

func TestHealthAndSystemEndpoints(t *testing.T) {
	_, _, router, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. GET /healthz
	req := newLocalRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var health map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &health)
	if health["status"] != "ok" || health["version"] != "1.2.3" {
		t.Errorf("unexpected healthz response: %v", health)
	}

	// 2. GET /
	req = newLocalRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// 3. GET /version
	req = newLocalRequest("GET", "/version", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// 4. GET /agents
	req = newLocalRequest("GET", "/agents", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var agents []string
	_ = json.Unmarshal(w.Body.Bytes(), &agents)
	if len(agents) < 10 {
		t.Errorf("expected at least 10 agents, got %d", len(agents))
	}

	// 5. GET /remote-access (Loopback caller)
	req = newLocalRequest("GET", "/remote-access", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestRemoteAuthMiddleware(t *testing.T) {
	_, _, router, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Loopback request without token -> Allowed
	req := newLocalRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected loopback request to be allowed without token, got %d", w.Code)
	}

	// 2. Non-loopback request without token -> 401 Unauthorized
	req = httptest.NewRequest("GET", "/api/sessions", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected non-loopback request to be unauthorized, got %d", w.Code)
	}

	// 3. Non-loopback request with invalid Bearer token -> 401 Unauthorized
	req = httptest.NewRequest("GET", "/api/sessions", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid token to be unauthorized, got %d", w.Code)
	}

	// 4. Non-loopback request with valid Bearer token -> 200 OK
	req = httptest.NewRequest("GET", "/api/sessions", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	req.Header.Set("Authorization", "Bearer secret-token-123")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected valid Bearer token to be allowed, got %d", w.Code)
	}

	// 5. Non-loopback request with valid ?token= query parameter -> 200 OK
	req = httptest.NewRequest("GET", "/api/sessions?token=secret-token-123", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected valid query param token to be allowed, got %d", w.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	_, _, router, cleanup := setupTestServer(t)
	defer cleanup()

	req := newLocalRequest("OPTIONS", "/api/sessions", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 for OPTIONS preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("unexpected CORS Allow-Origin header: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestSessionsEndpoints(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	seedTestData(t, db)

	// 1. GET /api/sessions
	req := newLocalRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var sessions []models.Session
	_ = json.Unmarshal(w.Body.Bytes(), &sessions)
	if len(sessions) < 1 {
		t.Fatalf("expected at least 1 session, got %d", len(sessions))
	}

	// 2. GET /api/sessions/{id}
	req = newLocalRequest("GET", "/api/sessions/sess-test-1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var detail models.Session
	_ = json.Unmarshal(w.Body.Bytes(), &detail)
	if len(detail.Turns) != 2 {
		t.Errorf("expected 2 turns, got %d", len(detail.Turns))
	}
	if len(detail.SubagentRuns) != 1 {
		t.Errorf("expected 1 subagent run, got %d", len(detail.SubagentRuns))
	}

	// 3. GET /api/recent
	req = newLocalRequest("GET", "/api/recent", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 4. GET /sessions/{id}/delegation
	req = newLocalRequest("GET", "/sessions/sess-test-1/delegation", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 5. GET /sessions/{id}/grok-forensics
	req = newLocalRequest("GET", "/sessions/sess-test-1/grok-forensics", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 6. DELETE /api/sessions/{id}
	req = newLocalRequest("DELETE", "/api/sessions/sess-test-1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify deleted
	req = newLocalRequest("GET", "/api/sessions/sess-test-1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

func TestStatsAndAnalyticsEndpoints(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	seedTestData(t, db)

	// 1. GET /api/stats
	req := newLocalRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var stats models.StatsOverview
	_ = json.Unmarshal(w.Body.Bytes(), &stats)
	if stats.TotalSessions < 1 {
		t.Errorf("expected at least 1 total session, got %d", stats.TotalSessions)
	}

	// 2. GET /api/stats/daily
	req = newLocalRequest("GET", "/api/stats/daily", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 3. GET /api/leaderboard
	req = newLocalRequest("GET", "/api/leaderboard", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 4. GET /analytics
	req = newLocalRequest("GET", "/analytics", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestProjectsAndAliasesEndpoints(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	seedTestData(t, db)

	// 1. GET /api/projects
	req := newLocalRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var projects []models.ProjectSummary
	_ = json.Unmarshal(w.Body.Bytes(), &projects)
	if len(projects) != 1 || projects[0].ProjectName != "test-project" {
		t.Errorf("unexpected projects response: %v", projects)
	}

	// 2. GET /api/projects/test-project
	req = newLocalRequest("GET", "/api/projects/test-project", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 3. POST /config/hide & GET /config/hidden & POST /config/unhide
	hideBody := bytes.NewBufferString(`{"path":"test-project"}`)
	req = newLocalRequest("POST", "/config/hide", hideBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on hide, got %d", w.Code)
	}

	req = newLocalRequest("GET", "/config/hidden", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	unhideBody := bytes.NewBufferString(`{"path":"test-project"}`)
	req = newLocalRequest("POST", "/config/unhide", unhideBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on unhide, got %d", w.Code)
	}

	// 4. POST /config/aliases & GET /config/aliases
	aliasBody := bytes.NewBufferString(`{"/old/path":"/new/path"}`)
	req = newLocalRequest("POST", "/config/aliases", aliasBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on set aliases, got %d", w.Code)
	}

	req = newLocalRequest("GET", "/config/aliases", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestPricingEndpoints(t *testing.T) {
	_, _, router, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. GET /api/pricing
	req := newLocalRequest("GET", "/api/pricing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. POST /api/pricing/override
	overrideJSON := `{"model_pattern":"custom-model","input_cost_per_m":2.5,"output_cost_per_m":10.0,"cache_read_cost_per_m":0.25,"cache_write_cost_per_m":3.0}`
	req = newLocalRequest("POST", "/api/pricing/override", bytes.NewBufferString(overrideJSON))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 3. DELETE /api/pricing/override/{pattern}
	req = newLocalRequest("DELETE", "/api/pricing/override/custom-model", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestBudgetsAndNotificationsEndpoints(t *testing.T) {
	_, _, router, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. PUT /budgets & GET /budgets
	budgetJSON := `{"budgets":[{"id":"b1","period":"monthly","limit_type":"usd","limit_value":100.0,"enabled":true}]}`
	req := newLocalRequest("PUT", "/budgets", bytes.NewBufferString(budgetJSON))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	req = newLocalRequest("GET", "/budgets", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. GET /notifications and action endpoints
	req = newLocalRequest("GET", "/notifications", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	req = newLocalRequest("POST", "/notifications/toasted", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	req = newLocalRequest("POST", "/notifications/read", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	req = newLocalRequest("POST", "/notifications/clear", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestConfigEndpoints(t *testing.T) {
	_, _, router, cleanup := setupTestServer(t)
	defer cleanup()

	endpoints := []string{
		"/config",
		"/config/update-check",
		"/config/telemetry",
		"/config/telemetry/preview",
		"/config/retention",
		"/config/power",
		"/config/power/meter",
		"/config/billing",
		"/config/billing-route",
		"/config/agent-features",
		"/summarizer/available",
		"/config/summarizer",
		"/summarizer/ollama/models",
		"/summarizer/codex/models",
		"/dsh/lifecycle",
		"/cache/status",
	}

	for _, ep := range endpoints {
		req := newLocalRequest("GET", ep, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("GET %s failed with status %d: %s", ep, w.Code, w.Body.String())
		}
	}
}

func TestArtifactEndpoint(t *testing.T) {
	_, _, router, cleanup := setupTestServer(t)
	defer cleanup()

	tmpFile, err := os.CreateTemp("", "artifact-test-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp artifact: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := "Sample artifact content"
	_, _ = tmpFile.WriteString(content)
	_ = tmpFile.Close()

	t.Run("ValidArtifact", func(t *testing.T) {
		req := newLocalRequest("GET", fmt.Sprintf("/artifacts?path=%s", tmpFile.Name()), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for artifact, got %d: %s", w.Code, w.Body.String())
		}
		if w.Body.String() != content {
			t.Errorf("unexpected artifact content: %s", w.Body.String())
		}
	})

	t.Run("MissingPath", func(t *testing.T) {
		req := newLocalRequest("GET", "/artifacts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for missing path, got %d", w.Code)
		}
	})

	t.Run("RelativePath", func(t *testing.T) {
		req := newLocalRequest("GET", "/artifacts?path=some/relative/path.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for relative path, got %d", w.Code)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := newLocalRequest("GET", fmt.Sprintf("/artifacts?path=%s/nonexistent_file.txt", os.TempDir()), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for nonexistent artifact, got %d", w.Code)
		}
	})
}

func TestMultiCriteriaSessionSearchAPI(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	baseTime := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

	// Seed 3 sessions with various models, costs, and tools
	s1 := &models.Session{
		ID:              "sess-search-1",
		SessionID:       "sess-search-1",
		AgentName:       "claude_code",
		ProjectName:     "token-analyzer",
		FilePath:        "/tmp/s1.json",
		StartTime:       baseTime.Add(-2 * time.Hour),
		EndTime:         baseTime.Add(-1 * time.Hour),
		DurationSeconds: 3600,
		ModelRaw:        "claude-3-7-sonnet",
		ModelResolved:   "claude-3-7-sonnet",
		InputTokens:     15000,
		OutputTokens:    3000,
		NetCostUSD:      0.09,
		GitBranch:       "feature/auth",
		Status:          "completed",
		Turns: []models.MessageTurn{
			{ID: "t1-1", SessionID: "sess-search-1", TurnIndex: 0, ToolsInvoked: []string{"search_web"}},
		},
	}
	s2 := &models.Session{
		ID:              "sess-search-2",
		SessionID:       "sess-search-2",
		AgentName:       "antigravity",
		ProjectName:     "frontend-app",
		FilePath:        "/tmp/s2.json",
		StartTime:       baseTime.Add(-1 * time.Hour),
		EndTime:         baseTime,
		DurationSeconds: 1800,
		ModelRaw:        "gemini-2.5-flash",
		ModelResolved:   "gemini-2.5-flash",
		InputTokens:     40000,
		OutputTokens:    8000,
		NetCostUSD:      0.02,
		GitBranch:       "main",
		Status:          "completed",
		Turns: []models.MessageTurn{
			{ID: "t2-1", SessionID: "sess-search-2", TurnIndex: 0, ToolsInvoked: []string{"run_command"}},
		},
	}
	s3 := &models.Session{
		ID:              "sess-search-3",
		SessionID:       "sess-search-3",
		AgentName:       "cursor",
		ProjectName:     "frontend-app",
		FilePath:        "/tmp/s3.json",
		StartTime:       baseTime,
		EndTime:         baseTime.Add(30 * time.Minute),
		DurationSeconds: 1800,
		ModelRaw:        "gpt-4o",
		ModelResolved:   "gpt-4o",
		InputTokens:     5000,
		OutputTokens:    1000,
		NetCostUSD:      0.03,
		GitBranch:       "fix/css",
		Status:          "error",
	}

	for _, s := range []*models.Session{s1, s2, s3} {
		if err := db.SaveSessionWithTurnsAndSubagents(ctx, s); err != nil {
			t.Fatalf("failed to seed session %s: %v", s.ID, err)
		}
	}

	// 1. Search with q parameter (FTS)
	{
		req := newLocalRequest("GET", "/api/sessions?q=token-analyzer", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("search request failed: %d", w.Code)
		}
		var list []models.Session
		_ = json.NewDecoder(w.Body).Decode(&list)
		if len(list) != 1 || list[0].ID != "sess-search-1" {
			t.Errorf("expected 1 session matching token-analyzer, got %d", len(list))
		}
	}

	// 2. Multi-value agent filter
	{
		req := newLocalRequest("GET", "/api/sessions?agent=antigravity,cursor", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("multi-agent request failed: %d", w.Code)
		}
		var list []models.Session
		_ = json.NewDecoder(w.Body).Decode(&list)
		if len(list) != 2 {
			t.Errorf("expected 2 sessions for antigravity,cursor, got %d", len(list))
		}
	}

	// 3. Min/Max Cost filter
	{
		req := newLocalRequest("GET", "/api/sessions?min_cost=0.05", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("min_cost request failed: %d", w.Code)
		}
		var list []models.Session
		_ = json.NewDecoder(w.Body).Decode(&list)
		if len(list) != 1 || list[0].ID != "sess-search-1" {
			t.Errorf("expected 1 session with cost >= 0.05, got %d", len(list))
		}
	}

	// 4. Paginated envelope format
	{
		req := newLocalRequest("GET", "/api/sessions?format=paginated&limit=2&page=1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("paginated request failed: %d", w.Code)
		}
		var resp struct {
			Sessions   []models.Session `json:"sessions"`
			Pagination struct {
				Page       int   `json:"page"`
				PageSize   int   `json:"page_size"`
				Total      int64 `json:"total"`
				TotalPages int   `json:"total_pages"`
			} `json:"pagination"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Sessions) != 2 || resp.Pagination.Total != 3 || resp.Pagination.TotalPages != 2 {
			t.Errorf("unexpected paginated response: total=%d, pages=%d, returned=%d",
				resp.Pagination.Total, resp.Pagination.TotalPages, len(resp.Sessions))
		}
	}

	// 5. Sorting by tokens asc
	{
		req := newLocalRequest("GET", "/api/sessions?sort_by=tokens&sort_order=asc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("sorting request failed: %d", w.Code)
		}
		var list []models.Session
		_ = json.NewDecoder(w.Body).Decode(&list)
		if len(list) != 3 || list[0].ID != "sess-search-3" || list[2].ID != "sess-search-2" {
			t.Errorf("unexpected token sort order: %v", list)
		}
	}
}

func TestGetSessionDetailRichTurns(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	sessID := "sess-rich-turns-1"

	sess := &models.Session{
		ID:              sessID,
		SessionID:       sessID,
		AgentName:       "claude_code",
		ProjectName:     "token-analyzer",
		FilePath:        "/tmp/rich_turns.jsonl",
		StartTime:       now.Add(-10 * time.Minute),
		EndTime:         now,
		DurationSeconds: 600,
		ModelRaw:        "claude-3-7-sonnet",
		ModelResolved:   "claude-3-7-sonnet",
		InputTokens:     5000,
		OutputTokens:    1200,
		NetCostUSD:      0.03,
		Status:          "completed",
		Turns: []models.MessageTurn{
			{
				ID:             fmt.Sprintf("%s:1", sessID),
				SessionID:      sessID,
				TurnIndex:      0,
				Timestamp:      now.Add(-9 * time.Minute),
				Role:           "user",
				ModelName:      "claude-3-7-sonnet",
				Content:        "Can you implement the rich markdown renderer?",
				RawPayloadJSON: `{"role":"user","content":"Can you implement the rich markdown renderer?"}`,
			},
			{
				ID:              fmt.Sprintf("%s:2", sessID),
				SessionID:       sessID,
				TurnIndex:       1,
				Timestamp:       now.Add(-8 * time.Minute),
				Role:            "assistant",
				ModelName:       "claude-3-7-sonnet",
				Content:         "```typescript\nconst a = 1;\n```\nHere is the implementation.",
				Thinking:        "We will use react-markdown with remark-gfm.",
				ReasoningEffort: "high",
				InputTokens:     5000,
				OutputTokens:    1200,
				ToolsInvoked:    []string{"write_to_file"},
				ToolCalls: []models.ToolCall{
					{
						ID:   "call_1",
						Name: "write_to_file",
						Args: map[string]interface{}{"TargetFile": "/path/to/ResponseBody.tsx"},
					},
				},
				ToolResults: []models.ToolResult{
					{
						ID:      "call_1",
						Name:    "write_to_file",
						Content: "File written successfully",
					},
				},
				RawPayloadJSON: `{"role":"assistant","content":"Done"}`,
			},
		},
	}

	if err := db.SaveSessionWithTurnsAndSubagents(ctx, sess); err != nil {
		t.Fatalf("failed to save rich session: %v", err)
	}

	req := newLocalRequest("GET", "/api/sessions/"+sessID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var fetched models.Session
	if err := json.NewDecoder(w.Body).Decode(&fetched); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(fetched.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(fetched.Turns))
	}
	if fetched.Turns[0].Content != "Can you implement the rich markdown renderer?" {
		t.Errorf("unexpected turn 0 content: %q", fetched.Turns[0].Content)
	}
	if fetched.Turns[1].Thinking != "We will use react-markdown with remark-gfm." || fetched.Turns[1].ReasoningEffort != "high" {
		t.Errorf("unexpected turn 1 thinking/effort: %q / %q", fetched.Turns[1].Thinking, fetched.Turns[1].ReasoningEffort)
	}
	if len(fetched.Turns[1].ToolCalls) != 1 || fetched.Turns[1].ToolCalls[0].Name != "write_to_file" {
		t.Errorf("unexpected turn 1 tool calls: %+v", fetched.Turns[1].ToolCalls)
	}
	if len(fetched.Turns[1].ToolResults) != 1 || fetched.Turns[1].ToolResults[0].Content != "File written successfully" {
		t.Errorf("unexpected turn 1 tool results: %+v", fetched.Turns[1].ToolResults)
	}
}

func TestDSHLifecycleAPI(t *testing.T) {
	ctx := context.Background()
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Absent file returns installed: false
	t.Setenv("TOKENTELEMETRY_DATA_DIR", t.TempDir())
	req := newLocalRequest("GET", "/dsh/lifecycle", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["installed"] != false {
		t.Errorf("expected installed: false, got %v", res["installed"])
	}
	if res["correlation"] != "none" {
		t.Errorf("expected correlation: none, got %v", res["correlation"])
	}

	// 2. With sidecar file
	tmpDataDir := t.TempDir()
	t.Setenv("TOKENTELEMETRY_DATA_DIR", tmpDataDir)
	logFile := filepath.Join(tmpDataDir, "dsh_lifecycle.jsonl")

	logContent := `{"ts":1000,"plugin":"cordis-plugin-a","from":0,"to":1}
{"ts":2000,"plugin":"cordis-plugin-a","from":1,"to":2}
{"ts":3000,"plugin":"cordis-plugin-b","from":1,"to":3,"error":"failed to init"}
`
	if err := os.WriteFile(logFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write sidecar file: %v", err)
	}

	req = newLocalRequest("GET", "/api/dsh/lifecycle", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resPresent models.DSHLifecycleSummary
	if err := json.NewDecoder(w.Body).Decode(&resPresent); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resPresent.Installed {
		t.Errorf("expected installed true")
	}
	if resPresent.Transitions != 3 {
		t.Errorf("expected 3 transitions, got %d", resPresent.Transitions)
	}
	if resPresent.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", resPresent.Failed)
	}

	// 3. With session_id correlation
	sessID := "dsh:test-lifecycle-sess"
	sess := &models.Session{
		ID:          sessID,
		SessionID:   "test-lifecycle-sess",
		AgentName:   "dsh",
		ProjectName: "lifecycle-proj",
		StartTime:   time.UnixMilli(500),
		EndTime:     time.UnixMilli(2500),
		Status:      "completed",
		DSH: &models.DSHContext{
			AgentPreset: "cordis",
		},
	}
	if err := db.SaveSessionWithTurnsAndSubagents(ctx, sess); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	req = newLocalRequest("GET", "/dsh/lifecycle?session_id="+sessID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resCorrelated models.DSHLifecycleSummary
	if err := json.NewDecoder(w.Body).Decode(&resCorrelated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resCorrelated.Correlation != "time-window" {
		t.Errorf("expected correlation time-window, got %s", resCorrelated.Correlation)
	}
	// ts: 1000 and 2000 are inside [500, 2500]; ts: 3000 is outside
	if resCorrelated.Transitions != 2 {
		t.Errorf("expected 2 transitions inside window, got %d", resCorrelated.Transitions)
	}

	// 4. Session not found returns 404
	req = newLocalRequest("GET", "/dsh/lifecycle?session_id=nonexistent", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for nonexistent session, got %d", w.Code)
	}

	// 5. GET /api/sessions/{id} attaches DSH lifecycle
	req = newLocalRequest("GET", "/api/sessions/"+sessID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var fetched models.Session
	if err := json.NewDecoder(w.Body).Decode(&fetched); err != nil {
		t.Fatalf("failed to decode session: %v", err)
	}
	if fetched.DSH == nil || fetched.DSH.Lifecycle == nil {
		t.Fatalf("expected attached DSH lifecycle")
	}
	if fetched.DSH.Lifecycle.Transitions != 2 {
		t.Errorf("expected 2 correlated transitions on session, got %d", fetched.DSH.Lifecycle.Transitions)
	}
}

func TestDSHSurfacingAcrossAgentsTraceAndDelegation(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()

	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "session.jsonl")

	sessionJsonl := fmt.Sprintf(`{"type":"session","id":"parent-101","time":%d,"agentPreset":"cordis"}
{"type":"request/context","time":%d,"data":{"provider":"cerebras","model":"zai-glm-4.7"}}
{"type":"request/header","time":%d,"data":{"header":{"tools":[{"name":"bash"},{"name":"web_search"}]}}}
{"type":"user/message","time":%d,"data":{"role":"user","source":{"kind":"skill-catalog","entries":[{"name":"code-review","description":"Review code"}]},"content":[{"type":"text","text":"catalog"}]}}
{"type":"user/message","time":%d,"data":{"role":"user","source":{"kind":"user"},"content":[{"type":"text","text":"Please review the code"}]}}
{"type":"assistant/message","time":%d,"data":{"turn":1,"step":1,"content":[{"type":"reasoning","text":"Thinking deeply..."},{"type":"text","text":"I have reviewed it."}],"usage":{"inputTokens":5000,"outputTokens":2000}}}
{"type":"tool/call","time":%d,"data":{"turn":1,"step":1,"callId":"c1","name":"bash","arguments":"{\"cmd\":\"git diff\"}"}}
{"type":"tool/result","time":%d,"data":{"turn":1,"step":1,"callId":"c1","content":[{"type":"text","text":"+ diff content"}]}}
`, now.Add(-10*time.Minute).UnixMilli(),
		now.Add(-9*time.Minute).UnixMilli(),
		now.Add(-9*time.Minute).UnixMilli(),
		now.Add(-8*time.Minute).UnixMilli(),
		now.Add(-7*time.Minute).UnixMilli(),
		now.Add(-6*time.Minute).UnixMilli(),
		now.Add(-5*time.Minute).UnixMilli(),
		now.Add(-4*time.Minute).UnixMilli())

	if err := os.WriteFile(sessionFile, []byte(sessionJsonl), 0644); err != nil {
		t.Fatalf("failed to write session file: %v", err)
	}

	// 1. Seed a parent DSH session and a child subagent DSH session
	parentID := "dsh:parent-101"
	childID := "dsh:child-202"

	parentSess := &models.Session{
		ID:           parentID,
		SessionID:    "parent-101",
		AgentName:    "dsh",
		ProjectName:  "dsh-project",
		FilePath:     sessionFile,
		StartTime:    now.Add(-10 * time.Minute),
		EndTime:      now,
		Status:       "completed",
		InputTokens:  5000,
		OutputTokens: 2000,
		NetCostUSD:   0.05,
		Turns: []models.MessageTurn{
			{
				ID:        "turn-1",
				SessionID: parentID,
				TurnIndex: 1,
				Role:      "user",
				Content:   "Please review the code",
			},
			{
				ID:              "turn-2",
				SessionID:       parentID,
				TurnIndex:       2,
				Role:            "assistant",
				Content:         "I have reviewed it.",
				Thinking:        "Thinking deeply...",
				ReasoningEffort: "high",
				InputTokens:     5000,
				OutputTokens:    2000,
				ToolsInvoked:    []string{"bash"},
				ToolCalls: []models.ToolCall{
					{ID: "c1", Name: "bash", ArgsJSON: `{"cmd":"git diff"}`},
				},
				ToolResults: []models.ToolResult{
					{ID: "c1", Content: "+ diff content"},
				},
			},
		},
	}
	if err := db.SaveSessionWithTurnsAndSubagents(ctx, parentSess); err != nil {
		t.Fatalf("failed to insert parent: %v", err)
	}

	childFile := filepath.Join(tmpDir, "child.jsonl")
	childJsonl := fmt.Sprintf(`{"type":"session","id":"child-202","time":%d,"origin":"subagent","parentSession":"%s"}
{"type":"sandbox/mode","time":%d,"data":{"mode":"read-only","source":"delegation"}}
{"type":"approval/policy","time":%d,"data":{"policy":"prompt","source":"delegation"}}
{"type":"assistant/message","time":%d,"data":{"turn":1,"step":1,"content":[{"type":"text","text":"child response"}],"usage":{"inputTokens":1200,"outputTokens":400}}}
`, now.Add(-5*time.Minute).UnixMilli(), parentID,
		now.Add(-5*time.Minute).UnixMilli(),
		now.Add(-5*time.Minute).UnixMilli(),
		now.Add(-4*time.Minute).UnixMilli())

	if err := os.WriteFile(childFile, []byte(childJsonl), 0644); err != nil {
		t.Fatalf("failed to write child session file: %v", err)
	}

	childSess := &models.Session{
		ID:              childID,
		SessionID:       "child-202",
		AgentName:       "dsh",
		ProjectName:     "dsh-project",
		FilePath:        childFile,
		StartTime:       now.Add(-5 * time.Minute),
		EndTime:         now.Add(-2 * time.Minute),
		Status:          "completed",
		IsSubagent:      true,
		ParentSessionID: parentID,
		SubagentType:    "dsh-subagent",
		InputTokens:     1200,
		OutputTokens:    400,
		NetCostUSD:      0.015,
	}
	if err := db.SaveSessionWithTurnsAndSubagents(ctx, childSess); err != nil {
		t.Fatalf("failed to insert child: %v", err)
	}

	// Verify 1: GET /agents includes "dsh"
	req := newLocalRequest("GET", "/agents", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 from /agents, got %d", w.Code)
	}
	var agents []string
	if err := json.NewDecoder(w.Body).Decode(&agents); err != nil {
		t.Fatalf("failed to decode agents: %v", err)
	}
	foundDSH := false
	for _, a := range agents {
		if a == "dsh" {
			foundDSH = true
			break
		}
	}
	if !foundDSH {
		t.Errorf("expected DSH in /agents list: %+v", agents)
	}

	// Verify 2: GET /api/sessions/{id} returns trace normalization and runtime capabilities
	req = newLocalRequest("GET", "/api/sessions/"+parentID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 from /api/sessions/%s, got %d", parentID, w.Code)
	}
	var fetchedParent models.Session
	if err := json.NewDecoder(w.Body).Decode(&fetchedParent); err != nil {
		t.Fatalf("failed to decode parent session: %v", err)
	}
	if len(fetchedParent.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(fetchedParent.Turns))
	}
	if fetchedParent.Turns[1].Thinking != "Thinking deeply..." || fetchedParent.Turns[1].ReasoningEffort != "high" {
		t.Errorf("expected thinking in turn 2, got %s", fetchedParent.Turns[1].Thinking)
	}
	if len(fetchedParent.Turns[1].ToolCalls) != 1 || fetchedParent.Turns[1].ToolCalls[0].Name != "bash" {
		t.Errorf("expected tool call in turn 2, got %+v", fetchedParent.Turns[1].ToolCalls)
	}
	if fetchedParent.DSH == nil || len(fetchedParent.DSH.SkillsCatalog) != 1 || fetchedParent.DSH.SkillsCatalog[0].Name != "code-review" {
		t.Errorf("expected runtime skills catalog, got %+v", fetchedParent.DSH)
	}

	// Verify 3: GET /api/sessions/{id}/delegation returns subagent runs and posture
	req = newLocalRequest("GET", "/api/sessions/"+parentID+"/delegation", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 from delegation, got %d", w.Code)
	}
	var delegationResp struct {
		Supported  bool                 `json:"supported"`
		SpawnCount int                  `json:"spawn_count"`
		Subagents  []models.SubagentRun `json:"subagents"`
		Totals     struct {
			Tokens int64   `json:"tokens"`
			Cost   float64 `json:"cost"`
		} `json:"totals"`
	}
	if err := json.NewDecoder(w.Body).Decode(&delegationResp); err != nil {
		t.Fatalf("failed to decode delegation response: %v", err)
	}
	if len(delegationResp.Subagents) != 1 {
		t.Fatalf("expected 1 subagent run, got %d", len(delegationResp.Subagents))
	}
	if delegationResp.Subagents[0].ChildSessionID != childID {
		t.Errorf("expected child session ID %s, got %s", childID, delegationResp.Subagents[0].ChildSessionID)
	}
	if delegationResp.Subagents[0].Tokens != 1600 {
		t.Errorf("expected 1600 tokens (1200+400), got %d", delegationResp.Subagents[0].Tokens)
	}
	if delegationResp.Subagents[0].CostUSD != 0.015 {
		t.Errorf("expected cost 0.015, got %f", delegationResp.Subagents[0].CostUSD)
	}
	if delegationResp.Subagents[0].Sandbox == nil || delegationResp.Subagents[0].Sandbox["mode"] != "read-only" {
		t.Errorf("expected inherited sandbox posture on subagent run, got %+v", delegationResp.Subagents[0].Sandbox)
	}
}




