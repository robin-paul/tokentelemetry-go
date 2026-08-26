package api

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
