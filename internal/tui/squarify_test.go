package tui

import (
	"testing"
)

func TestSquarify(t *testing.T) {
	items := []TreemapItemWeight{
		{Name: "6a", Weight: 6},
		{Name: "6b", Weight: 6},
		{Name: "4", Weight: 4},
		{Name: "3", Weight: 3},
		{Name: "2a", Weight: 2},
		{Name: "2b", Weight: 2},
		{Name: "1", Weight: 1},
	}
	bounds := Rect{X: 0, Y: 0, W: 6, H: 4}

	boxes := Squarify(items, bounds)
	if len(boxes) != len(items) {
		t.Fatalf("expected %d boxes, got %d", len(items), len(boxes))
	}

	expected := []struct {
		name string
		x, y, w, h int
	}{
		{"6a", 0, 0, 3, 2},
		{"6b", 0, 2, 3, 2},
		{"4", 3, 0, 2, 2},
		{"3", 5, 0, 1, 2},
		{"2a", 3, 2, 1, 2},
		{"2b", 4, 2, 1, 2},
		{"1", 5, 2, 1, 2},
	}

	for i, exp := range expected {
		b := boxes[i]
		if b.Name != exp.name {
			t.Errorf("box %d: expected name %q, got %q", i, exp.name, b.Name)
		}
		if b.X != exp.x || b.Y != exp.y || b.W != exp.w || b.H != exp.h {
			t.Errorf("box %s: expected bounds %+v, got {X:%d, Y:%d, W:%d, H:%d}", b.Name, exp, b.X, b.Y, b.W, b.H)
		}
	}
}

func TestSquarifyNonZeroDimensions(t *testing.T) {
	items := []TreemapItemWeight{
		{Name: "item1", Weight: 100},
		{Name: "item2", Weight: 50},
		{Name: "item3", Weight: 2},
		{Name: "item4", Weight: 1},
		{Name: "item5", Weight: 0.1},
	}
	bounds := Rect{X: 0, Y: 0, W: 5, H: 5}
	boxes := Squarify(items, bounds)

	for _, b := range boxes {
		if b.W <= 0 || b.H <= 0 {
			t.Errorf("box %s has zero or negative dimension: W=%d, H=%d", b.Name, b.W, b.H)
		}
	}
}
