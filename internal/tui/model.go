package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// ViewMode represents the active screen view in the TUI.
type ViewMode int

const (
	ViewModeLive ViewMode = iota
	ViewModeSessions
)

// Message types for Bubble Tea event loop
type TurnIngestedMsg struct {
	Turn    *models.MessageTurn
	Session *models.Session
}

type SessionIngestedMsg struct {
	Session *models.Session
}

type StatusUpdateMsg string

type HubHealthMsg struct {
	Status  string
	Latency time.Duration
	Error   string
}

type ErrorMsg struct {
	Err error
}

type TickMsg time.Time

// TokenSample stores timestamped token count for rolling throughput calculations.
type TokenSample struct {
	Timestamp time.Time
	Tokens    int64
}

// Model represents the state machine for the interactive Bubble Tea terminal dashboard.
type Model struct {
	Width                    int
	Height                   int
	ViewMode                 ViewMode
	Table                    table.Model
	Rows                     []table.Row
	SessionTable             table.Model
	SessionRows              []table.Row
	Sessions                 []*models.Session
	FilteredSessions         []*models.Session
	HarnessFilter            string
	ExpandedInspector        bool
	TotalInputTokens         int64
	TotalOutputTokens        int64
	TotalCacheReadTokens     int64
	TotalCacheCreationTokens int64
	TotalNetCostUSD          float64
	TotalGrossCostUSD        float64
	TotalTurns               int
	TotalSessions            int
	RecentSamples            []TokenSample
	HubURL                   string
	HubStatus                string
	HubLatency               time.Duration
	ActiveRoots              int
	Paused                   bool
	StartTime                time.Time
	StatusMessage            string
	ErrorMessage             string
}

// NewModel constructs an initialized Bubble Tea Model.
func NewModel(hubURL string, activeRoots int) Model {
	columns := []table.Column{
		{Title: "TIME", Width: 10},
		{Title: "AGENT", Width: 14},
		{Title: "PROJECT", Width: 16},
		{Title: "MODEL", Width: 22},
		{Title: "IN / OUT / CACHE", Width: 24},
		{Title: "COST (USD)", Width: 12},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipglossNormalBorder()).
		BorderForeground(ColorMuted).
		BorderBottom(true).
		Bold(true).
		Foreground(ColorSecondary)
	s.Selected = s.Selected.
		Foreground(ColorFgBright).
		Background(ColorPrimary).
		Bold(false)
	t.SetStyles(s)

	sessionCols := []table.Column{
		{Title: "TIME", Width: 10},
		{Title: "HARNESS", Width: 14},
		{Title: "SESSION ID", Width: 18},
		{Title: "MODEL", Width: 22},
		{Title: "TURNS", Width: 8},
		{Title: "IN / OUT / CACHE", Width: 24},
		{Title: "COST (USD)", Width: 12},
	}

	st := table.New(
		table.WithColumns(sessionCols),
		table.WithFocused(true),
		table.WithHeight(8),
	)
	st.SetStyles(s)

	return Model{
		ViewMode:          ViewModeLive,
		Table:             t,
		Rows:              make([]table.Row, 0),
		SessionTable:      st,
		SessionRows:       make([]table.Row, 0),
		Sessions:          make([]*models.Session, 0),
		FilteredSessions:  make([]*models.Session, 0),
		RecentSamples:     make([]TokenSample, 0),
		HubURL:            hubURL,
		HubStatus:         "CHECKING...",
		ActiveRoots:       activeRoots,
		StartTime:         time.Now(),
		StatusMessage:     "Listening for agent transcript updates... ([Tab]/[s] for Sessions)",
		ExpandedInspector: false,
	}
}

// NewSessionsModel constructs a Bubble Tea Model initialized in Sessions mode with preloaded sessions.
func NewSessionsModel(hubURL string, activeRoots int, initialSessions []*models.Session, harnessFilter string) Model {
	m := NewModel(hubURL, activeRoots)
	m.ViewMode = ViewModeSessions
	m.HarnessFilter = harnessFilter
	m.Sessions = initialSessions

	for _, s := range initialSessions {
		m.TotalSessions++
		m.TotalGrossCostUSD += s.GrossCostUSD
		m.TotalNetCostUSD += s.NetCostUSD
		m.TotalInputTokens += s.InputTokens
		m.TotalOutputTokens += s.OutputTokens
		m.TotalCacheReadTokens += s.CacheReadTokens
		m.TotalCacheCreationTokens += s.CacheCreationTokens
		m.TotalTurns += len(s.Turns)
	}

	m.applyHarnessFilter()
	m.StatusMessage = fmt.Sprintf("Loaded %d sessions. Use [↑/↓] to select, [Enter] to inspect, [h] to filter harness.", len(m.FilteredSessions))
	return m
}

// Init starts the 1-second background tick timer.
func (m Model) Init() tea.Cmd {
	return tickCmd()
}

// Update handles incoming messages and state changes.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.recalculateLayout()

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC, msg.String() == "ctrl+c", msg.String() == "q":
			return m, tea.Quit

		case msg.String() == "tab", msg.String() == "s":
			if m.ViewMode == ViewModeLive {
				m.ViewMode = ViewModeSessions
				m.applyHarnessFilter()
				m.StatusMessage = "Switched to Recent Sessions view (press [Tab]/[s] for Live Turns, [h] to filter harness, [Enter] to toggle inspector)"
			} else {
				m.ViewMode = ViewModeLive
				m.StatusMessage = "Switched to Live Turns feed (press [Tab]/[s] for Sessions)"
			}
			m.recalculateLayout()
			return m, nil

		case msg.String() == "h" && m.ViewMode == ViewModeSessions:
			m.cycleHarnessFilter()
			return m, nil

		case (msg.String() == "enter" || msg.Type == tea.KeyEnter) && m.ViewMode == ViewModeSessions:
			m.ExpandedInspector = !m.ExpandedInspector
			if m.ExpandedInspector {
				m.StatusMessage = "Expanded session inspector pane"
			} else {
				m.StatusMessage = "Collapsed session inspector pane"
			}
			m.recalculateLayout()
			return m, nil

		case msg.String() == "c":
			m.Rows = nil
			m.Table.SetRows(m.Rows)
			m.SessionRows = nil
			m.Sessions = nil
			m.FilteredSessions = nil
			m.SessionTable.SetRows(m.SessionRows)
			m.TotalInputTokens = 0
			m.TotalOutputTokens = 0
			m.TotalCacheReadTokens = 0
			m.TotalCacheCreationTokens = 0
			m.TotalNetCostUSD = 0
			m.TotalGrossCostUSD = 0
			m.TotalTurns = 0
			m.TotalSessions = 0
			m.RecentSamples = nil
			m.StatusMessage = "Cleared telemetry metrics and session feed"

		case msg.String() == "p":
			m.Paused = !m.Paused
			if m.Paused {
				m.StatusMessage = "Stream paused (events buffered)"
			} else {
				m.StatusMessage = "Stream resumed"
			}
		}

	case TurnIngestedMsg:
		if m.Paused {
			return m, nil
		}
		if msg.Turn != nil {
			m.TotalTurns++
			m.TotalInputTokens += msg.Turn.InputTokens
			m.TotalOutputTokens += msg.Turn.OutputTokens
			m.TotalCacheReadTokens += msg.Turn.CacheReadTokens
			m.TotalCacheCreationTokens += msg.Turn.CacheCreationTokens
			m.TotalNetCostUSD += msg.Turn.CostUSD

			turnTokens := msg.Turn.InputTokens + msg.Turn.OutputTokens + msg.Turn.CacheReadTokens
			m.RecentSamples = append(m.RecentSamples, TokenSample{
				Timestamp: time.Now(),
				Tokens:    turnTokens,
			})

			agentName := "unknown"
			projectName := "default"
			if msg.Session != nil {
				agentName = msg.Session.AgentName
				projectName = msg.Session.ProjectName
			}

			modelName := msg.Turn.ModelName
			if modelName == "" && msg.Session != nil {
				modelName = msg.Session.ModelRaw
			}

			timeStr := msg.Turn.Timestamp.Format("15:04:05")
			if timeStr == "00:00:00" {
				timeStr = time.Now().Format("15:04:05")
			}

			tokensCol := fmt.Sprintf("%s / %s / %s",
				humanize.Comma(msg.Turn.InputTokens),
				humanize.Comma(msg.Turn.OutputTokens),
				humanize.Comma(msg.Turn.CacheReadTokens),
			)

			costCol := fmt.Sprintf("$%.4f", msg.Turn.CostUSD)

			row := table.Row{
				timeStr,
				agentName,
				projectName,
				modelName,
				tokensCol,
				costCol,
			}

			// Prepend newest to top (keep last 200 rows)
			m.Rows = append([]table.Row{row}, m.Rows...)
			if len(m.Rows) > 200 {
				m.Rows = m.Rows[:200]
			}
			m.Table.SetRows(m.Rows)

			// Update session list if session is attached
			if msg.Session != nil {
				m.upsertSession(msg.Session, msg.Turn)
			}
		}

	case SessionIngestedMsg:
		if !m.Paused && msg.Session != nil {
			m.TotalSessions++
			m.TotalGrossCostUSD += msg.Session.GrossCostUSD
			m.StatusMessage = fmt.Sprintf("Ingested %s session (%s): %d turns",
				msg.Session.AgentName, msg.Session.ProjectName, len(msg.Session.Turns))
			m.upsertSession(msg.Session, nil)
		}

	case HubHealthMsg:
		m.HubStatus = msg.Status
		m.HubLatency = msg.Latency
		if msg.Error != "" {
			m.ErrorMessage = msg.Error
		} else {
			m.ErrorMessage = ""
		}

	case StatusUpdateMsg:
		m.StatusMessage = string(msg)

	case ErrorMsg:
		if msg.Err != nil {
			m.ErrorMessage = msg.Err.Error()
		}

	case TickMsg:
		// Prune samples older than 10 seconds for rolling throughput calculation
		cutoff := time.Now().Add(-10 * time.Second)
		var valid []TokenSample
		for _, s := range m.RecentSamples {
			if s.Timestamp.After(cutoff) {
				valid = append(valid, s)
			}
		}
		m.RecentSamples = valid
		return m, tickCmd()
	}

	if m.ViewMode == ViewModeSessions {
		m.SessionTable, cmd = m.SessionTable.Update(msg)
	} else {
		m.Table, cmd = m.Table.Update(msg)
	}
	return m, cmd
}

// upsertSession updates or appends a session to m.Sessions and refreshes filtered view.
func (m *Model) upsertSession(s *models.Session, turn *models.MessageTurn) {
	if s == nil {
		return
	}

	foundIdx := -1
	for i, existing := range m.Sessions {
		if existing.ID == s.ID || (existing.SessionID != "" && existing.SessionID == s.SessionID) {
			foundIdx = i
			break
		}
	}

	if foundIdx >= 0 {
		existing := m.Sessions[foundIdx]
		existing.InputTokens = s.InputTokens
		existing.OutputTokens = s.OutputTokens
		existing.CacheReadTokens = s.CacheReadTokens
		existing.CacheCreationTokens = s.CacheCreationTokens
		existing.GrossCostUSD = s.GrossCostUSD
		existing.NetCostUSD = s.NetCostUSD
		existing.ModelRaw = s.ModelRaw
		existing.ModelResolved = s.ModelResolved
		if len(s.Turns) > len(existing.Turns) {
			existing.Turns = s.Turns
		} else if turn != nil {
			alreadyPresent := false
			for _, t := range existing.Turns {
				if t.TurnIndex == turn.TurnIndex {
					alreadyPresent = true
					break
				}
			}
			if !alreadyPresent {
				existing.Turns = append(existing.Turns, *turn)
			}
		}
	} else {
		copySess := *s
		if turn != nil && len(copySess.Turns) == 0 {
			copySess.Turns = []models.MessageTurn{*turn}
		}
		m.Sessions = append([]*models.Session{&copySess}, m.Sessions...)
		if len(m.Sessions) > 200 {
			m.Sessions = m.Sessions[:200]
		}
	}

	m.applyHarnessFilter()
}

// applyHarnessFilter filters m.Sessions into m.FilteredSessions and builds table rows.
func (m *Model) applyHarnessFilter() {
	var filtered []*models.Session
	for _, s := range m.Sessions {
		if m.HarnessFilter == "" || strings.EqualFold(s.AgentName, m.HarnessFilter) || strings.Contains(strings.ToLower(s.AgentName), strings.ToLower(m.HarnessFilter)) {
			filtered = append(filtered, s)
		}
	}
	m.FilteredSessions = filtered
	m.rebuildSessionRows()
}

// cycleHarnessFilter cycles through available harness names from discovered sessions.
func (m *Model) cycleHarnessFilter() {
	distinctAgentsMap := make(map[string]bool)
	for _, s := range m.Sessions {
		if s.AgentName != "" {
			distinctAgentsMap[strings.ToLower(s.AgentName)] = true
		}
	}

	var agents []string
	for a := range distinctAgentsMap {
		agents = append(agents, a)
	}
	sort.Strings(agents)

	options := append([]string{""}, agents...)
	currentIdx := 0
	for i, opt := range options {
		if strings.EqualFold(opt, m.HarnessFilter) {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(options)
	m.HarnessFilter = options[nextIdx]
	m.applyHarnessFilter()

	if m.HarnessFilter == "" {
		m.StatusMessage = "Harness filter: ALL agents"
	} else {
		m.StatusMessage = fmt.Sprintf("Harness filter: %s", m.HarnessFilter)
	}
}

// rebuildSessionRows converts m.FilteredSessions to Bubble Tea table rows.
func (m *Model) rebuildSessionRows() {
	rows := make([]table.Row, 0, len(m.FilteredSessions))
	for _, s := range m.FilteredSessions {
		ts := s.StartTime
		if ts.IsZero() {
			ts = s.CreatedAt
		}
		timeStr := ts.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "recently"
		}

		sessShortID := s.SessionID
		if sessShortID == "" {
			sessShortID = s.ID
		}
		if len(sessShortID) > 16 {
			sessShortID = sessShortID[:16] + "..."
		}

		modelStr := s.ModelRaw
		if modelStr == "" {
			modelStr = s.ModelResolved
		}
		if modelStr == "" {
			modelStr = "default"
		}

		tokensCol := fmt.Sprintf("%s / %s / %s",
			humanize.Comma(s.InputTokens),
			humanize.Comma(s.OutputTokens),
			humanize.Comma(s.CacheReadTokens),
		)

		costCol := fmt.Sprintf("$%.4f", s.NetCostUSD)
		turnsCol := fmt.Sprintf("%d", len(s.Turns))

		rows = append(rows, table.Row{
			timeStr,
			s.AgentName,
			sessShortID,
			modelStr,
			turnsCol,
			tokensCol,
			costCol,
		})
	}
	m.SessionRows = rows
	m.SessionTable.SetRows(m.SessionRows)
}

// SelectedSession returns the currently highlighted session in the sessions table, or nil.
func (m *Model) SelectedSession() *models.Session {
	if len(m.FilteredSessions) == 0 {
		return nil
	}
	idx := m.SessionTable.Cursor()
	if idx < 0 || idx >= len(m.FilteredSessions) {
		return m.FilteredSessions[0]
	}
	return m.FilteredSessions[idx]
}

// CalculateThroughput returns estimated tokens per second over the recent 10s window.
func (m *Model) CalculateThroughput() float64 {
	if len(m.RecentSamples) == 0 {
		return 0
	}
	var sum int64
	for _, s := range m.RecentSamples {
		sum += s.Tokens
	}
	return float64(sum) / 10.0
}

// CalculateCacheHitRate returns percentage of prompt tokens read from cache.
func (m *Model) CalculateCacheHitRate() float64 {
	totalPrompt := m.TotalInputTokens + m.TotalCacheReadTokens
	if totalPrompt == 0 {
		return 0
	}
	return (float64(m.TotalCacheReadTokens) / float64(totalPrompt)) * 100.0
}

func (m *Model) recalculateLayout() {
	if m.Width <= 0 || m.Height <= 0 {
		return
	}

	availableWidth := m.Width - 6
	if availableWidth < 60 {
		availableWidth = 60
	}

	// Live Table Layout
	liveTableHeight := m.Height - 14
	if liveTableHeight < 5 {
		liveTableHeight = 5
	}
	m.Table.SetHeight(liveTableHeight)

	colTime := 10
	colAgent := 14
	colProject := 16
	colCost := 12

	remaining := availableWidth - (colTime + colAgent + colProject + colCost)
	colModel := remaining * 45 / 100
	colTokens := remaining * 55 / 100

	if colModel < 15 {
		colModel = 15
	}
	if colTokens < 20 {
		colTokens = 20
	}

	m.Table.SetColumns([]table.Column{
		{Title: "TIME", Width: colTime},
		{Title: "AGENT", Width: colAgent},
		{Title: "PROJECT", Width: colProject},
		{Title: "MODEL", Width: colModel},
		{Title: "IN / OUT / CACHE", Width: colTokens},
		{Title: "COST (USD)", Width: colCost},
	})
	m.Table.SetWidth(availableWidth)

	// Sessions Table Layout (split with inspector pane)
	sessionTableHeight := 7
	if m.Height > 35 && !m.ExpandedInspector {
		sessionTableHeight = 10
	} else if m.Height <= 24 {
		sessionTableHeight = 5
	}
	m.SessionTable.SetHeight(sessionTableHeight)

	colSessTime := 10
	colSessHarness := 14
	colSessID := 18
	colSessTurns := 8
	colSessCost := 12

	remSess := availableWidth - (colSessTime + colSessHarness + colSessID + colSessTurns + colSessCost)
	colSessModel := remSess * 45 / 100
	colSessTokens := remSess * 55 / 100

	if colSessModel < 15 {
		colSessModel = 15
	}
	if colSessTokens < 20 {
		colSessTokens = 20
	}

	m.SessionTable.SetColumns([]table.Column{
		{Title: "TIME", Width: colSessTime},
		{Title: "HARNESS", Width: colSessHarness},
		{Title: "SESSION ID", Width: colSessID},
		{Title: "MODEL", Width: colSessModel},
		{Title: "TURNS", Width: colSessTurns},
		{Title: "IN / OUT / CACHE", Width: colSessTokens},
		{Title: "COST (USD)", Width: colSessCost},
	})
	m.SessionTable.SetWidth(availableWidth)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
