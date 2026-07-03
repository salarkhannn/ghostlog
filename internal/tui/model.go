package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
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
	AutoScroll         bool
	CPSMetric          float64
	sessionStart       time.Time
	commitTimestamps   []time.Time
	totalAdded         int
	totalRemoved       int

	vp      viewport.Model
	vpReady bool
	width   int
	height  int
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
	return tea.Batch(watcher.WaitForCommit(m.commitCh), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpW, vpH := rightPaneDims(m.width, m.height)
		if !m.vpReady {
			m.vp = viewport.New(vpW, vpH)
			m.vpReady = true
		} else {
			m.vp.Width = vpW
			m.vp.Height = vpH
		}
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
			m.commitTimestamps = append(m.commitTimestamps, time.Now())
			for _, d := range info.Diffs {
				m.totalAdded += d.LinesAdded
				m.totalRemoved += d.LinesRemoved
			}
			if m.AutoScroll {
				m.SelectedBurstIndex = len(m.Bursts) - 1
			}
			m.refreshViewport()
		}
		return m, watcher.WaitForCommit(m.commitCh)
	}
	return m, nil
}

func (m *Model) refreshViewport() {
	if !m.vpReady || len(m.Bursts) == 0 {
		return
	}
	if m.SelectedBurstIndex < 0 || m.SelectedBurstIndex >= len(m.Bursts) {
		return
	}
	diff, err := git.Show(m.repoPath, m.Bursts[m.SelectedBurstIndex].Hashes)
	if err != nil {
		m.vp.SetContent("error: " + err.Error())
		return
	}
	m.vp.SetContent(diff)
	if m.AutoScroll {
		m.vp.GotoBottom()
	}
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

func rightPaneDims(totalW, totalH int) (w, h int) {
	w = totalW*60/100 - 4
	h = totalH - 4
	return
}
