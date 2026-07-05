package tui

import (
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func findSpatialNeighbor(m Model, currentIdx int, dir string) int {
	w, h := m.TreemapDims()
	items, boxes := m.layoutTreemap(w, h)
	if len(items) == 0 {
		return currentIdx
	}
	if currentIdx < 0 || currentIdx >= len(items) {
		return 0
	}

	curr := boxes[currentIdx]
	cx := float64(curr.X) + float64(curr.W)/2.0
	cy := float64(curr.Y) + float64(curr.H)/2.0

	for _, threshold := range []float64{1.5, 0.1} {
		bestIdx := -1
		minScore := math.MaxFloat64

		for i, b := range boxes {
			if i == currentIdx {
				continue
			}
			bx := float64(b.X) + float64(b.W)/2.0
			by := float64(b.Y) + float64(b.H)/2.0

			dx := bx - cx
			dy := by - cy

			inDirection := false
			var score float64

			switch dir {
			case "right":
				if dx > 0.1 && math.Abs(dx) > math.Abs(dy)*threshold {
					inDirection = true
					score = dx*dx + 4.0*dy*dy
				}
			case "left":
				if dx < -0.1 && math.Abs(dx) > math.Abs(dy)*threshold {
					inDirection = true
					score = dx*dx + 4.0*dy*dy
				}
			case "down":
				if dy > 0.1 && math.Abs(dy) > math.Abs(dx)*threshold {
					inDirection = true
					score = 4.0*dx*dx + dy*dy
				}
			case "up":
				if dy < -0.1 && math.Abs(dy) > math.Abs(dx)*threshold {
					inDirection = true
					score = 4.0*dx*dx + dy*dy
				}
			}

			if inDirection && score < minScore {
				minScore = score
				bestIdx = i
			}
		}

		if bestIdx != -1 {
			return bestIdx
		}
	}

	return currentIdx
}

func handleMouse(m Model, msg tea.MouseMsg) (Model, tea.Cmd) {
	if m.ViewMode == "burst" {
		var cmd tea.Cmd
		var listCmd tea.Cmd
		
		leftW := m.width * 70 / 100
		if msg.X < leftW {
			oldIdx := m.burstList.Index()
			m.burstList, listCmd = m.burstList.Update(msg)
			if m.burstList.Index() != oldIdx {
				m.AutoScroll = false
				m.vp.GotoTop()
				m.refreshViewport()
			}
		} else {
			oldY := m.vp.YOffset
			m.vp, cmd = m.vp.Update(msg)
			if m.vp.YOffset < oldY {
				m.AutoScroll = false
			}
		}
		
		return m, tea.Batch(cmd, listCmd)
	}

	if m.ViewMode != "treemap" {
		return m, nil
	}

	isClick := msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress
	isHover := msg.Action == tea.MouseActionMotion || msg.Button == tea.MouseButtonNone
	if !isClick && !isHover {
		return m, nil
	}

	w, h := m.TreemapDims()

	// Click on Breadcrumb to go up/zoom out (Y=1 with no top border)
	if msg.Y == 1 && msg.X >= 0 && msg.X < w {
		if isClick && m.CurrentDir != "" {
			parent := filepath.Dir(m.CurrentDir)
			if parent == "." || parent == "/" || parent == m.CurrentDir {
				parent = ""
			}
			oldDir := m.CurrentDir
			m.CurrentDir = parent

			parentItems, _ := m.getLayout(w, h)
			m.SelectedTreemapIndex = 0
			for idx, pi := range parentItems {
				if pi.path == oldDir {
					m.SelectedTreemapIndex = idx
					break
				}
			}
		}
		return m, nil
	}

	// Interaction inside grid (starts at Y=2 with no top border, X=1 with left padding=1)
	gridX := msg.X - 1
	gridY := msg.Y - 2
	if gridX >= 0 && gridX < w && gridY >= 0 && gridY < h {
		_, boxes := m.getLayout(w, h)
		for i, box := range boxes {
			if gridX >= box.X && gridX < box.X+box.W && gridY >= box.Y && gridY < box.Y+box.H {
				if isClick {
					if m.SelectedTreemapIndex == i {
						if box.IsDir {
							m.CurrentDir = box.Path
							m.SelectedTreemapIndex = 0
						}
					} else {
						m.SelectedTreemapIndex = i
					}
				} else if isHover {
					m.SelectedTreemapIndex = i
				}
				break
			}
		}
	}

	return m, nil
}

func handleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Type == tea.KeyTab {
		if m.FocusPane == "list" {
			m.FocusPane = "diff"
		} else {
			m.FocusPane = "list"
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		if m.quitting && time.Since(m.quitTime) < 3*time.Second {
			return m, tea.Quit
		}
		m.quitting = true
		m.quitTime = time.Now()
		return m, nil
	case "v":
		if m.ViewMode == "treemap" {
			m.ViewMode = "burst"
		} else {
			m.ViewMode = "treemap"
		}
		return m, nil
	case "[", "<", "p":
		m.burstList.CursorUp()
		m.AutoScroll = false
		m.vp.GotoTop()
		m.refreshViewport()
		return m, nil
	case "]", ">", "n":
		m.burstList.CursorDown()
		m.AutoScroll = false
		m.vp.GotoTop()
		m.refreshViewport()
		return m, nil
	case "s":
		if m.ViewMode != "sessions" {
			m.ViewMode = "sessions"
			files, _ := filepath.Glob(filepath.Join(m.repoPath, ".ghostlog", "sessions", "*.txt"))
			var items []list.Item
			for _, f := range files {
				items = append(items, sessionItem{filepath.Base(f)})
			}
			m.sessionList.SetItems(items)
			return m, nil
		}
	}

	if m.FocusPane == "diff" && m.ViewMode != "sessions" {
		oldY := m.vp.YOffset
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		if m.vp.YOffset < oldY {
			m.AutoScroll = false
		}
		return m, cmd
	}

	if m.ViewMode == "sessions" {
		switch msg.Type {
		case tea.KeyEsc:
			m.ViewMode = "burst"
			return m, nil
		case tea.KeyEnter:
			idx := m.sessionList.Index()
			items := m.sessionList.Items()
			if idx >= 0 && idx < len(items) {
				selected := items[idx].(sessionItem).filename
				m.loadSession(selected)
				m.ViewMode = "burst"
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.sessionList, cmd = m.sessionList.Update(msg)
		return m, cmd
	}

	if m.ViewMode == "treemap" {
		w, h := m.TreemapDims()
		items, _ := m.getLayout(w, h)

		switch msg.Type {
		case tea.KeyEnter:
			if len(items) > 0 && m.SelectedTreemapIndex >= 0 && m.SelectedTreemapIndex < len(items) {
				sel := items[m.SelectedTreemapIndex]
				if sel.isDir {
					m.CurrentDir = sel.path
					m.SelectedTreemapIndex = 0
				}
			}
			return m, nil
		case tea.KeyBackspace:
			if m.CurrentDir != "" {
				parent := filepath.Dir(m.CurrentDir)
				if parent == "." || parent == "/" || parent == m.CurrentDir {
					parent = ""
				}
				oldDir := m.CurrentDir
				m.CurrentDir = parent
				parentItems, _ := m.layoutTreemap(w, h)
				m.SelectedTreemapIndex = 0
				for idx, pi := range parentItems {
					if pi.path == oldDir {
						m.SelectedTreemapIndex = idx
						break
					}
				}
			}
			return m, nil
		case tea.KeyLeft:
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "left")
			return m, nil
		case tea.KeyRight:
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "right")
			return m, nil
		case tea.KeyUp:
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "up")
			return m, nil
		case tea.KeyDown:
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "down")
			return m, nil
		}

		switch msg.String() {
		case "j", "down":
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "down")
			return m, nil
		case "k", "up":
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "up")
			return m, nil
		case "a", "left":
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "left")
			return m, nil
		case "d", "right":
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "right")
			return m, nil
		case "l":
			if len(items) > 0 && m.SelectedTreemapIndex >= 0 && m.SelectedTreemapIndex < len(items) {
				sel := items[m.SelectedTreemapIndex]
				if sel.isDir {
					m.CurrentDir = sel.path
					m.SelectedTreemapIndex = 0
				}
			}
			return m, nil
		case "h":
			if m.CurrentDir != "" {
				parent := filepath.Dir(m.CurrentDir)
				if parent == "." || parent == "/" || parent == m.CurrentDir {
					parent = ""
				}
				oldDir := m.CurrentDir
				m.CurrentDir = parent
				parentItems, _ := m.layoutTreemap(w, h)
				m.SelectedTreemapIndex = 0
				for idx, pi := range parentItems {
					if pi.path == oldDir {
						m.SelectedTreemapIndex = idx
						break
					}
				}
			}
			return m, nil
		case "enter", "\n", "\r":
			if len(items) > 0 && m.SelectedTreemapIndex >= 0 && m.SelectedTreemapIndex < len(items) {
				sel := items[m.SelectedTreemapIndex]
				if sel.isDir {
					m.CurrentDir = sel.path
					m.SelectedTreemapIndex = 0
				}
			}
			return m, nil
		case "backspace":
			if m.CurrentDir != "" {
				parent := filepath.Dir(m.CurrentDir)
				if parent == "." || parent == "/" || parent == m.CurrentDir {
					parent = ""
				}
				oldDir := m.CurrentDir
				m.CurrentDir = parent
				parentItems, _ := m.layoutTreemap(w, h)
				m.SelectedTreemapIndex = 0
				for idx, pi := range parentItems {
					if pi.path == oldDir {
						m.SelectedTreemapIndex = idx
						break
					}
				}
			}
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "a":
		m.AutoScroll = !m.AutoScroll
		if m.AutoScroll && len(m.Bursts) > 0 {
			m.burstList.Select(len(m.Bursts) - 1)
			m.refreshViewport()
		}
		return m, nil
	case "c":
		idx := m.burstList.Index()
		if idx >= 0 && idx < len(m.Bursts) {
			copyToClipboard(strings.Join(m.Bursts[idx].Hashes, " "))
		}
		return m, nil
	}

	var listCmd tea.Cmd
	oldIdx := m.burstList.Index()
	m.burstList, listCmd = m.burstList.Update(msg)
	if m.burstList.Index() != oldIdx {
		m.AutoScroll = false
		m.vp.GotoTop()
		m.refreshViewport()
	}
	return m, listCmd
}

func copyToClipboard(s string) {
	clipboard.WriteAll(s)
}
