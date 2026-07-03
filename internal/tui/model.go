package tui

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/salarkhannn/ghostlog/internal/analyzer"
	"github.com/salarkhannn/ghostlog/internal/git"
	"github.com/salarkhannn/ghostlog/internal/watcher"
)

type tickMsg time.Time

type TreemapCell struct {
	Path        string
	Lines       int
	LastTouched time.Time
}

type Model struct {
	repoPath           string
	commitCh           <-chan watcher.CommitMsg
	az                 *analyzer.Analyzer
	Bursts             []analyzer.Burst
	SelectedBurstIndex int
	AutoScroll         bool
	ViewMode           string // "burst" or "treemap"
	Treemap            []*TreemapCell
	TotalLines         int
	CPSMetric          float64
	sessionStart       time.Time
	commitTimestamps   []time.Time
	totalAdded         int
	totalRemoved       int

	vp      viewport.Model
	vpReady bool
	width   int
	height  int

	lastTouchedMap map[string]time.Time
}

func New(repoPath string, ch <-chan watcher.CommitMsg) Model {
	return Model{
		repoPath:       repoPath,
		commitCh:       ch,
		az:             analyzer.New(repoPath),
		AutoScroll:     true,
		ViewMode:       "burst",
		sessionStart:   time.Now(),
		lastTouchedMap: make(map[string]time.Time),
	}
}

type treemapMsg []*TreemapCell

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		watcher.WaitForCommit(m.commitCh),
		tickCmd(),
		m.loadTreemap(),
	)
}

func (m Model) loadTreemap() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("git", "-C", m.repoPath, "ls-files").Output()
		if err != nil {
			return treemapMsg{}
		}
		var cells []*TreemapCell
		lines := bytes.Split(out, []byte("\n"))
		for _, l := range lines {
			if len(l) == 0 {
				continue
			}
			path := string(l)
			wc, err := exec.Command("wc", "-l", filepath.Join(m.repoPath, path)).Output()
			if err != nil {
				continue
			}
			var count int
			fmt.Sscanf(string(wc), "%d", &count)
			if count > 0 {
				cells = append(cells, &TreemapCell{Path: path, Lines: count})
			}
		}
		return treemapMsg(cells)
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case treemapMsg:
		m.Treemap = msg
		m.TotalLines = 0
		for _, c := range m.Treemap {
			if m.lastTouchedMap != nil {
				if t, ok := m.lastTouchedMap[c.Path]; ok {
					c.LastTouched = t
				}
			}
			m.TotalLines += c.Lines
		}
		return m, nil

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
			if m.lastTouchedMap == nil {
				m.lastTouchedMap = make(map[string]time.Time)
			}
			for _, d := range info.Diffs {
				m.totalAdded += d.LinesAdded
				m.totalRemoved += d.LinesRemoved
				m.lastTouchedMap[d.Path] = time.Now()
				for _, cell := range m.Treemap {
					if cell.Path == d.Path {
						cell.LastTouched = time.Now()
					}
				}
			}
			if m.AutoScroll {
				m.SelectedBurstIndex = len(m.Bursts) - 1
			}
			m.refreshViewport()
			return m, tea.Batch(
				watcher.WaitForCommit(m.commitCh),
				m.loadTreemap(),
			)
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

func (m Model) Verdict() string {
	spikes := 0
	untested := 0
	for _, b := range m.Bursts {
		if (b.ComplexityAfter - b.ComplexityBefore) > 10 {
			spikes++
		}
		if len(b.UntestedFunctions) > 0 {
			untested++
		}
	}
	dur := m.sessionStart
	elapsed := time.Since(dur).Round(time.Second)
	min := int(elapsed.Minutes()) % 60
	s := int(elapsed.Seconds()) % 60
	return fmt.Sprintf("session: %d bursts, %d complexity spikes, %d untested, duration %02d:%02d", len(m.Bursts), spikes, untested, min, s)
}
