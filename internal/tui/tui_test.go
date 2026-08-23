package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

func TestTUIModelInitialization(t *testing.T) {
	m := NewModel("http://localhost:8000", 5)

	if m.HubURL != "http://localhost:8000" {
		t.Fatalf("expected HubURL http://localhost:8000, got %s", m.HubURL)
	}
	if m.ActiveRoots != 5 {
		t.Fatalf("expected ActiveRoots 5, got %d", m.ActiveRoots)
	}
	if len(m.Rows) != 0 {
		t.Fatalf("expected 0 initial rows, got %d", len(m.Rows))
	}
	if m.Paused {
		t.Fatalf("expected initial model to not be paused")
	}
}

func TestTUIModelResizeAndResponsiveLayout(t *testing.T) {
	m := NewModel("http://localhost:8000", 3)

	// Narrow window
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 24})
	mNarrow := updated.(Model)
	if mNarrow.Width != 70 || mNarrow.Height != 24 {
		t.Fatalf("expected width 70 and height 24, got %dx%d", mNarrow.Width, mNarrow.Height)
	}
	viewNarrow := mNarrow.View()
	if !strings.Contains(viewNarrow, "THROUGHPUT") {
		t.Fatalf("expected narrow view to render KPI cards")
	}

	// Wide window
	updatedWide, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mWide := updatedWide.(Model)
	if mWide.Width != 140 || mWide.Height != 40 {
		t.Fatalf("expected width 140 and height 40, got %dx%d", mWide.Width, mWide.Height)
	}
	viewWide := mWide.View()
	if !strings.Contains(viewWide, "CACHE EFFICIENCY") || !strings.Contains(viewWide, "ESTIMATED COST") {
		t.Fatalf("expected wide view to render all KPI cards")
	}
}

func TestTUIModelTurnAndSessionIngestion(t *testing.T) {
	m := NewModel("http://localhost:8000", 4)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = updated.(Model)

	sess := &models.Session{
		ID:           "claude:sess-1",
		SessionID:    "sess-1",
		AgentName:    "claude_code",
		ProjectName:  "tokentelemetry",
		ModelRaw:     "claude-3-7-sonnet",
		GrossCostUSD: 0.1250,
		NetCostUSD:   0.0850,
		Turns:        make([]models.MessageTurn, 1),
	}

	turn := &models.MessageTurn{
		ID:                  "claude:sess-1:1",
		SessionID:           "claude:sess-1",
		TurnIndex:           1,
		Timestamp:           time.Now(),
		ModelName:           "claude-3-7-sonnet",
		InputTokens:         10000,
		OutputTokens:        2500,
		CacheReadTokens:     50000,
		CacheCreationTokens: 1000,
		CostUSD:             0.0850,
		ToolsInvoked:        []string{"view_file"},
	}

	// Ingest Turn
	updated, _ = m.Update(TurnIngestedMsg{Turn: turn, Session: sess})
	m = updated.(Model)

	if m.TotalTurns != 1 {
		t.Fatalf("expected 1 turn, got %d", m.TotalTurns)
	}
	if m.TotalInputTokens != 10000 {
		t.Fatalf("expected 10000 input tokens, got %d", m.TotalInputTokens)
	}
	if m.TotalOutputTokens != 2500 {
		t.Fatalf("expected 2500 output tokens, got %d", m.TotalOutputTokens)
	}
	if m.TotalCacheReadTokens != 50000 {
		t.Fatalf("expected 50000 cache read tokens, got %d", m.TotalCacheReadTokens)
	}
	if len(m.Rows) != 1 {
		t.Fatalf("expected 1 row in table, got %d", len(m.Rows))
	}

	// Check Hit Rate
	hitRate := m.CalculateCacheHitRate()
	if hitRate <= 0 || hitRate > 100 {
		t.Fatalf("expected valid hit rate percentage, got %.2f%%", hitRate)
	}

	// Ingest Session
	updated, _ = m.Update(SessionIngestedMsg{Session: sess})
	m = updated.(Model)
	if m.TotalSessions != 1 {
		t.Fatalf("expected 1 session, got %d", m.TotalSessions)
	}
	if m.TotalGrossCostUSD != 0.1250 {
		t.Fatalf("expected 0.1250 gross cost, got %f", m.TotalGrossCostUSD)
	}

	// Test Hub Health
	updated, _ = m.Update(HubHealthMsg{Status: "ONLINE", Latency: 2 * time.Millisecond})
	m = updated.(Model)
	if m.HubStatus != "ONLINE" {
		t.Fatalf("expected HubStatus ONLINE, got %s", m.HubStatus)
	}

	// Test Pause
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(Model)
	if !m.Paused {
		t.Fatalf("expected paused to be true after pressing 'p'")
	}

	// Ingest while paused should not modify counts
	updated, _ = m.Update(TurnIngestedMsg{Turn: turn, Session: sess})
	m = updated.(Model)
	if m.TotalTurns != 1 {
		t.Fatalf("expected 1 turn while paused, got %d", m.TotalTurns)
	}

	// Test Resume
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(Model)
	if m.Paused {
		t.Fatalf("expected paused to be false after pressing 'p' again")
	}

	// Test Clear
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(Model)
	if m.TotalTurns != 0 || len(m.Rows) != 0 {
		t.Fatalf("expected 0 turns and 0 rows after clear")
	}
}

func TestTUIErrorAndStatusHandling(t *testing.T) {
	m := NewModel("http://localhost:8000", 2)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)

	// Status Message
	updated, _ = m.Update(StatusUpdateMsg("Custom status message"))
	m = updated.(Model)
	if m.StatusMessage != "Custom status message" {
		t.Fatalf("expected status message 'Custom status message', got %s", m.StatusMessage)
	}
	if !strings.Contains(m.View(), "Custom status message") {
		t.Fatalf("expected view to contain status message")
	}

	// Error Message
	updated, _ = m.Update(ErrorMsg{Err: errors.New("network timeout")})
	m = updated.(Model)
	if m.ErrorMessage != "network timeout" {
		t.Fatalf("expected error message 'network timeout', got %s", m.ErrorMessage)
	}
	if !strings.Contains(m.View(), "network timeout") {
		t.Fatalf("expected view to render error message")
	}
}

func TestTUISinkThreadSafety(t *testing.T) {
	sink := NewTUISink()

	// Calling sink methods before program attachment should safely no-op
	sink.OnSession(&models.Session{ID: "test-sess"})
	sink.OnTurn(&models.MessageTurn{ID: "test-turn"}, nil)
	sink.OnStatus("test-status")
	sink.OnError(errors.New("test-err"))
	if err := sink.Close(); err != nil {
		t.Fatalf("expected clean close: %v", err)
	}
}
