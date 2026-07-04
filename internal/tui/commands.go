package tui

import (
	"math"
	"os/exec"
	"path/filepath"
	"strings"

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
	if m.ViewMode == "treemap" {
		w, h := m.TreemapDims()
		items, _ := m.getLayout(w, h)

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyTab:
			if len(items) > 0 {
				m.SelectedTreemapIndex = (m.SelectedTreemapIndex + 1) % len(items)
			}
			return m, nil

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

				w, h := m.TreemapDims()
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

		// Handle key runes/strings fallback
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "v":
			m.ViewMode = "burst"
			return m, nil
		case "j", "down":
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "down")
			return m, nil
		case "k", "up":
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "up")
			return m, nil
		case "l": // Zoom in
			if len(items) > 0 && m.SelectedTreemapIndex >= 0 && m.SelectedTreemapIndex < len(items) {
				sel := items[m.SelectedTreemapIndex]
				if sel.isDir {
					m.CurrentDir = sel.path
					m.SelectedTreemapIndex = 0
				}
			}
			return m, nil
		case "h": // Zoom out
			if m.CurrentDir != "" {
				parent := filepath.Dir(m.CurrentDir)
				if parent == "." || parent == "/" || parent == m.CurrentDir {
					parent = ""
				}
				oldDir := m.CurrentDir
				m.CurrentDir = parent

				w, h := m.TreemapDims()
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
		case "a", "left":
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "left")
			return m, nil
		case "d", "right":
			m.SelectedTreemapIndex = findSpatialNeighbor(m, m.SelectedTreemapIndex, "right")
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

				w, h := m.TreemapDims()
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
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.SelectedBurstIndex < len(m.Bursts)-1 {
			m.SelectedBurstIndex++
			m.AutoScroll = false
			m.vp.GotoTop()
			m.refreshViewport()
		}

	case "k", "up":
		if m.SelectedBurstIndex > 0 {
			m.SelectedBurstIndex--
			m.AutoScroll = false
			m.vp.GotoTop()
			m.refreshViewport()
		}

	case "a":
		m.AutoScroll = !m.AutoScroll
		if m.AutoScroll && len(m.Bursts) > 0 {
			m.SelectedBurstIndex = len(m.Bursts) - 1
			m.refreshViewport()
		}

	case "c":
		if m.SelectedBurstIndex >= 0 && m.SelectedBurstIndex < len(m.Bursts) {
			copyToClipboard(strings.Join(m.Bursts[m.SelectedBurstIndex].Hashes, " "))
		}

	case "v":
		if m.ViewMode == "treemap" {
			m.ViewMode = "burst"
		} else {
			m.ViewMode = "treemap"
		}

	default:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}

	return m, nil
}

func copyToClipboard(s string) {
	for _, args := range [][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"wl-copy"},
	} {
		c := exec.Command(args[0], args[1:]...)
		c.Stdin = strings.NewReader(s)
		if c.Run() == nil {
			return
		}
	}
}
