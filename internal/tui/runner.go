package tui

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// TUISink bridges collector events into Bubble Tea via thread-safe Program.Send.
type TUISink struct {
	program *tea.Program
	mu      sync.RWMutex
}

// NewTUISink creates an EventSink connected to a Bubble Tea Program.
func NewTUISink() *TUISink {
	return &TUISink{}
}

// SetProgram attaches the active tea.Program to the sink.
func (s *TUISink) SetProgram(p *tea.Program) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.program = p
}

// OnSession delivers session messages to the TUI.
func (s *TUISink) OnSession(session *models.Session) {
	s.mu.RLock()
	p := s.program
	s.mu.RUnlock()

	if p != nil {
		p.Send(SessionIngestedMsg{Session: session})
	}
}

// OnTurn delivers turn messages to the TUI.
func (s *TUISink) OnTurn(turn *models.MessageTurn, session *models.Session) {
	s.mu.RLock()
	p := s.program
	s.mu.RUnlock()

	if p != nil {
		p.Send(TurnIngestedMsg{Turn: turn, Session: session})
	}
}

// OnStatus delivers status updates to the TUI.
func (s *TUISink) OnStatus(status string) {
	s.mu.RLock()
	p := s.program
	s.mu.RUnlock()

	if p != nil {
		p.Send(StatusUpdateMsg(status))
	}
}

// OnError delivers errors to the TUI.
func (s *TUISink) OnError(err error) {
	s.mu.RLock()
	p := s.program
	s.mu.RUnlock()

	if p != nil && err != nil {
		p.Send(ErrorMsg{Err: err})
	}
}

// Close terminates the sink.
func (s *TUISink) Close() error {
	return nil
}

// Run starts the interactive Bubble Tea terminal user interface.
func Run(ctx context.Context, cfg *collector.Config, pipeline *collector.Pipeline, tuiSink *TUISink) error {
	activeRoots := 0
	if cfg != nil {
		activeRoots = len(cfg.ScanRoots)
	}

	hubURL := "http://localhost:8000"
	if cfg != nil && cfg.HubURL != "" {
		hubURL = cfg.HubURL
	}

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	model := NewModel(hubURL, activeRoots)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(runCtx),
	)

	if tuiSink != nil {
		tuiSink.SetProgram(program)
	}

	// Launch background Hub health polling loop
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Initial check
		if health, err := pipeline.PingHub(runCtx); err == nil && health != nil {
			program.Send(HubHealthMsg{
				Status:  health.Status,
				Latency: health.Latency,
				Error:   health.Error,
			})
		}

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				health, err := pipeline.PingHub(runCtx)
				if err != nil {
					program.Send(HubHealthMsg{
						Status: "OFFLINE",
						Error:  err.Error(),
					})
				} else if health != nil {
					program.Send(HubHealthMsg{
						Status:  health.Status,
						Latency: health.Latency,
						Error:   health.Error,
					})
				}
			}
		}
	}()

	// Start collector pipeline
	if err := pipeline.Start(runCtx); err != nil {
		return err
	}

	// Run Bubble Tea program (blocks until quit or context cancellation)
	_, err := program.Run()

	// Cancel context to immediately stop background goroutines and health loops
	runCancel()

	// Stop collector pipeline on exit
	_ = pipeline.Stop()

	return err
}

// RunSessionsBrowser starts the interactive Bubble Tea terminal user interface pre-loaded with discovered sessions.
func RunSessionsBrowser(ctx context.Context, cfg *collector.Config, pipeline *collector.Pipeline, initialSessions []*models.Session, harnessFilter string) error {
	activeRoots := 0
	if cfg != nil {
		activeRoots = len(cfg.ScanRoots)
	}

	hubURL := "http://localhost:8000"
	if cfg != nil && cfg.HubURL != "" {
		hubURL = cfg.HubURL
	}

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	model := NewSessionsModel(hubURL, activeRoots, initialSessions, harnessFilter)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(runCtx),
	)

	// Launch background Hub health polling loop
	if pipeline != nil {
		go func() {
			if health, err := pipeline.PingHub(runCtx); err == nil && health != nil {
				program.Send(HubHealthMsg{
					Status:  health.Status,
					Latency: health.Latency,
					Error:   health.Error,
				})
			}
		}()
	}

	// Run Bubble Tea program (blocks until quit or context cancellation)
	_, err := program.Run()
	return err
}

