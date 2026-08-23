package collector

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// EventSink provides an interface for receiving collector ingestion and lifecycle events.
type EventSink interface {
	OnSession(session *models.Session)
	OnTurn(turn *models.MessageTurn, session *models.Session)
	OnStatus(status string)
	OnError(err error)
	Close() error
}

// SlogSink is a structured slog-backed event sink for headless/daemon executions.
type SlogSink struct {
	logger *slog.Logger
	mu     sync.Mutex
}

// NewSlogSink creates an EventSink using Go's log/slog.
func NewSlogSink(out io.Writer, level slog.Level, jsonFormat bool) *SlogSink {
	if out == nil {
		out = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if jsonFormat {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}

	return &SlogSink{
		logger: slog.New(handler),
	}
}

// OnSession logs session-level ingestion events.
func (s *SlogSink) OnSession(session *models.Session) {
	if session == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("session ingested",
		"session_id", session.SessionID,
		"agent", session.AgentName,
		"project", session.ProjectName,
		"model", session.ModelRaw,
		"turns", len(session.Turns),
		"input_tokens", session.InputTokens,
		"output_tokens", session.OutputTokens,
		"cache_read_tokens", session.CacheReadTokens,
		"net_cost_usd", session.NetCostUSD,
	)
}

// OnTurn logs turn-level ingestion events.
func (s *SlogSink) OnTurn(turn *models.MessageTurn, session *models.Session) {
	if turn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	sessID := ""
	agentName := ""
	if session != nil {
		sessID = session.SessionID
		agentName = session.AgentName
	}

	s.logger.Debug("turn ingested",
		"session_id", sessID,
		"agent", agentName,
		"turn_index", turn.TurnIndex,
		"model", turn.ModelName,
		"input_tokens", turn.InputTokens,
		"output_tokens", turn.OutputTokens,
		"cache_read_tokens", turn.CacheReadTokens,
		"cost_usd", turn.CostUSD,
		"tools", turn.ToolsInvoked,
	)
}

// OnStatus logs status changes.
func (s *SlogSink) OnStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger.Info("collector status", "status", status)
}

// OnError logs runtime collector errors.
func (s *SlogSink) OnError(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger.Error("collector error", "error", err.Error())
}

// Close flushes the sink.
func (s *SlogSink) Close() error {
	return nil
}

// SilentSink is a no-op sink for tests or silent runs.
type SilentSink struct {
	Sessions []*models.Session
	Turns    []*models.MessageTurn
	Statuses []string
	Errors   []error
	mu       sync.Mutex
}

// NewSilentSink returns a memory-accumulating silent sink.
func NewSilentSink() *SilentSink {
	return &SilentSink{}
}

// OnSession appends session.
func (s *SilentSink) OnSession(session *models.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sessions = append(s.Sessions, session)
}

// OnTurn appends turn.
func (s *SilentSink) OnTurn(turn *models.MessageTurn, session *models.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Turns = append(s.Turns, turn)
}

// OnStatus appends status.
func (s *SilentSink) OnStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Statuses = append(s.Statuses, status)
}

// OnError appends error.
func (s *SilentSink) OnError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.Errors = append(s.Errors, err)
	}
}

// Close closes silent sink.
func (s *SilentSink) Close() error {
	return nil
}

// ConsoleSink writes human-readable real-time updates to standard output.
type ConsoleSink struct {
	out io.Writer
	mu  sync.Mutex
}

// NewConsoleSink creates a new human-readable console sink.
func NewConsoleSink(out io.Writer) *ConsoleSink {
	if out == nil {
		out = os.Stdout
	}
	return &ConsoleSink{out: out}
}

// OnSession writes session summary.
func (c *ConsoleSink) OnSession(session *models.Session) {
	if session == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(c.out, "[%s] Session: %-12s | %-16s | %-18s | Tokens: %d/%d (Cache: %d) | $%.4f\n",
		time.Now().Format("15:04:05"),
		session.AgentName,
		session.ProjectName,
		session.ModelRaw,
		session.InputTokens,
		session.OutputTokens,
		session.CacheReadTokens,
		session.NetCostUSD,
	)
}

// OnTurn writes turn summary if needed.
func (c *ConsoleSink) OnTurn(turn *models.MessageTurn, session *models.Session) {
	// Kept lightweight on console sink
}

// OnStatus writes status string.
func (c *ConsoleSink) OnStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(c.out, "[%s] %s\n", time.Now().Format("15:04:05"), status)
}

// OnError writes error.
func (c *ConsoleSink) OnError(err error) {
	if err == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(c.out, "[%s] ERROR: %v\n", time.Now().Format("15:04:05"), err)
}

// Close closes console sink.
func (c *ConsoleSink) Close() error {
	return nil
}
