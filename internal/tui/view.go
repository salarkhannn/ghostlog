package tui

import (
	"fmt"

	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type cell struct {
	char  rune
	style *lipgloss.Style
	skip  bool
}

var (
	accent   = AccentColor
	muted    = MutedColor
	text     = FgColor
	selected = SelectedColor
	border   = BorderColor
	green    = OkColor
	red      = SoftRedColor
	yellow   = WarnColor

	barStyle      = BarStyle
	accentStyle   = AccentStyle
	selectedStyle = SelectedStyle
	dimStyle      = DimStyle
	okStyle       = OkStyle
	conflictStyle = ConflictStyle
	addStyle      = AddStyle
	subStyle      = SubStyle
	inlineStyle   = InlineStyle

	paneStyle       = RightPaneStyle
	rootStyle       = RootStyle
	warnStyle       = WarnStyle
	breadcrumbStyle = BreadcrumbStyle
)

func (m Model) View() string {
	width := m.width

	if !m.vpReady || !m.burstListReady {
		return barStyle.Width(width).Render("ghostlog: waiting for terminal size...")
	}

	top := m.renderTopBar(width)
	bot := m.renderBottomBar(width)

	contentH := m.height - lipgloss.Height(top) - lipgloss.Height(bot)
	leftW := width * 70 / 100
	rightW := width - leftW - 2

	var leftStyle lipgloss.Style
	var rightStyle lipgloss.Style
	if m.FocusPane == "diff" {
		leftStyle = LeftPaneStyle.BorderForeground(BorderColor)
		rightStyle = paneStyle.BorderForeground(PrimaryColor)
	} else {
		leftStyle = LeftPaneStyle.BorderForeground(PrimaryColor)
		rightStyle = paneStyle.BorderForeground(BorderColor)
	}

	var left string
	if m.ViewMode == "treemap" {
		left = leftStyle.Render(m.renderTreemap(leftW-4, contentH-2))
	} else if m.ViewMode == "sessions" {
		left = leftStyle.Width(leftW - 4).Render(m.sessionList.View())
	} else {
		left = leftStyle.Width(leftW - 4).Render(m.burstList.View())
	}

	right := rightStyle.Width(rightW - 2).Render(m.vp.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	
	finalOutput := lipgloss.JoinVertical(lipgloss.Left, top, body, bot)
	
	return rootStyle.Render(finalOutput)
}

func (m Model) renderTopBar(w int) string {
	session := fmtDuration(time.Since(m.sessionStart))
	speedStr := fmt.Sprintf("[AGENT SPEED: %.1f commits/min]", m.CPSMetric)
	speed := accentStyle.Render(speedStr)
	sessStr := fmt.Sprintf(" | SESSION: %s | ", session)
	sess := inlineStyle.Render(sessStr)
	
	repo := strings.TrimSpace(m.repoPath)
	maxRepoLen := w - runewidth.StringWidth(speedStr) - runewidth.StringWidth(sessStr) - 10
	if maxRepoLen > 5 && runewidth.StringWidth(repo) > maxRepoLen {
		repo = runewidth.Truncate(repo, maxRepoLen, "...")
	}
	
	watching := dimStyle.Render("watching " + repo)
	mid := lipgloss.JoinHorizontal(lipgloss.Top, speed, sess, watching)
	return barStyle.MaxWidth(w).Render(mid)
}

func (m Model) renderBottomBar(w int) string {
	scroll := "auto: off"
	if m.AutoScroll {
		scroll = "auto: on"
	}
	line := lipgloss.JoinHorizontal(lipgloss.Top,
		inlineStyle.Render("Total: "),
		addStyle.Render(fmt.Sprintf("+%d", m.totalAdded)),
		inlineStyle.Render(" "),
		subStyle.Render(fmt.Sprintf("-%d", m.totalRemoved)),
		inlineStyle.Render(fmt.Sprintf(" | %d bursts | %s | [tab] focus | [a]uto / [c]opy / [s]essions / [q]uit", len(m.Bursts), scroll)),
	)
	return barStyle.MaxWidth(w).Render(line)
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

func (m Model) renderTreemap(w, h int) string {
	if m.TotalLines == 0 || len(m.Treemap) == 0 {
		return dimStyle.Render("Loading treemap...")
	}

	gridH := h - 2
	if gridH < 3 {
		gridH = 3
	}

	items, boxes := m.getLayout(w, gridH)
	if len(items) == 0 {
		return dimStyle.Render("Empty directory.\n[Backspace] Go back")
	}

	selIdx := m.SelectedTreemapIndex
	if selIdx >= len(items) {
		selIdx = len(items) - 1
	}
	if selIdx < 0 {
		selIdx = 0
	}

	breadcrumbStr := "📁 [root]"
	if m.CurrentDir != "" {
		breadcrumbStr = "📁 " + strings.ReplaceAll(m.CurrentDir, "/", " > ")
	}
	breadcrumbStr = runewidth.Truncate(breadcrumbStr, w, "...")
	breadcrumb := breadcrumbStyle.Render(breadcrumbStr)

	controlsStr := "[Tab/H/L] Nav | [Enter] In | [Bksp] Out"
	controlsStr = runewidth.Truncate(controlsStr, w, "")
	controls := dimStyle.Render(controlsStr)

	grid := make([][]cell, gridH)
	for i := range grid {
		grid[i] = make([]cell, w)
		for j := range grid[i] {
			grid[i][j] = cell{char: ' '}
		}
	}

	draw := func(cx, cy int, r rune, style *lipgloss.Style) {
		if cx >= 0 && cx < w && cy >= 0 && cy < gridH {
			grid[cy][cx] = cell{char: r, style: style}
		}
	}

	for i, box := range boxes {
		rW := box.W - 1
		rH := box.H - 1
		if rW <= 0 || rH <= 0 {
			continue
		}

		isSel := i == selIdx
		color := fadeColor(box.LastTouched, box.ColorIndex)

		var borderStyle lipgloss.Style
		if isSel {
			borderStyle = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
		} else {
			borderStyle = lipgloss.NewStyle().Foreground(color)
		}

		hasBorder := isSel || (box.IsDir && box.W >= 3 && box.H >= 3)

		var topL, topR, botL, botR, horiz, vert rune
		if isSel {
			topL, topR, botL, botR, horiz, vert = '╔', '╗', '╚', '╝', '═', '║'
		} else {
			topL, topR, botL, botR, horiz, vert = '┌', '┐', '└', '┘', '─', '│'
		}

		bgStyle := lipgloss.NewStyle().Background(color)
		cellBorderStyle := borderStyle.Background(color)

		for cy := box.Y; cy < box.Y+rH; cy++ {
			for cx := box.X; cx < box.X+rW; cx++ {
				if hasBorder {
					isTop := cy == box.Y
					isBottom := cy == box.Y+rH-1
					isLeft := cx == box.X
					isRight := cx == box.X+rW-1

					if isTop && isLeft {
						draw(cx, cy, topL, &cellBorderStyle)
					} else if isTop && isRight {
						draw(cx, cy, topR, &cellBorderStyle)
					} else if isBottom && isLeft {
						draw(cx, cy, botL, &cellBorderStyle)
					} else if isBottom && isRight {
						draw(cx, cy, botR, &cellBorderStyle)
					} else if isTop || isBottom {
						draw(cx, cy, horiz, &cellBorderStyle)
					} else if isLeft || isRight {
						draw(cx, cy, vert, &cellBorderStyle)
					} else {
						draw(cx, cy, ' ', &bgStyle)
					}
				} else {
					draw(cx, cy, ' ', &bgStyle)
				}
			}
		}

		var availW, availH int
		var textStartX, textStartY int
		if hasBorder {
			availW = rW - 2
			availH = rH - 2
			textStartX = box.X + 1
			textStartY = box.Y + 1
		} else {
			availW = rW
			availH = rH
			textStartX = box.X
			textStartY = box.Y
		}
		if availW < 0 {
			availW = 0
		}
		if availH < 0 {
			availH = 0
		}

		var textStyle lipgloss.Style
		if isSel {
			textStyle = lipgloss.NewStyle().Background(color).Foreground(FgColor).Bold(true)
		} else if !box.LastTouched.IsZero() && time.Since(box.LastTouched).Seconds() < 2.0 {
			textStyle = lipgloss.NewStyle().Background(color).Foreground(FgColor).Bold(true)
		} else {
			textStyle = lipgloss.NewStyle().Background(color).Foreground(FgColor)
		}

		emoji := ""
		if box.IsDir {
			switch m.IconMode {
			case IconModeAscii:
				emoji = "[D] "
			case IconModeNerdFont:
				emoji = "\uf07b  "
			default:
				emoji = "📁 "
			}
		} else {
			emoji = getFileIcon(box.Name, m.IconMode)
		}

		stableW, stableH := rW, rH
		if box.IsDir {
			stableW, stableH = rW-2, rH-2
		}

		tier := getLabelTier(stableW, stableH, box.Lines)
		


		isTier2 := tier == 2
		isTier1 := tier == 1
		isTier0 := tier == 0

		innerDraw := func(cx, cy int, r rune, style *lipgloss.Style) {
			if cx >= textStartX && cx < textStartX+availW && cy >= textStartY && cy < textStartY+availH {
				draw(cx, cy, r, style)
			}
		}

		if isTier0 {
			if availW >= 3 && availH >= 1 && emoji != "" {
				emojiW := runewidth.StringWidth(emoji)
				centerX := textStartX + (availW-emojiW)/2
				centerY := textStartY + availH/2
				currentX := centerX
				for _, r := range emoji {
					rw := runewidth.RuneWidth(r)
					if currentX >= 0 && currentX < w && centerY >= 0 && centerY < gridH {
						innerDraw(currentX, centerY, r, &textStyle)
					}
					currentX++
					for i := 1; i < rw; i++ {
						if currentX >= 0 && currentX < w && centerY >= 0 && centerY < gridH {
							grid[centerY][currentX] = cell{skip: true}
						}
						currentX++
					}
				}
			}
			continue
		}

		nameStr := emoji + box.Name
		if box.IsDir {
			nameStr += "/"
		}

		if isTier1 {
			centerY := textStartY + availH/2
			displayStr := runewidth.Truncate(nameStr, stableW, "...")
			totalWidth := runewidth.StringWidth(displayStr)
			startX := textStartX + (availW-totalWidth)/2
			currentX := startX

			for _, r := range displayStr {
				rw := runewidth.RuneWidth(r)
				if currentX >= 0 && currentX < w && centerY >= 0 && centerY < gridH {
					innerDraw(currentX, centerY, r, &textStyle)
				}
				currentX++
				for i := 1; i < rw; i++ {
					if currentX >= 0 && currentX < w && centerY >= 0 && centerY < gridH {
						grid[centerY][currentX] = cell{skip: true}
					}
					currentX++
				}
			}
		}

		if isTier2 {
			line1Y := textStartY + availH/2 - 1
			line2Y := textStartY + availH/2

			displayStr := runewidth.Truncate(nameStr, stableW, "...")
			totalWidth := runewidth.StringWidth(displayStr)
			startX := textStartX + (availW-totalWidth)/2
			currentX := startX

			for _, r := range displayStr {
				rw := runewidth.RuneWidth(r)
				if currentX >= 0 && currentX < w && line1Y >= 0 && line1Y < gridH {
					innerDraw(currentX, line1Y, r, &textStyle)
				}
				currentX++
				for i := 1; i < rw; i++ {
					if currentX >= 0 && currentX < w && line1Y >= 0 && line1Y < gridH {
						grid[line1Y][currentX] = cell{skip: true}
					}
					currentX++
				}
			}

			linesLabel := fmt.Sprintf("%d L", box.Lines)
			linesWidth := runewidth.StringWidth(linesLabel)
			if linesWidth <= availW {
				startLinesX := textStartX + (availW-linesWidth)/2
				currentX = startLinesX
				for _, r := range linesLabel {
					rw := runewidth.RuneWidth(r)
					if currentX >= 0 && currentX < w && line2Y >= 0 && line2Y < gridH {
						innerDraw(currentX, line2Y, r, &textStyle)
					}
					currentX++
					for i := 1; i < rw; i++ {
						if currentX >= 0 && currentX < w && line2Y >= 0 && line2Y < gridH {
							grid[line2Y][currentX] = cell{skip: true}
						}
						currentX++
					}
				}
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(breadcrumb)
	sb.WriteRune('\n')

	defaultGridStyle := rootStyle

	for _, row := range grid {
		var currentStyle *lipgloss.Style
		var currentRun strings.Builder

		flush := func() {
			if currentRun.Len() > 0 {
				if currentStyle != nil {
					sb.WriteString(currentStyle.Render(currentRun.String()))
				} else {
					sb.WriteString(defaultGridStyle.Render(currentRun.String()))
				}
				currentRun.Reset()
			}
		}

		for _, c := range row {
			if c.skip {
				continue
			}
			if c.char == 0 {
				c.char = ' '
			}
			if c.style != currentStyle {
				flush()
				currentStyle = c.style
			}
			currentRun.WriteRune(c.char)
		}
		flush()
		sb.WriteRune('\n')
	}

	sb.WriteString(controls)
	return sb.String()
}

func fadeColor(lastTouched time.Time, index int) lipgloss.Color {
	baseColors := TreemapBaseColors
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
	
	var br, bg, bb int
	fmt.Sscanf(baseColor, "#%02x%02x%02x", &br, &bg, &bb)
	
	r := int(float64(br) + (255.0 - float64(br))*ratio)
	g := int(float64(bg) + (0.0 - float64(bg))*ratio)
	b := int(float64(bb) + (85.0 - float64(bb))*ratio)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func getFileIcon(name string, mode IconMode) string {
	if strings.HasPrefix(name, "+") {
		return ""
	}
	lower := strings.ToLower(name)
	ext := ""
	if idx := strings.LastIndex(lower, "."); idx != -1 {
		ext = lower[idx:]
	}

	if mode == IconModeAscii {
		return "[F] "
	}
	
	if mode == IconModeEmoji {
		switch ext {
		case ".go": return "🐹 "
		case ".py": return "🐍 "
		case ".rs": return "🦀 "
		case ".rb": return "💎 "
		case ".js", ".jsx": return "🟨 "
		case ".ts", ".tsx": return "🟦 "
		case ".java", ".class", ".jar": return "☕ "
		case ".cpp", ".cc", ".cxx", ".hpp", ".h", ".c": return "⚙️ "
		case ".sh", ".bash", ".zsh": return "🐚 "
		case ".php": return "🐘 "
		case ".cs": return "🎮 "
		case ".swift": return "🍎 "
		case ".kt", ".kts": return "🤖 "
		case ".html", ".htm": return "🌐 "
		case ".css", ".scss", ".sass", ".less": return "🎨 "
		case ".json", ".yaml", ".yml", ".toml", ".xml", ".ini": return "📝 "
		case ".md": return "📖 "
		case ".sql", ".db", ".sqlite": return "🗄️ "
		case ".dockerfile": return "🐳 "
		case ".txt", ".log", ".conf": return "📄 "
		}
		if lower == "dockerfile" { return "🐳 " }
		if lower == "makefile" { return "🛠️ " }
		if lower == "go.mod" || lower == "go.sum" { return "🐹 " }
		return "📄 "
	}

	// NerdFont mode
	if mode == IconModeNerdFont {
		icon := "\uf15b  " // default
		switch ext {
		case ".go": icon = "\U000f07d3  "
		case ".py": icon = "\ue73c  "
		case ".rs": icon = "\ue7a8  "
		case ".rb": icon = "\ue739  "
		case ".js", ".jsx": icon = "\ue781  "
		case ".ts", ".tsx": icon = "\ue8ca  "
		case ".java", ".class", ".jar": icon = "\ue738  "
		case ".cpp", ".cc", ".cxx", ".hpp", ".h": icon = "\ue61d  "
		case ".c": icon = "\ue61e  "
		case ".sh", ".bash", ".zsh": icon = "\ue760  "
		case ".php": icon = "\ue73d  "
		case ".cs": icon = "\ue7b2  "
		case ".swift": icon = "\ue755  "
		case ".kt", ".kts": icon = "\ue81b  "
		case ".html", ".htm": icon = "\ue736  "
		case ".css", ".scss", ".sass", ".less": icon = "\ue749  "
		case ".json", ".yaml", ".yml", ".toml", ".xml", ".ini": icon = "\ue80b  "
		case ".md": icon = "\ue73e  "
		case ".sql", ".db", ".sqlite": icon = "\ue706  "
		case ".dockerfile": icon = "\uf308  "
		case ".txt", ".log", ".conf": icon = "\uf15b  "
		}
		
		if lower == "dockerfile" { icon = "\uf308  " }
		if lower == "makefile" { icon = "\ue795  " }
		if lower == "go.mod" || lower == "go.sum" { icon = "\U000f07d3  " }
		
		return icon
	}
	
	return "\uf15b  "
}

func getLabelTier(w, h, lines int) int {
	intended := 2
	if lines < 10 {
		intended = 1
	}
	if lines == 0 {
		intended = 0
	}

	tier := intended
	area := w * h

	if tier == 2 {
		if area < 24 || w < 3 || h < 2 {
			tier = 1
		}
	}
	if tier == 1 {
		if area < 7 || w < 3 || h < 1 {
			tier = 0
		}
	}

	return tier
}
