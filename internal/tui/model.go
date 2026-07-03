package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/salarkhannn/ghostlog/internal/analyzer"
	"github.com/salarkhannn/ghostlog/internal/git"
	"github.com/salarkhannn/ghostlog/internal/watcher"
)

type tickMsg time.Time

type Model struct {
	repoPath           string
	commitCh           <-chan watcher.CommitMsg
	az                 *analyzer.Analyzer
	Bursts             []analyzer.Burst
	SelectedBurstIndex int
	ScrollOffset       int
	AutoScroll         bool
	CPSMetric          float64
	sessionStart       time.Time
	commitTimestamps   []time.Time
	selectedDiff       string
	width              int
	height             int
	totalAdded         int
	totalRemoved       int
}

func New(repoPath string, ch <-chan watcher.CommitMsg) Model {
	return Model{
		repoPath:     repoPath,
		commitCh:     ch,
		az:           analyzer.New(),
		AutoScroll:   true,
		sessionStart: time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		watcher.WaitForCommit(m.commitCh),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return handleKey(m, msg)

	case tickMsg:
		m.CPSMetric = m.calcCPM()
		return m, tickCmd()

	case watcher.CommitMsg:
		info, err := git.DiffTree(msg.RepoPath, msg.Hash)
		if err != nil {
			return m, watcher.WaitForCommit(m.commitCh)
		}
		bursts, changed := m.az.Add(info)
		if changed {
			m.Bursts = bursts
			now := time.Now()
			m.commitTimestamps = append(m.commitTimestamps, now)
			for _, d := range info.Diffs {
				m.totalAdded += d.LinesAdded
				m.totalRemoved += d.LinesRemoved
			}
			if m.AutoScroll {
				m.SelectedBurstIndex = len(m.Bursts) - 1
			}
			m.selectedDiff = loadDiff(m.repoPath, m.Bursts, m.SelectedBurstIndex)
		}
		return m, watcher.WaitForCommit(m.commitCh)
	}
	return m, nil
}

func (m Model) calcCPM() float64 {
	cutoff := time.Now().Add(-60 * time.Second)
	n := 0
	for _, ts := range m.commitTimestamps {
		if ts.After(cutoff) {
			n++
		}
	}
	return float64(n)
}

func loadDiff(repoPath string, bursts []analyzer.Burst, idx int) string {
	if len(bursts) == 0 || idx < 0 || idx >= len(bursts) {
		return ""
	}
	diff, err := git.Show(repoPath, bursts[idx].Hashes)
	if err != nil {
		return "error loading diff: " + err.Error()
	}
	return diff
}
