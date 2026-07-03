package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/salarkhannn/ghostlog/internal/analyzer"
	"github.com/salarkhannn/ghostlog/internal/tui/formatting"
)

var (
	colorBg       = lipgloss.Color("#0d0d1a")
	colorAccent   = lipgloss.Color("#e94560")
	colorMuted    = lipgloss.Color("#4a4a6a")
	colorText     = lipgloss.Color("#c9d1d9")
	colorSelected = lipgloss.Color("#1e2a3a")
	colorBorder   = lipgloss.Color("#2a2a4a")
	colorGreen    = lipgloss.Color("#3fb950")
	colorRed      = lipgloss.Color("#f85149")

	topBarStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 2)

	bottomBarStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorMuted).
			Padding(0, 2)

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	selectedRowStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorText).
				Bold(true)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	burstHeaderStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	addedStyle   = lipgloss.NewStyle().Foreground(colorGreen)
	removedStyle = lipgloss.NewStyle().Foreground(colorRed)
)

func (m Model) View() string {
	if m.width == 0 {
		return "ghostlog: initializing...\n"
	}

	top := m.renderTopBar()
	bot := m.renderBottomBar()

	contentH := m.height - lipgloss.Height(top) - lipgloss.Height(bot)
	leftW := m.width * 40 / 100
	rightW := m.width - leftW

	left := paneStyle.
		Width(leftW - 2).
		Height(contentH - 2).
		Render(m.renderBurstList(leftW-4, contentH-4))

	right := paneStyle.
		Width(rightW - 2).
		Height(contentH - 2).
		Render(m.renderDiff(rightW-4, contentH-4))

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, top, body, bot)
}

func (m Model) renderTopBar() string {
	speed := fmt.Sprintf("⚡ Agent Speed: %.1f commits/min", m.CPSMetric)
	watching := fmt.Sprintf("  watching %s", m.repoPath)
	gap := m.width - lipgloss.Width(speed) - lipgloss.Width(watching) - 4
	if gap < 1 {
		gap = 1
	}
	line := speed + strings.Repeat(" ", gap) + watching
	return topBarStyle.Width(m.width).Render(line)
}

func (m Model) renderBottomBar() string {
	session := time.Since(m.sessionStart).Round(time.Second)
	totalCommits := 0
	for _, b := range m.Bursts {
		totalCommits += len(b.Hashes)
	}
	scroll := "auto-scroll: off"
	if m.AutoScroll {
		scroll = "auto-scroll: on"
	}
	line := fmt.Sprintf(
		"  commits: %d  +%d -%d lines  session: %s  %s  [j/k] navigate  [a] toggle scroll  [c] copy  [q] quit",
		totalCommits, m.totalAdded, m.totalRemoved, session, scroll,
	)
	return bottomBarStyle.Width(m.width).Render(line)
}

func (m Model) renderBurstList(w, h int) string {
	if len(m.Bursts) == 0 {
		return normalRowStyle.Render("\n  Waiting for commits...\n\n  Start your AI coding agent.\n  ghostlog will capture every commit.")
	}

	var sb strings.Builder
	visible := h
	start := 0
	if m.SelectedBurstIndex >= visible {
		start = m.SelectedBurstIndex - visible + 1
	}

	for i := start; i < len(m.Bursts) && i < start+visible; i++ {
		b := m.Bursts[i]
		label := renderBurstLabel(i+1, b, w)
		if i == m.SelectedBurstIndex {
			sb.WriteString(selectedRowStyle.Width(w).Render(label))
		} else {
			sb.WriteString(normalRowStyle.Width(w).Render(label))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func renderBurstLabel(n int, b analyzer.Burst, w int) string {
	dur := formatting.Duration(int64(b.LastTime.Sub(b.StartTime).Seconds()))
	if dur == "0s" {
		dur = "<1s"
	}
	added := addedStyle.Render(fmt.Sprintf("+%d", b.LinesAdded))
	removed := removedStyle.Render(fmt.Sprintf("-%d", b.LinesRemoved))
	header := burstHeaderStyle.Render(fmt.Sprintf("[Burst #%d]", b.ID))
	return fmt.Sprintf("%s %d commits in %s  %s %s (+%s) across %d files",
		header, len(b.Hashes), dur, added, removed,
		formatting.Bytes(b.BytesAdded), b.FilesChanged,
	)
}

func (m Model) renderDiff(w, h int) string {
	if m.selectedDiff == "" {
		if len(m.Bursts) == 0 {
			return normalRowStyle.Render("\n  Select a burst to view its diff.")
		}
		return normalRowStyle.Render("\n  Loading diff...")
	}

	lines := strings.Split(m.selectedDiff, "\n")
	end := m.ScrollOffset + h
	if end > len(lines) {
		end = len(lines)
	}
	if m.ScrollOffset >= len(lines) {
		return ""
	}
	return strings.Join(lines[m.ScrollOffset:end], "\n")
}
