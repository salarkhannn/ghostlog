package tui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func handleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
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
