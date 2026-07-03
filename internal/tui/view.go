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
	bg       = lipgloss.Color("#1a1a2e")
	accent   = lipgloss.Color("#e94560")
	muted    = lipgloss.Color("#6e6e8e")
	text     = lipgloss.Color("#c9d1d9")
	selected = lipgloss.Color("#16213e")
	border   = lipgloss.Color("#2a2a4a")
	green    = lipgloss.Color("#3fb950")
	red      = lipgloss.Color("#f85149")
	yellow   = lipgloss.Color("#f0a500")

	barStyle = lipgloss.NewStyle().
			Background(bg).
			Foreground(muted).
			Padding(0, 1)

	accentStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Background(selected).
			Foreground(text).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(muted)

	okStyle = lipgloss.NewStyle().
		Foreground(green)

	conflictStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true)

	addStyle = lipgloss.NewStyle().Foreground(green)
	subStyle = lipgloss.NewStyle().Foreground(red)

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(border).
			Background(bg)
)

func (m Model) View() string {
	if !m.vpReady {
		return barStyle.Width(m.width).Render("ghostlog: waiting for terminal size...")
	}

	top := m.renderTopBar()
	bot := m.renderBottomBar()

	contentH := m.height - lipgloss.Height(top) - lipgloss.Height(bot)
	leftW := m.width * 40 / 100
	rightW := m.width - leftW

	var left string
	if m.ViewMode == "treemap" {
		left = paneStyle.
			Width(leftW - 2).
			Height(contentH - 2).
			Render(m.renderTreemap(leftW-4, contentH-4))
	} else {
		left = paneStyle.
			Width(leftW - 2).
			Height(contentH - 2).
			Render(m.renderBurstList(leftW - 4))
	}

	right := paneStyle.
		Width(rightW - 2).
		Height(contentH - 2).
		Render(m.vp.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, top, body, bot)
}

func (m Model) renderTopBar() string {
	session := fmtDuration(time.Since(m.sessionStart))
	speed := accentStyle.Render(fmt.Sprintf("[AGENT SPEED: %.1f commits/min]", m.CPSMetric))
	sess := fmt.Sprintf("SESSION: %s", session)
	watching := dimStyle.Render("watching " + m.repoPath)
	mid := fmt.Sprintf("%s | %s | %s", speed, sess, watching)
	return barStyle.Width(m.width).Render(mid)
}

func (m Model) renderBottomBar() string {
	scroll := "auto: off"
	if m.AutoScroll {
		scroll = "auto: on"
	}
	line := fmt.Sprintf(
		"Total: %s %s | %d bursts | %s | [a]uto / [c]opy / [q]uit",
		addStyle.Render(fmt.Sprintf("+%d", m.totalAdded)),
		subStyle.Render(fmt.Sprintf("-%d", m.totalRemoved)),
		len(m.Bursts),
		scroll,
	)
	return barStyle.Width(m.width).Render(line)
}

func (m Model) renderBurstList(w int) string {
	if len(m.Bursts) == 0 {
		return dimStyle.Render("\n  Waiting for commits...\n\n  Start your AI agent.\n  ghostlog captures every commit.")
	}

	var sb strings.Builder
	for i, b := range m.Bursts {
		line := formatBurst(i+1, b, w-3)
		if i == m.SelectedBurstIndex {
			sb.WriteString(selectedStyle.Render("> " + line))
		} else {
			sb.WriteString(dimStyle.Render("  " + line))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func formatBurst(n int, b analyzer.Burst, w int) string {
	dur := formatting.Duration(int64(b.LastTime.Sub(b.StartTime).Seconds()))
	if dur == "0s" {
		dur = "<1s"
	}

	status := okStyle.Render("[OK]")
	if b.HasConflict || (b.ComplexityAfter-b.ComplexityBefore) > 10 || len(b.UntestedFunctions) > 0 {
		status = conflictStyle.Render("[WARN]")
	}

	added := addStyle.Render(fmt.Sprintf("+%d", b.LinesAdded))
	removed := subStyle.Render(fmt.Sprintf("-%d", b.LinesRemoved))

	return fmt.Sprintf("[#%d] %d commits in %s  %s %s %s (+%s) across %d files",
		b.ID,
		len(b.Hashes),
		dur,
		status,
		added,
		removed,
		formatting.Bytes(b.BytesAdded),
		b.FilesChanged,
	)
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (m Model) renderTreemap(w, h int) string {
	if m.TotalLines == 0 || len(m.Treemap) == 0 {
		return dimStyle.Render("Loading treemap...")
	}
	totalChars := w * h
	var sb strings.Builder
	for _, c := range m.Treemap {
		chars := int(float64(c.Lines) / float64(m.TotalLines) * float64(totalChars))
		if chars < 1 {
			chars = 1
		}
		color := fadeColor(c.LastTouched)
		style := lipgloss.NewStyle().Foreground(color)
		block := style.Render(strings.Repeat("█", chars))
		sb.WriteString(block)
	}
	// The output might exceed the box due to rounding and minimum 1 char. 
	// We just truncate the final string to totalChars (ignoring ansi codes is hard, so we just let lipgloss clip it)
	// Actually lipgloss handles overflow in paneStyle, but to avoid breaking layout we can do a simple chunking.
	// We can use lipgloss width/height constraints.
	return sb.String()
}

func fadeColor(lastTouched time.Time) lipgloss.Color {
	if lastTouched.IsZero() {
		return lipgloss.Color("#1a1a2e")
	}
	elapsed := time.Since(lastTouched).Seconds()
	if elapsed > 2.0 {
		return lipgloss.Color("#1a1a2e")
	}
	ratio := 1.0 - (elapsed / 2.0)
	if ratio < 0 {
		ratio = 0
	}
	r := int(26.0 + (233.0 - 26.0)*ratio)
	g := int(26.0 + (69.0 - 26.0)*ratio)
	b := int(46.0 + (96.0 - 46.0)*ratio)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}
