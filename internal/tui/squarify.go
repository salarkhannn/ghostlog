package tui

import (
	"math"
	"time"
)

type TreemapItemWeight struct {
	Path        string
	Name        string
	IsDir       bool
	Lines       int
	LastTouched time.Time
	ColorIndex  int
	Weight      float64
}

type Rect struct {
	X, Y, W, H int
}

type PositionedBox struct {
	Path        string
	Name        string
	IsDir       bool
	Lines       int
	LastTouched time.Time
	ColorIndex  int
	Weight      float64
	X, Y, W, H  int
}

type floatRect struct {
	x, y, w, h float64
}

func Squarify(items []TreemapItemWeight, bounds Rect) []PositionedBox {
	if len(items) == 0 || bounds.W <= 0 || bounds.H <= 0 {
		return nil
	}

	var totalWeight float64
	for _, it := range items {
		totalWeight += it.Weight
	}
	if totalWeight <= 0 {
		totalWeight = float64(len(items))
		for i := range items {
			items[i].Weight = 1.0
		}
	}

	totalArea := float64(bounds.W * bounds.H)
	areas := make([]float64, len(items))
	for i, it := range items {
		areas[i] = it.Weight * totalArea / totalWeight
	}

	container := floatRect{
		x: float64(bounds.X),
		y: float64(bounds.Y),
		w: float64(bounds.W),
		h: float64(bounds.H),
	}

	type placement struct {
		item  TreemapItemWeight
		bound floatRect
	}
	var placements []placement

	worstRatio := func(row []float64, s float64) float64 {
		if len(row) == 0 {
			return math.MaxFloat64
		}
		var sum float64
		minA := row[0]
		maxA := row[0]
		for _, a := range row {
			sum += a
			if a < minA {
				minA = a
			}
			if a > maxA {
				maxA = a
			}
		}
		s2 := s * s
		sum2 := sum * sum
		r1 := sum2 / (s2 * minA)
		r2 := (s2 * maxA) / sum2
		if r1 > r2 {
			return r1
		}
		return r2
	}

	layoutRow := func(rowItems []TreemapItemWeight, rowAreas []float64, s float64, rect *floatRect) {
		var sumArea float64
		for _, a := range rowAreas {
			sumArea += a
		}
		if sumArea == 0 {
			return
		}

		isHorizontal := rect.w >= rect.h
		thickness := sumArea / s

		currOffset := 0.0
		if isHorizontal {
			for i, it := range rowItems {
				itemH := s * rowAreas[i] / sumArea
				placements = append(placements, placement{
					item: it,
					bound: floatRect{
						x: rect.x,
						y: rect.y + currOffset,
						w: thickness,
						h: itemH,
					},
				})
				currOffset += itemH
			}
			rect.x += thickness
			rect.w -= thickness
		} else {
			for i, it := range rowItems {
				itemW := s * rowAreas[i] / sumArea
				placements = append(placements, placement{
					item: it,
					bound: floatRect{
						x: rect.x + currOffset,
						y: rect.y,
						w: itemW,
						h: thickness,
					},
				})
				currOffset += itemW
			}
			rect.y += thickness
			rect.h -= thickness
		}
	}

	var rowItems []TreemapItemWeight
	var rowAreas []float64

	for i := 0; i < len(items); i++ {
		it := items[i]
		a := areas[i]

		s := container.w
		if container.h < container.w {
			s = container.h
		}

		if worstRatio(append(rowAreas, a), s) <= worstRatio(rowAreas, s) {
			rowItems = append(rowItems, it)
			rowAreas = append(rowAreas, a)
		} else {
			layoutRow(rowItems, rowAreas, s, &container)
			rowItems = []TreemapItemWeight{it}
			rowAreas = []float64{a}
		}
	}
	if len(rowItems) > 0 {
		s := container.w
		if container.h < container.w {
			s = container.h
		}
		layoutRow(rowItems, rowAreas, s, &container)
	}

	res := make([]PositionedBox, len(placements))
	for i, p := range placements {
		x0 := math.Round(p.bound.x)
		y0 := math.Round(p.bound.y)
		x1 := math.Round(p.bound.x + p.bound.w)
		y1 := math.Round(p.bound.y + p.bound.h)

		if x1 > float64(bounds.X+bounds.W) {
			x1 = float64(bounds.X + bounds.W)
		}
		if y1 > float64(bounds.Y+bounds.H) {
			y1 = float64(bounds.Y + bounds.H)
		}

		w := int(x1 - x0)
		h := int(y1 - y0)
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}

		res[i] = PositionedBox{
			Path:        p.item.Path,
			Name:        p.item.Name,
			IsDir:       p.item.IsDir,
			Lines:       p.item.Lines,
			LastTouched: p.item.LastTouched,
			ColorIndex:  p.item.ColorIndex,
			Weight:      p.item.Weight,
			X:           int(x0),
			Y:           int(y0),
			W:           w,
			H:           h,
		}
	}

	return res
}
