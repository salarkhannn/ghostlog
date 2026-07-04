package tui

import (
	"fmt"
	"testing"
	"time"
)

func TestRenderTUI(t *testing.T) {
	m := New(".", nil)
	m.ViewMode = "treemap"
	m.width = 80
	m.height = 24
	m.vpReady = true

	cells := []*TreemapCell{
		{Path: "main.go", Lines: 120, LastTouched: time.Now()},
		{Path: "check.go", Lines: 65, LastTouched: time.Now()},
		{Path: "export.go", Lines: 89, LastTouched: time.Now()},
		{Path: "internal/tui/view.go", Lines: 409, LastTouched: time.Now()},
		{Path: "internal/tui/model.go", Lines: 393, LastTouched: time.Now()},
		{Path: "internal/tui/commands.go", Lines: 344, LastTouched: time.Now()},
		{Path: "internal/tui/theme.go", Lines: 50, LastTouched: time.Now()},
		{Path: "internal/tui/squarify.go", Lines: 215, LastTouched: time.Now()},
		{Path: "internal/tui/squarify_test.go", Lines: 60, LastTouched: time.Now()},
		{Path: "internal/git/git.go", Lines: 200, LastTouched: time.Now()},
		{Path: "internal/secrets/secrets.go", Lines: 150, LastTouched: time.Now()},
		{Path: "internal/analyzer/analyzer.go", Lines: 180, LastTouched: time.Now()},
		{Path: "internal/watcher/watcher.go", Lines: 90, LastTouched: time.Now()},
		{Path: "README.md", Lines: 45, LastTouched: time.Now()},
		{Path: "go.mod", Lines: 15, LastTouched: time.Now()},
	}

	m.Treemap = cells
	m.TotalLines = 0
	for _, c := range cells {
		m.TotalLines += c.Lines
	}

	viewStr := m.View()
	fmt.Println("--- VIEW START ---")
	fmt.Println(viewStr)
	fmt.Println("--- VIEW END ---")

	w, h := m.TreemapDims()
	_, boxes := m.layoutTreemap(w, h)
	fmt.Println("--- BOX DETAILS ---")
	for _, b := range boxes {
		fmt.Printf("Box %s: X=%d, Y=%d, W=%d, H=%d\n", b.Name, b.X, b.Y, b.W, b.H)
	}
}
