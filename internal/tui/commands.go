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
			m.ScrollOffset = 0
			m.selectedDiff = loadDiff(m.repoPath, m.Bursts, m.SelectedBurstIndex)
		}

	case "k", "up":
		if m.SelectedBurstIndex > 0 {
			m.SelectedBurstIndex--
			m.AutoScroll = false
			m.ScrollOffset = 0
			m.selectedDiff = loadDiff(m.repoPath, m.Bursts, m.SelectedBurstIndex)
		}

	case "pgdown", "ctrl+d":
		m.ScrollOffset += 20

	case "pgup", "ctrl+u":
		m.ScrollOffset -= 20
		if m.ScrollOffset < 0 {
			m.ScrollOffset = 0
		}

	case "a":
		m.AutoScroll = !m.AutoScroll
		if m.AutoScroll && len(m.Bursts) > 0 {
			m.SelectedBurstIndex = len(m.Bursts) - 1
			m.ScrollOffset = 0
			m.selectedDiff = loadDiff(m.repoPath, m.Bursts, m.SelectedBurstIndex)
		}

	case "c":
		if m.SelectedBurstIndex >= 0 && m.SelectedBurstIndex < len(m.Bursts) {
			hashes := m.Bursts[m.SelectedBurstIndex].Hashes
			copyToClipboard(strings.Join(hashes, " "))
		}
	}

	return m, nil
}

func copyToClipboard(s string) {
	for _, cmd := range [][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"wl-copy"},
	} {
		c := exec.Command(cmd[0], cmd[1:]...) //nolint
		c.Stdin = strings.NewReader(s)
		if err := c.Run(); err == nil {
			return
		}
	}
}
