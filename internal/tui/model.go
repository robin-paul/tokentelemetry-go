package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
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
	Width                   int
	Height                  int
	Table                   table.Model
	Rows                    []table.Row
	TotalInputTokens        int64
	TotalOutputTokens       int64
	TotalCacheReadTokens    int64
	TotalCacheCreationTokens int64
	TotalNetCostUSD         float64
	TotalGrossCostUSD       float64
	TotalTurns              int
	TotalSessions           int
	RecentSamples           []TokenSample
	HubURL                  string
	HubStatus               string
	HubLatency              time.Duration
	ActiveRoots             int
	Paused                  bool
	StartTime               time.Time
	StatusMessage           string
	ErrorMessage            string
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

	return Model{
		Table:         t,
		Rows:          make([]table.Row, 0),
		RecentSamples: make([]TokenSample, 0),
		HubURL:        hubURL,
		HubStatus:     "CHECKING...",
		ActiveRoots:   activeRoots,
		StartTime:     time.Now(),
		StatusMessage: "Listening for agent transcript updates...",
	}
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
		case msg.String() == "c":
			m.Rows = nil
			m.Table.SetRows(m.Rows)
			m.TotalInputTokens = 0
			m.TotalOutputTokens = 0
			m.TotalCacheReadTokens = 0
			m.TotalCacheCreationTokens = 0
			m.TotalNetCostUSD = 0
			m.TotalGrossCostUSD = 0
			m.TotalTurns = 0
			m.TotalSessions = 0
			m.RecentSamples = nil
			m.StatusMessage = "Cleared telemetry metrics and turn feed"
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

			// Prepend newest to top or append to bottom (keep last 200 rows)
			m.Rows = append([]table.Row{row}, m.Rows...)
			if len(m.Rows) > 200 {
				m.Rows = m.Rows[:200]
			}
			m.Table.SetRows(m.Rows)
		}

	case SessionIngestedMsg:
		if !m.Paused && msg.Session != nil {
			m.TotalSessions++
			m.TotalGrossCostUSD += msg.Session.GrossCostUSD
			m.StatusMessage = fmt.Sprintf("Ingested %s session (%s): %d turns",
				msg.Session.AgentName, msg.Session.ProjectName, len(msg.Session.Turns))
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

	m.Table, cmd = m.Table.Update(msg)
	return m, cmd
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

	// Height available for table: total height minus header (4), KPI cards (6), footer (3)
	tableHeight := m.Height - 14
	if tableHeight < 5 {
		tableHeight = 5
	}
	m.Table.SetHeight(tableHeight)

	// Responsive column widths
	availableWidth := m.Width - 6
	if availableWidth < 60 {
		availableWidth = 60
	}

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
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
