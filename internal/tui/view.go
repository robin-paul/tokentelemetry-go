package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

func lipglossNormalBorder() lipgloss.Border {
	return lipgloss.NormalBorder()
}

// View renders the full terminal user interface layout.
func (m Model) View() string {
	if m.Width == 0 {
		return "Initializing TokenTelemetry TUI..."
	}

	var sections []string

	// 1. Header
	sections = append(sections, m.renderHeader())

	// 2. KPI Metrics Cards
	sections = append(sections, m.renderKPICards())

	// 3. Main Content: Live Feed or Sessions Browser
	if m.ViewMode == ViewModeSessions {
		sections = append(sections, m.renderSessionsView())
	} else {
		sections = append(sections, m.renderTable())
	}

	// 4. Footer & Help Bar
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderHeader() string {
	title := StyleTitle.Render("⚡ TOKEN TELEMETRY COLLECTOR")

	var modeBadge string
	if m.ViewMode == ViewModeSessions {
		modeBadge = lipgloss.NewStyle().Bold(true).Foreground(ColorFgBright).Background(ColorAccent).Padding(0, 1).Render("📋 SESSIONS VIEW")
	} else {
		modeBadge = lipgloss.NewStyle().Bold(true).Foreground(ColorFgBright).Background(ColorPrimary).Padding(0, 1).Render("⚡ LIVE TURNS")
	}

	var hubBadge string
	if m.Paused {
		hubBadge = StyleBadgePaused.Render("⏸ PAUSED")
	} else if m.HubStatus == "ONLINE" {
		hubBadge = StyleBadgeOnline.Render(fmt.Sprintf("● HUB: ONLINE (%v)", m.HubLatency.Round(100*time.Microsecond)))
	} else if m.HubStatus == "OFFLINE" {
		hubBadge = StyleBadgeOffline.Render("● HUB: OFFLINE")
	} else {
		hubBadge = StyleBadgePaused.Render("● HUB: " + m.HubStatus)
	}

	uptime := time.Since(m.StartTime).Truncate(time.Second)
	info := StyleCardSub.Render(fmt.Sprintf("UPTIME: %s | WATCHING: %d ROOTS", uptime, m.ActiveRoots))

	gap := m.Width - lipgloss.Width(title) - lipgloss.Width(modeBadge) - lipgloss.Width(hubBadge) - lipgloss.Width(info) - 6
	if gap < 1 {
		gap = 1
	}

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center,
		title,
		" ",
		modeBadge,
		" ",
		hubBadge,
		strings.Repeat(" ", gap),
		info,
	)

	return StyleHeader.Width(m.Width - 2).Render(headerContent)
}

func (m Model) renderKPICards() string {
	availableWidth := m.Width - 4
	if availableWidth < 40 {
		availableWidth = 40
	}

	cardWidth := (availableWidth / 3) - 2
	if cardWidth < 22 {
		cardWidth = 22
	}

	// Card 1: Throughput
	tps := m.CalculateThroughput()
	c1Title := StyleCardTitle.Render("THROUGHPUT")
	c1Val := StyleCardValue.Render(fmt.Sprintf("%.1f tok/s", tps))
	c1Sub := StyleCardSub.Render(fmt.Sprintf("%s turns | %s sessions",
		humanize.Comma(int64(m.TotalTurns)), humanize.Comma(int64(m.TotalSessions))))
	card1 := StyleCard.Width(cardWidth).Render(lipgloss.JoinVertical(lipgloss.Left, c1Title, c1Val, c1Sub))

	// Card 2: Cache Efficiency
	hitRate := m.CalculateCacheHitRate()
	c2Title := StyleCardTitle.Render("CACHE EFFICIENCY")
	c2Val := StyleCardValue.Render(fmt.Sprintf("%.1f%% Hit Rate", hitRate))
	c2Sub := StyleCardSub.Render(fmt.Sprintf("%s cached tokens", humanize.Comma(m.TotalCacheReadTokens)))
	card2 := StyleCard.Width(cardWidth).Render(lipgloss.JoinVertical(lipgloss.Left, c2Title, c2Val, c2Sub))

	// Card 3: Estimated Cost
	c3Title := StyleCardTitle.Render("ESTIMATED COST")
	c3Val := StyleCardValue.Render(fmt.Sprintf("$%.4f Net", m.TotalNetCostUSD))
	savedUSD := m.TotalGrossCostUSD - m.TotalNetCostUSD
	if savedUSD < 0 {
		savedUSD = 0
	}
	c3Sub := StyleCardSub.Render(fmt.Sprintf("Saved: $%.4f USD", savedUSD))
	card3 := StyleCard.Width(cardWidth).Render(lipgloss.JoinVertical(lipgloss.Left, c3Title, c3Val, c3Sub))

	if m.Width < 80 {
		// Stacked on narrow screens
		return lipgloss.JoinVertical(lipgloss.Left, card1, card2, card3)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, card1, " ", card2, " ", card3)
}

func (m Model) renderTable() string {
	return lipgloss.NewStyle().
		Margin(1, 0, 0, 0).
		Render(m.Table.View())
}

func (m Model) renderSessionsView() string {
	var sections []string

	filterStr := "ALL AGENTS"
	if m.HarnessFilter != "" {
		filterStr = strings.ToUpper(m.HarnessFilter)
	}
	harnessHeader := StyleCardTitle.Render(fmt.Sprintf("📂 RECENT SESSIONS [%s] (%d matching sessions)", filterStr, len(m.FilteredSessions)))
	sections = append(sections, harnessHeader)

	// Session Table
	sections = append(sections, lipgloss.NewStyle().Margin(0, 0, 1, 0).Render(m.SessionTable.View()))

	// Inspector Pane
	sections = append(sections, m.renderInspectorPane())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderInspectorPane() string {
	sess := m.SelectedSession()
	if sess == nil {
		return StyleCard.Width(m.Width - 4).Render(StyleCardSub.Render("No session selected or found matching filter."))
	}

	availWidth := m.Width - 6
	if availWidth < 40 {
		availWidth = 40
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🔍 SESSION INSPECTOR: " + sess.ID)

	var lines []string
	lines = append(lines, title)

	// Line 1: File path
	filePath := sess.FilePath
	if filePath == "" {
		filePath = "(in-memory / active stream)"
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorFgMuted).Render("File: ")+lipgloss.NewStyle().Foreground(ColorFgBright).Render(filePath))

	// Line 2: Agent / Model / Duration / Status
	durStr := fmt.Sprintf("%.1fs", sess.DurationSeconds)
	if sess.DurationSeconds <= 0 && !sess.StartTime.IsZero() && !sess.EndTime.IsZero() {
		durStr = fmt.Sprintf("%.1fs", sess.EndTime.Sub(sess.StartTime).Seconds())
	}
	modelInfo := sess.ModelRaw
	if sess.ModelResolved != "" && sess.ModelResolved != sess.ModelRaw {
		modelInfo = fmt.Sprintf("%s (resolved: %s)", sess.ModelRaw, sess.ModelResolved)
	}
	lines = append(lines, fmt.Sprintf("%s %s  │  %s %s  │  %s %s  │  %s %s",
		StyleCardSub.Render("Agent:"), lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(sess.AgentName),
		StyleCardSub.Render("Model:"), lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render(modelInfo),
		StyleCardSub.Render("Duration:"), lipgloss.NewStyle().Foreground(ColorFgBright).Render(durStr),
		StyleCardSub.Render("Status:"), lipgloss.NewStyle().Foreground(ColorFgBright).Render(sess.Status),
	))

	// Line 3: Tokens & Cost
	savedUSD := sess.GrossCostUSD - sess.NetCostUSD
	if savedUSD < 0 {
		savedUSD = 0
	}
	lines = append(lines, fmt.Sprintf("%s %s in, %s out, %s cache read  │  %s $%.4f (Gross: $%.4f, Saved: $%.4f)",
		StyleCardSub.Render("Tokens:"),
		humanize.Comma(sess.InputTokens),
		humanize.Comma(sess.OutputTokens),
		humanize.Comma(sess.CacheReadTokens),
		StyleCardSub.Render("Cost:"),
		sess.NetCostUSD,
		sess.GrossCostUSD,
		savedUSD,
	))

	// Turns breakdown
	if len(sess.Turns) > 0 {
		lines = append(lines, "")
		turnsHeader := StyleCardTitle.Render(fmt.Sprintf("Message Turns (%d turns):", len(sess.Turns)))
		if !m.ExpandedInspector && len(sess.Turns) > 3 {
			turnsHeader += StyleCardSub.Render(" [showing latest 3 turns, press Enter to expand]")
		}
		lines = append(lines, turnsHeader)

		turnsToShow := sess.Turns
		if !m.ExpandedInspector && len(turnsToShow) > 3 {
			turnsToShow = turnsToShow[len(turnsToShow)-3:]
		}

		for _, t := range turnsToShow {
			toolsStr := ""
			if len(t.ToolsInvoked) > 0 {
				toolsStr = fmt.Sprintf(" | tools: [%s]", strings.Join(t.ToolsInvoked, ", "))
			}
			turnLine := fmt.Sprintf("  #%-2d [%-9s] [%-16s] in:%-5s out:%-4s cache:%-5s cost:$%.4f%s",
				t.TurnIndex,
				t.Role,
				truncateString(t.ModelName, 16),
				humanize.Comma(t.InputTokens),
				humanize.Comma(t.OutputTokens),
				humanize.Comma(t.CacheReadTokens),
				t.CostUSD,
				toolsStr,
			)
			lines = append(lines, lipgloss.NewStyle().Foreground(ColorFgBright).Render(turnLine))
		}
	}

	if len(sess.SubagentRuns) > 0 {
		lines = append(lines, "")
		lines = append(lines, StyleCardTitle.Render(fmt.Sprintf("Subagent Runs (%d):", len(sess.SubagentRuns))))
		for _, sub := range sess.SubagentRuns {
			subLine := fmt.Sprintf("  • Child ID: %s | Agent: %s | Tokens: %s | Cost: $%.4f",
				sub.ChildSessionID, sub.AgentType, humanize.Comma(sub.Tokens), sub.CostUSD)
			lines = append(lines, lipgloss.NewStyle().Foreground(ColorFgBright).Render(subLine))
		}
	}

	content := strings.Join(lines, "\n")
	return StyleCard.Width(availWidth).Render(content)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func (m Model) renderFooter() string {
	var statusLine string
	if m.ErrorMessage != "" {
		statusLine = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true).Render("Error: " + m.ErrorMessage)
	} else {
		statusLine = StyleCardSub.Render(m.StatusMessage)
	}

	var keybindings string
	if m.ViewMode == ViewModeSessions {
		keybindings = fmt.Sprintf("%s quit  %s live turns  %s harness  %s expand/collapse  %s select",
			StyleKeyHelp.Render("[q]"),
			StyleKeyHelp.Render("[Tab/s]"),
			StyleKeyHelp.Render("[h]"),
			StyleKeyHelp.Render("[Enter]"),
			StyleKeyHelp.Render("[↑/↓]"),
		)
	} else {
		keybindings = fmt.Sprintf("%s quit  %s sessions  %s clear  %s pause/resume  %s scroll",
			StyleKeyHelp.Render("[q]"),
			StyleKeyHelp.Render("[Tab/s]"),
			StyleKeyHelp.Render("[c]"),
			StyleKeyHelp.Render("[p]"),
			StyleKeyHelp.Render("[↑/↓]"),
		)
	}

	gap := m.Width - lipgloss.Width(statusLine) - lipgloss.Width(keybindings) - 4
	if gap < 1 {
		gap = 1
	}

	footerContent := lipgloss.JoinHorizontal(lipgloss.Center,
		statusLine,
		strings.Repeat(" ", gap),
		keybindings,
	)

	return StyleFooter.Width(m.Width - 2).Render(footerContent)
}
