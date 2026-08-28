package api

import (
	"bytes"
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

func TestCanonicalRepoAndWorktreeDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git", "worktrees", "feature-branch"), 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	wtDir := filepath.Join(tmpDir, "feature-worktree")
	if err := os.MkdirAll(wtDir, 0755); err != nil {
		t.Fatalf("failed to create wt dir: %v", err)
	}

	gitFileContent := "gitdir: " + filepath.Join(repoDir, ".git", "worktrees", "feature-branch") + "\n"
	if err := os.WriteFile(filepath.Join(wtDir, ".git"), []byte(gitFileContent), 0644); err != nil {
		t.Fatalf("failed to write .git file: %v", err)
	}

	// Also write gitdir in repo .git/worktrees/feature-branch
	if err := os.WriteFile(filepath.Join(repoDir, ".git", "worktrees", "feature-branch", "gitdir"), []byte(filepath.Join(wtDir, ".git")+"\n"), 0644); err != nil {
		t.Fatalf("failed to write gitdir file: %v", err)
	}

	resolvedRepo := canonicalRepo(wtDir)
	if resolvedRepo != repoDir {
		t.Errorf("expected canonical repo %q, got %q", repoDir, resolvedRepo)
	}

	// Test fallback path regex for .claude/worktrees/feature-x
	claudeWtPath := filepath.Join(repoDir, ".claude", "worktrees", "feature-x")
	resolvedClaudeWt := canonicalRepo(claudeWtPath)
	if resolvedClaudeWt != repoDir {
		t.Errorf("expected claude worktree canonical repo %q, got %q", repoDir, resolvedClaudeWt)
	}
}

func TestEnrichProjectListWorktrees(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "enrich-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "my-repo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git", "worktrees", "wt-1"), 0755)

	wt1Dir := filepath.Join(tmpDir, "my-repo-wt-1")
	_ = os.MkdirAll(wt1Dir, 0755)
	_ = os.WriteFile(filepath.Join(wt1Dir, ".git"), []byte("gitdir: "+filepath.Join(repoDir, ".git", "worktrees", "wt-1")+"\n"), 0644)

	raw := []models.ProjectSummary{
		{
			ProjectName:   repoDir,
			SessionCount:  5,
			TotalTokens:   50000,
			TotalCostUSD:  0.50,
			Agents:        []string{"claude"},
			SubagentCount: 2,
			PlanCount:     1,
		},
		{
			ProjectName:   wt1Dir,
			SessionCount:  3,
			TotalTokens:   30000,
			TotalCostUSD:  0.30,
			Agents:        []string{"cursor"},
			SubagentCount: 1,
			PlanCount:     0,
		},
	}

	enriched := enrichProjectList(raw)
	if len(enriched) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(enriched))
	}

	var parent *models.ProjectSummary
	var child *models.ProjectSummary
	for i := range enriched {
		if enriched[i].Path == repoDir {
			parent = &enriched[i]
		} else if enriched[i].Path == wt1Dir {
			child = &enriched[i]
		}
	}

	if parent == nil || !parent.IsRepoRoot {
		t.Fatalf("expected parent to be repo root")
	}
	if len(parent.Worktrees) != 1 {
		t.Errorf("expected 1 worktree under parent, got %d", len(parent.Worktrees))
	}
	if parent.Aggregate == nil {
		t.Fatalf("expected aggregate on parent")
	}
	if parent.Aggregate.SessionCount != 8 {
		t.Errorf("expected aggregate session count 8, got %d", parent.Aggregate.SessionCount)
	}
	if parent.Aggregate.TotalTokens != 80000 {
		t.Errorf("expected aggregate tokens 80000, got %d", parent.Aggregate.TotalTokens)
	}
	if parent.Aggregate.TotalCostUSD < 0.79 || parent.Aggregate.TotalCostUSD > 0.81 {
		t.Errorf("expected aggregate cost 0.80, got %f", parent.Aggregate.TotalCostUSD)
	}

	if child == nil || !child.IsWorktree {
		t.Fatalf("expected child to be marked as worktree")
	}
	if child.ParentPath != repoDir {
		t.Errorf("expected child parent path %q, got %q", repoDir, child.ParentPath)
	}
}

func TestProjectAPIWithWorktrees(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "api-wt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "acme-repo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git", "worktrees", "wt-a"), 0755)

	wtDir := filepath.Join(tmpDir, "acme-repo-wt-a")
	_ = os.MkdirAll(wtDir, 0755)
	_ = os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: "+filepath.Join(repoDir, ".git", "worktrees", "wt-a")+"\n"), 0644)

	// Ingest sessions for root and worktree
	sess1 := models.Session{
		ID:           "s-root-1",
		SessionID:    "s-root-1",
		AgentName:    "claude",
		ProjectName:  repoDir,
		FilePath:     filepath.Join(repoDir, "session.jsonl"),
		StartTime:    time.Now().Add(-1 * time.Hour),
		EndTime:      time.Now(),
		InputTokens:  1000,
		OutputTokens: 200,
		NetCostUSD:   0.05,
		Status:       "completed",
	}
	sess2 := models.Session{
		ID:           "s-wt-1",
		SessionID:    "s-wt-1",
		AgentName:    "cursor",
		ProjectName:  wtDir,
		FilePath:     filepath.Join(wtDir, "session.jsonl"),
		StartTime:    time.Now().Add(-30 * time.Minute),
		EndTime:      time.Now(),
		InputTokens:  2000,
		OutputTokens: 400,
		NetCostUSD:   0.10,
		Status:       "completed",
	}

	_ = db.UpsertSession(context.Background(), &sess1)
	_ = db.UpsertSession(context.Background(), &sess2)

	// 1. GET /api/projects
	req := newLocalRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var projects []models.ProjectSummary
	_ = json.Unmarshal(w.Body.Bytes(), &projects)
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d: %v", len(projects), projects)
	}

	// 2. GET /api/projects/{name}
	req = newLocalRequest("GET", "/api/projects/"+filepath.Base(repoDir), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var detailResp struct {
		Project  models.ProjectSummary `json:"project"`
		Sessions []models.Session      `json:"sessions"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &detailResp)
	if detailResp.Project.ProjectName != repoDir {
		t.Errorf("expected project %q, got %q", repoDir, detailResp.Project.ProjectName)
	}
	if !detailResp.Project.IsRepoRoot {
		t.Errorf("expected project to be repo root")
	}
	if len(detailResp.Project.Worktrees) != 1 {
		t.Errorf("expected 1 worktree in detail, got %d", len(detailResp.Project.Worktrees))
	}
}

func TestSeparatorVariantsYieldSingleProjectCard(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Seed 2 sessions from the same project folder with different separator styles
	sess1 := models.Session{
		ID:           "s-win-back",
		SessionID:    "s-win-back",
		AgentName:    "claude",
		ProjectName:  `C:\Users\dev\myproject`,
		FilePath:     `C:\Users\dev\myproject\log1.json`,
		StartTime:    time.Now().Add(-1 * time.Hour),
		EndTime:      time.Now(),
		InputTokens:  1000,
		OutputTokens: 200,
		NetCostUSD:   0.05,
		Status:       "completed",
	}
	sess2 := models.Session{
		ID:           "s-win-fwd",
		SessionID:    "s-win-fwd",
		AgentName:    "claude",
		ProjectName:  "C:/Users/dev/myproject/",
		FilePath:     "C:/Users/dev/myproject/log2.json",
		StartTime:    time.Now().Add(-30 * time.Minute),
		EndTime:      time.Now(),
		InputTokens:  2000,
		OutputTokens: 400,
		NetCostUSD:   0.10,
		Status:       "completed",
	}

	if err := db.SaveSessionWithTurnsAndSubagents(ctx, &sess1); err != nil {
		t.Fatalf("failed to save sess1: %v", err)
	}
	if err := db.SaveSessionWithTurnsAndSubagents(ctx, &sess2); err != nil {
		t.Fatalf("failed to save sess2: %v", err)
	}

	// GET /api/projects must return exactly 1 card
	req := newLocalRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var projects []models.ProjectSummary
	_ = json.Unmarshal(w.Body.Bytes(), &projects)
	if len(projects) != 1 {
		t.Fatalf("expected exactly 1 project card for separator variants, got %d: %+v", len(projects), projects)
	}

	card := projects[0]
	if card.ProjectName != "C:/Users/dev/myproject" {
		t.Errorf("expected canonical card path 'C:/Users/dev/myproject', got %q", card.ProjectName)
	}
	if card.SessionCount != 2 {
		t.Errorf("expected session_count 2, got %d", card.SessionCount)
	}
	if card.TotalTokens != 3600 {
		t.Errorf("expected total tokens 3600, got %d", card.TotalTokens)
	}
}

func TestHiddenProjectAcrossSeparatorStyles(t *testing.T) {
	_, db, router, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	sess := models.Session{
		ID:           "s-win-hide",
		SessionID:    "s-win-hide",
		AgentName:    "claude",
		ProjectName:  "C:/Users/dev/myproject",
		FilePath:     "C:/Users/dev/myproject/log.json",
		StartTime:    time.Now().Add(-1 * time.Hour),
		EndTime:      time.Now(),
		InputTokens:  1000,
		OutputTokens: 200,
		NetCostUSD:   0.05,
		Status:       "completed",
	}
	if err := db.SaveSessionWithTurnsAndSubagents(ctx, &sess); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	// 1. Hide using Windows backslash form
	hideBody := `{"path":"C:\\Users\\dev\\myproject\\"}`
	req := newLocalRequest("POST", "/config/hide", bytes.NewBufferString(hideBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("hide failed with %d: %s", w.Code, w.Body.String())
	}

	// 2. Verify GET /config/hidden has canonical form
	req = newLocalRequest("GET", "/config/hidden", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var hidden []string
	_ = json.Unmarshal(w.Body.Bytes(), &hidden)
	if len(hidden) != 1 || hidden[0] != "C:/Users/dev/myproject" {
		t.Fatalf("expected hidden to contain 'C:/Users/dev/myproject', got %v", hidden)
	}

	// 3. Verify GET /api/projects excludes the card
	req = newLocalRequest("GET", "/api/projects", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var visible []models.ProjectSummary
	_ = json.Unmarshal(w.Body.Bytes(), &visible)
	if len(visible) != 0 {
		t.Errorf("expected project to be hidden, but got %d projects", len(visible))
	}

	// 4. Verify GET /api/projects?include_hidden=true includes the card
	req = newLocalRequest("GET", "/api/projects?include_hidden=true", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var allProjects []models.ProjectSummary
	_ = json.Unmarshal(w.Body.Bytes(), &allProjects)
	if len(allProjects) != 1 {
		t.Errorf("expected 1 project with include_hidden=true, got %d", len(allProjects))
	}

	// 5. Unhide using forward-slash form
	unhideBody := `{"path":"C:/Users/dev/myproject"}`
	req = newLocalRequest("POST", "/config/unhide", bytes.NewBufferString(unhideBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unhide failed with %d: %s", w.Code, w.Body.String())
	}

	// 6. Verify GET /config/hidden is now empty
	req = newLocalRequest("GET", "/config/hidden", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	hidden = nil
	_ = json.Unmarshal(w.Body.Bytes(), &hidden)
	if len(hidden) != 0 {
		t.Errorf("expected hidden to be empty after unhide, got %v", hidden)
	}

	// 7. Verify GET /api/projects includes the card again
	req = newLocalRequest("GET", "/api/projects", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	visible = nil
	_ = json.Unmarshal(w.Body.Bytes(), &visible)
	if len(visible) != 1 {
		t.Errorf("expected project to be visible again, got %d", len(visible))
	}
}

