package tui

import (
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func handleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.ViewMode == "treemap" {
		items := m.getGroupedItems()

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "v":
			m.ViewMode = "burst"
			return m, nil

		case "j", "down", "tab":
			if len(items) > 0 {
				m.SelectedTreemapIndex = (m.SelectedTreemapIndex + 1) % len(items)
			}
			return m, nil

		case "k", "up", "shift+tab":
			if len(items) > 0 {
				m.SelectedTreemapIndex = (m.SelectedTreemapIndex - 1 + len(items)) % len(items)
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

		case "backspace", "left":
			if m.CurrentDir != "" {
				parent := filepath.Dir(m.CurrentDir)
				if parent == "." || parent == "/" || parent == m.CurrentDir {
					parent = ""
				}
				oldDir := m.CurrentDir
				m.CurrentDir = parent

				parentItems := m.getGroupedItems()
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
