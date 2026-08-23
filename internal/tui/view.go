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

	// 3. Live Turn Feed Table
	sections = append(sections, m.renderTable())

	// 4. Footer & Help Bar
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderHeader() string {
	title := StyleTitle.Render("⚡ TOKEN TELEMETRY COLLECTOR")

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

	gap := m.Width - lipgloss.Width(title) - lipgloss.Width(hubBadge) - lipgloss.Width(info) - 4
	if gap < 1 {
		gap = 1
	}

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center,
		title,
		"  ",
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

func (m Model) renderFooter() string {
	var statusLine string
	if m.ErrorMessage != "" {
		statusLine = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true).Render("Error: " + m.ErrorMessage)
	} else {
		statusLine = StyleCardSub.Render(m.StatusMessage)
	}

	keybindings := fmt.Sprintf("%s quit  %s clear  %s pause/resume  %s scroll",
		StyleKeyHelp.Render("[q]"),
		StyleKeyHelp.Render("[c]"),
		StyleKeyHelp.Render("[p]"),
		StyleKeyHelp.Render("[↑/↓]"),
	)

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
