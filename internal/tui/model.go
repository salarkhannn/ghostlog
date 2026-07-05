package tui

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/salarkhannn/ghostlog/internal/tui/formatting"
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

type IconMode string

const (
	IconModeEmoji    IconMode = "emoji"
	IconModeNerdFont IconMode = "nerd"
	IconModeAscii    IconMode = "none"
)

type LayoutCache struct {
	w, h        int
	dir         string
	commitCnt   int
	boxes       []PositionedBox
	parentItems []*treemapItem
}


type burstItem struct {
	burst analyzer.Burst
	idx   int
}

func (i burstItem) Title() string {
	if len(i.burst.Hashes) > 0 && i.burst.Hashes[0] == "UNCOMMITTED" {
		return "[NOT COMMITTED] Uncommitted Changes"
	}
	dur := formatting.Duration(int64(i.burst.LastTime.Sub(i.burst.StartTime).Seconds()))
	if dur == "0s" {
		dur = "<1s"
	}
	return fmt.Sprintf("[#%d] %d commits in %s", i.burst.ID, len(i.burst.Hashes), dur)
}

func (i burstItem) Description() string {
	status := "OK"
	if len(i.burst.SecretLeaks) > 0 {
		status = "WARN-leak"
	} else if i.burst.HasConflict || (i.burst.ComplexityAfter-i.burst.ComplexityBefore) > 10 || len(i.burst.UntestedFunctions) > 0 {
		status = "WARN"
	}
	return fmt.Sprintf("%s | +%d -%d (%s) files:%d", status, i.burst.LinesAdded, i.burst.LinesRemoved, formatting.Bytes(i.burst.BytesAdded), i.burst.FilesChanged)
}

func (i burstItem) FilterValue() string {
	return fmt.Sprintf("#%d", i.burst.ID)
}

type sessionItem struct {
	filename string
}

func (i sessionItem) Title() string       { return strings.TrimSuffix(i.filename, ".txt") }
func (i sessionItem) Description() string { return "Ghostlog Session" }
func (i sessionItem) FilterValue() string { return i.filename }

type Model struct {
	repoPath           string
	commitCh           <-chan watcher.CommitMsg
	az                 *analyzer.Analyzer
	Bursts             []analyzer.Burst
	burstList          list.Model
	burstListReady     bool
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

	lastTouchedMap       map[string]time.Time
	CurrentDir           string
	SelectedTreemapIndex int
	layoutCache          *LayoutCache
	IconMode             IconMode
	ActiveSessionID      string
	ViewSessionID        string
	sessionList          list.Model
	FocusPane            string // "list" or "diff"
	uncommittedBurst     *analyzer.Burst
}

func New(repoPath string, ch <-chan watcher.CommitMsg, iconMode IconMode) Model {
	if iconMode == "" {
		iconMode = IconModeNerdFont
	}
	m := Model{
		repoPath:       repoPath,
		commitCh:       ch,
		az:             analyzer.New(repoPath),
		AutoScroll:     true,
		ViewMode:       "burst",
		sessionStart:   time.Now(),
		layoutCache:    &LayoutCache{},
		lastTouchedMap: make(map[string]time.Time),
		IconMode:       iconMode,
	}
	
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(PrimaryColor).BorderLeftForeground(PrimaryColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(PrimaryColor).BorderLeftForeground(PrimaryColor)
	
	m.burstList = list.New([]list.Item{}, delegate, 0, 0)
	m.burstList.Title = "Bursts"
	m.burstList.Styles.Title = AccentStyle
	m.burstList.SetShowStatusBar(false)
	m.burstList.SetFilteringEnabled(true)
	
	m.ActiveSessionID = "session-" + time.Now().Format("20060102-150405")
	m.ViewSessionID = m.ActiveSessionID
	m.FocusPane = "list"
	os.MkdirAll(filepath.Join(repoPath, ".ghostlog", "sessions"), 0755)

	m.sessionList = list.New([]list.Item{}, delegate, 0, 0)
	m.sessionList.Title = "Sessions"
	m.sessionList.Styles.Title = AccentStyle
	m.sessionList.SetShowStatusBar(false)
	
	return m
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

type uncommittedMsg *git.CommitInfo

func (m Model) checkUncommitted() tea.Cmd {
	return func() tea.Msg {
		info, _ := git.DiffUncommitted(m.repoPath)
		return uncommittedMsg(info)
	}
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
		
		leftW := m.width * 70 / 100
		rightW := m.width - leftW - 2
		
		contentH := m.height - 2 // Account for top/bottom bars
		
		if !m.vpReady {
			os.Stdout.WriteString("\x1b[?7l")
			m.vp = viewport.New(rightW-2, contentH-2)
			m.vp.Style = ViewportStyle
			m.vpReady = true
		} else {
			m.vp.Width = rightW - 2
			m.vp.Height = contentH - 2
		}
		
		// burstList needs leftW-4 because LeftPaneStyle has Padding(0, 1) and Border
		m.burstList.SetSize(leftW-4, contentH-2)
		m.burstListReady = true
		
		m.sessionList.SetSize(leftW-4, contentH-2)

		m.refreshViewport()
		return m, nil

	case tea.KeyMsg:
		return handleKey(m, msg)

	case tea.MouseMsg:
		return handleMouse(m, msg)

	case tickMsg:
		m.CPSMetric = m.calcCPM()
		return m, tea.Batch(tickCmd(), m.checkUncommitted())

	case uncommittedMsg:
		if msg != nil && len(msg.Diffs) > 0 {
			b := analyzer.Burst{
				ID:           len(m.Bursts) + 1,
				Hashes:       []string{"UNCOMMITTED"},
				StartTime:    time.Now(),
				LastTime:     time.Now(),
				FilesChanged: len(msg.Diffs),
			}
			for _, d := range msg.Diffs {
				b.LinesAdded += d.LinesAdded
				b.LinesRemoved += d.LinesRemoved
				b.BytesAdded += d.Bytes
				b.SecretLeaks = append(b.SecretLeaks, d.SecretLeaks...)
			}
			m.uncommittedBurst = &b
		} else {
			m.uncommittedBurst = nil
		}
		m.refreshBurstListItems()
		return m, nil

	case watcher.CommitMsg:
		sessionPath := filepath.Join(m.repoPath, ".ghostlog", "sessions", m.ActiveSessionID+".txt")
		if f, err := os.OpenFile(sessionPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			f.WriteString(msg.Hash + "\n")
			f.Close()
		}

		if m.ViewSessionID != m.ActiveSessionID {
			return m, watcher.WaitForCommit(m.commitCh)
		}

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
			
			m.refreshBurstListItems()
			
			if m.AutoScroll {
				m.burstList.Select(len(m.Bursts) - 1)
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

func (m *Model) refreshBurstListItems() {
	var items []list.Item
	for i, b := range m.Bursts {
		items = append(items, burstItem{burst: b, idx: i})
	}
	if m.uncommittedBurst != nil {
		items = append(items, burstItem{burst: *m.uncommittedBurst, idx: len(m.Bursts)})
	}
	m.burstList.SetItems(items)
	m.refreshViewport()
}

func (m *Model) refreshViewport() {
	if !m.vpReady {
		return
	}
	if len(m.Bursts) == 0 {
		content := "\n" + dimStyle.Render("  Waiting for first commit...")
		m.vp.SetContent(ViewportStyle.Width(m.vp.Width).Render(content))
		return
	}
	idx := m.burstList.Index()
	items := m.burstList.Items()
	if idx < 0 || idx >= len(items) {
		content := "\n" + dimStyle.Render("  No burst selected.")
		m.vp.SetContent(ViewportStyle.Width(m.vp.Width).Render(content))
		return
	}
	
	oldY := m.vp.YOffset
	
	bi := items[idx].(burstItem)
	if len(bi.burst.Hashes) > 0 && bi.burst.Hashes[0] == "UNCOMMITTED" {
		diff, err := git.ShowUncommitted(m.repoPath)
		if err != nil {
			m.vp.SetContent("error: " + err.Error())
		} else {
			m.vp.SetContent(diff)
		}
	} else {
		diff, err := git.Show(m.repoPath, bi.burst.Hashes)
		if err != nil {
			m.vp.SetContent("error: " + err.Error())
			return
		}
		m.vp.SetContent(diff)
	}
	if m.AutoScroll {
		m.vp.GotoBottom()
	} else {
		m.vp.SetYOffset(oldY)
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
	leftW := totalW * 70 / 100
	rightW := totalW - leftW
	w = rightW - 4
	h = totalH - 6
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return
}

func (m Model) Verdict() string {
	spikes := 0
	untested := 0
	leaks := 0
	for _, b := range m.Bursts {
		if (b.ComplexityAfter - b.ComplexityBefore) > 10 {
			spikes++
		}
		if len(b.UntestedFunctions) > 0 {
			untested++
		}
		if len(b.SecretLeaks) > 0 {
			leaks++
		}
	}
	dur := m.sessionStart
	elapsed := time.Since(dur).Round(time.Second)
	min := int(elapsed.Minutes()) % 60
	s := int(elapsed.Seconds()) % 60
	return fmt.Sprintf("session: %d bursts, %d complexity spikes, %d untested, %d secret leaks, duration %02d:%02d", len(m.Bursts), spikes, untested, leaks, min, s)
}

func (m Model) TreemapDims() (w, h int) {
	leftW := m.width * 70 / 100
	contentH := m.height - 2
	w = leftW - 2
	h = contentH - 2
	if w < 1 {
		w = 1
	}
	if h < 3 {
		h = 3
	}
	return w, h
}

func getAvailDims(box PositionedBox) (availW, availH int) {
	hasBorder := box.IsDir
	rW := box.W - 1
	rH := box.H - 1
	if hasBorder {
		availW = rW - 2
		availH = rH - 2
	} else {
		availW = rW
		availH = rH
	}
	if availW < 0 {
		availW = 0
	}
	if availH < 0 {
		availH = 0
	}
	return
}

func (m Model) getLayout(w, h int) ([]*treemapItem, []PositionedBox) {
	if m.layoutCache == nil {
		m.layoutCache = &LayoutCache{}
	}
	if m.layoutCache.boxes != nil && m.layoutCache.w == w && m.layoutCache.h == h && m.layoutCache.dir == m.CurrentDir && m.layoutCache.commitCnt == len(m.commitTimestamps) {
		return m.layoutCache.parentItems, m.layoutCache.boxes
	}
	items, boxes := m.layoutTreemap(w, h)
	m.layoutCache.w = w
	m.layoutCache.h = h
	m.layoutCache.dir = m.CurrentDir
	m.layoutCache.commitCnt = len(m.commitTimestamps)
	m.layoutCache.boxes = boxes
	m.layoutCache.parentItems = items
	return items, boxes
}

func (m Model) layoutTreemap(w, h int) ([]*treemapItem, []PositionedBox) {
	if w <= 0 || h <= 0 {
		return nil, nil
	}
	allGroups := m.getAllGroupedItems()
	if len(allGroups) == 0 {
		return nil, nil
	}

	zoomDepth := 0
	actualDir := m.CurrentDir
	for strings.HasSuffix(actualDir, "/?others") {
		zoomDepth++
		actualDir = strings.TrimSuffix(actualDir, "/?others")
	}
	if actualDir == "?others" {
		zoomDepth++
	}

	for {
		K := len(allGroups)
		for K > 0 {
			var currentItems []*treemapItem
			if K == len(allGroups) {
				currentItems = allGroups
			} else {
				currentItems = make([]*treemapItem, 0, K)
				currentItems = append(currentItems, allGroups[:K-1]...)
				
				otherLines := 0
				var otherTouched time.Time
				for _, it := range allGroups[K-1:] {
					otherLines += it.lines
					if it.lastTouched.After(otherTouched) {
						otherTouched = it.lastTouched
					}
				}
				if otherLines > 0 {
					path := "?others"
					if m.CurrentDir != "" {
						path = m.CurrentDir + "/?others"
					}
					currentItems = append(currentItems, &treemapItem{
						path:        path,
						name:        fmt.Sprintf("+%d others", len(allGroups)-K+1),
						isDir:       true,
						lines:       otherLines,
						lastTouched: otherTouched,
					})
				}
			}

			sort.Slice(currentItems, func(i, j int) bool {
				wi := math.Log2(float64(currentItems[i].lines) + 1.0)
				if wi < 1.0 {
					wi = 1.0
				}
				wj := math.Log2(float64(currentItems[j].lines) + 1.0)
				if wj < 1.0 {
					wj = 1.0
				}
				if wi == wj {
					if currentItems[i].isDir != currentItems[j].isDir {
						return currentItems[i].isDir
					}
					return currentItems[i].name < currentItems[j].name
				}
				return wi > wj
			})

			weights := make([]TreemapItemWeight, len(currentItems))
			for i, item := range currentItems {
				weight := math.Log2(float64(item.lines) + 1.0)
				if weight < 1.0 {
					weight = 1.0
				}
				weights[i] = TreemapItemWeight{
					Path:        item.path,
					Name:        item.name,
					IsDir:       item.isDir,
					Lines:       item.lines,
					LastTouched: item.lastTouched,
					ColorIndex:  i,
					Weight:      weight,
				}
			}

			boxes := Squarify(weights, Rect{X: 0, Y: 0, W: w, H: h})

			if K <= 1 || len(boxes) <= 1 {
				return currentItems, boxes
			}

			var totalWeight float64
			for _, wt := range weights {
				totalWeight += wt.Weight
			}

			smallCount := 0
			for _, wt := range weights {
				if wt.Weight < totalWeight*0.15 {
					smallCount++
				}
			}

			if smallCount <= 4 {
				if zoomDepth > 0 && K < len(allGroups) {
					allGroups = allGroups[K-1:]
					zoomDepth--
					break
				}
				return currentItems, boxes
			}
			K--
		}
		if K <= 0 {
			break
		}
	}

	return nil, nil
}

func (m Model) getAllGroupedItems() []*treemapItem {
	groups := make(map[string]*treemapItem)
	
	actualDir := m.CurrentDir
	
	// Strip all occurrences of ?others from actualDir for file matching
	actualDir = strings.ReplaceAll(actualDir, "?others/", "")
	actualDir = strings.ReplaceAll(actualDir, "/?others", "")
	if actualDir == "?others" {
		actualDir = ""
	}

	prefix := actualDir
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
			// Keep the virtual zoom history in the new path!
			fullChildPath = m.CurrentDir + "/" + childName
			// Clean it up just in case (e.g. root case)
			if m.CurrentDir == "" {
				fullChildPath = childName
			}
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
		if items[i].lines == items[j].lines {
			if items[i].isDir != items[j].isDir {
				return items[i].isDir
			}
			return items[i].name < items[j].name
		}
		return items[i].lines > items[j].lines
	})

	return items
}

func (m *Model) loadSession(filename string) {
	m.ViewSessionID = strings.TrimSuffix(filename, ".txt")
	m.az = analyzer.New(m.repoPath)
	m.Bursts = nil
	m.totalAdded = 0
	m.totalRemoved = 0
	m.commitTimestamps = nil
	m.lastTouchedMap = make(map[string]time.Time)
	m.burstList.SetItems(nil)

	b, err := os.ReadFile(filepath.Join(m.repoPath, ".ghostlog", "sessions", filename))
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(b)), "\n")
		for _, hash := range lines {
			if hash == "" {
				continue
			}
			info, err := git.DiffTree(m.repoPath, hash)
			if err == nil {
				bursts, changed := m.az.Add(info)
				if changed {
					m.Bursts = bursts
					m.commitTimestamps = append(m.commitTimestamps, time.Now())
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
				}
			}
		}
	}

	m.refreshBurstListItems()
}
