package tui

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
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

type treemapItem struct {
	path        string
	name        string
	isDir       bool
	lines       int
	lastTouched time.Time
}

func (m Model) getGroupedItems() []*treemapItem {
	groups := make(map[string]*treemapItem)
	prefix := m.CurrentDir
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for _, c := range m.Treemap {
		if prefix != "" && !strings.HasPrefix(c.Path, prefix) {
			continue
		}

		rel := c.Path[len(prefix):]
		parts := strings.Split(rel, "/")
		childName := parts[0]
		if childName == "" {
			continue
		}

		fullChildPath := childName
		if m.CurrentDir != "" {
			fullChildPath = filepath.Join(m.CurrentDir, childName)
		}

		isDir := len(parts) > 1

		item, exists := groups[childName]
		if !exists {
			item = &treemapItem{
				path:        fullChildPath,
				name:        childName,
				isDir:       isDir,
				lines:       0,
				lastTouched: c.LastTouched,
			}
			groups[childName] = item
		}

		item.lines += c.Lines
		if c.LastTouched.After(item.lastTouched) {
			item.lastTouched = c.LastTouched
		}
		if isDir {
			item.isDir = true
		}
	}

	var items []*treemapItem
	for _, item := range groups {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].lines > items[j].lines
	})

	return items
}

type treemapBox struct {
	path        string
	name        string
	isDir       bool
	lines       int
	lastTouched time.Time
	colorIndex  int
	x, y, w, h  int
	weight      float64
}

func partition(boxes []*treemapBox, x, y, w, h int) {
	if len(boxes) == 0 {
		return
	}
	if len(boxes) == 1 {
		boxes[0].x = x
		boxes[0].y = y
		boxes[0].w = w
		boxes[0].h = h
		return
	}

	total := 0.0
	for _, b := range boxes {
		total += b.weight
	}
	if total == 0 {
		for _, b := range boxes {
			b.weight = 1.0
		}
		total = float64(len(boxes))
	}

	firstWeight := boxes[0].weight
	ratio := firstWeight / total

	if float64(w) > 2.0*float64(h) {
		splitW := int(float64(w) * ratio)
		if splitW < 1 {
			splitW = 1
		}
		if splitW >= w {
			splitW = w - 1
		}
		partition(boxes[:1], x, y, splitW, h)
		partition(boxes[1:], x+splitW, y, w-splitW, h)
	} else {
		splitH := int(float64(h) * ratio)
		if splitH < 1 {
			splitH = 1
		}
		if splitH >= h {
			splitH = h - 1
		}
		partition(boxes[:1], x, y, w, splitH)
		partition(boxes[1:], x, y+splitH, w, h-splitH)
	}
}

type cell struct {
	char  rune
	style lipgloss.Style
}

func (m Model) renderTreemap(w, h int) string {
	if m.TotalLines == 0 || len(m.Treemap) == 0 {
		return dimStyle.Render("Loading treemap...")
	}

	grouped := m.getGroupedItems()
	if len(grouped) == 0 {
		return dimStyle.Render("Empty directory.\n[Backspace] Go back")
	}

	selIdx := m.SelectedTreemapIndex
	if selIdx >= len(grouped) {
		selIdx = len(grouped) - 1
	}
	if selIdx < 0 {
		selIdx = 0
	}

	boxes := make([]*treemapBox, len(grouped))
	for i, item := range grouped {
		weight := math.Log2(float64(item.lines) + 1.0)
		if weight < 1.0 {
			weight = 1.0
		}
		boxes[i] = &treemapBox{
			path:        item.path,
			name:        item.name,
			isDir:       item.isDir,
			lines:       item.lines,
			lastTouched: item.lastTouched,
			colorIndex:  i,
			weight:      weight,
		}
	}

	breadcrumb := "📁 [root]"
	if m.CurrentDir != "" {
		breadcrumb = "📁 " + strings.ReplaceAll(m.CurrentDir, "/", " > ")
	}
	breadcrumb = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(breadcrumb)

	controls := dimStyle.Render("[Tab] Cycle | [Enter] Open | [Backspace] Up")

	gridH := h - 2
	if gridH < 3 {
		gridH = 3
	}

	partition(boxes, 0, 0, w, gridH)

	grid := make([][]cell, gridH)
	for i := range grid {
		grid[i] = make([]cell, w)
		for j := range grid[i] {
			grid[i][j] = cell{char: ' '}
		}
	}

	draw := func(cx, cy int, r rune, style lipgloss.Style) {
		if cx >= 0 && cx < w && cy >= 0 && cy < gridH {
			grid[cy][cx] = cell{char: r, style: style}
		}
	}

	for i, box := range boxes {
		if box.w <= 0 || box.h <= 0 {
			continue
		}

		isSel := i == selIdx
		color := fadeColor(box.lastTouched, box.colorIndex)

		var borderStyle lipgloss.Style
		if isSel {
			borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00f0ff")).Bold(true)
		} else {
			borderStyle = lipgloss.NewStyle().Foreground(color)
		}

		var topL, topR, botL, botR, horiz, vert rune
		if isSel {
			topL, topR, botL, botR, horiz, vert = '╔', '╗', '╚', '╝', '═', '║'
		} else {
			topL, topR, botL, botR, horiz, vert = '┌', '┐', '└', '┘', '─', '│'
		}

		for cy := box.y; cy < box.y+box.h; cy++ {
			for cx := box.x; cx < box.x+box.w; cx++ {
				isTop := cy == box.y
				isBottom := cy == box.y+box.h-1
				isLeft := cx == box.x
				isRight := cx == box.x+box.w-1

				if isTop && isLeft {
					draw(cx, cy, topL, borderStyle)
				} else if isTop && isRight {
					draw(cx, cy, topR, borderStyle)
				} else if isBottom && isLeft {
					draw(cx, cy, botL, borderStyle)
				} else if isBottom && isRight {
					draw(cx, cy, botR, borderStyle)
				} else if isTop || isBottom {
					draw(cx, cy, horiz, borderStyle)
				} else if isLeft || isRight {
					draw(cx, cy, vert, borderStyle)
				}
			}
		}

		availableWidth := box.w - 2
		if availableWidth > 0 && box.h >= 3 {
			label := box.name
			if box.isDir {
				label += "/"
			}
			if len(label) > availableWidth {
				if availableWidth > 3 {
					label = label[:availableWidth-3] + "..."
				} else {
					label = label[:availableWidth]
				}
			}

			centerY := box.y + box.h/2
			if box.h >= 4 && box.h%2 == 0 {
				centerY = box.y + box.h/2 - 1
			}

			var textStyle lipgloss.Style
			if isSel {
				textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true)
			} else if !box.lastTouched.IsZero() && time.Since(box.lastTouched).Seconds() < 2.0 {
				textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true)
			} else {
				textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a0a0c0"))
			}

			startX := box.x + (box.w-len(label))/2
			for idx, r := range label {
				draw(startX+idx, centerY, r, textStyle)
			}

			if box.h >= 4 {
				linesLabel := fmt.Sprintf("%d L", box.lines)
				if len(linesLabel) <= availableWidth {
					startLinesX := box.x + (box.w-len(linesLabel))/2
					for idx, r := range linesLabel {
						draw(startLinesX+idx, centerY+1, r, textStyle)
					}
				}
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(breadcrumb)
	sb.WriteRune('\n')

	for _, row := range grid {
		for _, cell := range row {
			if cell.char == ' ' {
				sb.WriteRune(' ')
			} else {
				sb.WriteString(cell.style.Render(string(cell.char)))
			}
		}
		sb.WriteRune('\n')
	}

	sb.WriteString(controls)
	return sb.String()
}

func fadeColor(lastTouched time.Time, index int) lipgloss.Color {
	baseColors := []string{"#3a3a5a", "#2d2d46", "#45456b", "#363654", "#292940"}
	baseColor := baseColors[index%len(baseColors)]
	
	if lastTouched.IsZero() {
		return lipgloss.Color(baseColor)
	}
	elapsed := time.Since(lastTouched).Seconds()
	if elapsed > 2.0 {
		return lipgloss.Color(baseColor)
	}
	ratio := 1.0 - (elapsed / 2.0)
	if ratio < 0 {
		ratio = 0
	}
	
	// Parse the base color hex string to RGB
	var br, bg, bb int
	fmt.Sscanf(baseColor, "#%02x%02x%02x", &br, &bg, &bb)
	
	// Highlight RGB: R=233, G=69, B=96 (#e94560)
	r := int(float64(br) + (233.0 - float64(br))*ratio)
	g := int(float64(bg) + (69.0 - float64(bg))*ratio)
	b := int(float64(bb) + (96.0 - float64(bb))*ratio)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}
