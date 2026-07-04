package tui

import (
	"testing"
)

func TestLabelTiers(t *testing.T) {
	// Task 2: Verify that equal-size rectangles land in the same tier
	// strictly based on their physical dimensions, not their string length.
	
	testCases := []struct {
		name string
		w, h, lines int
		expectedTier int
	}{
		{
			name: ".env", // Short name
			w: 12, h: 2, lines: 10,
			expectedTier: 2, // intended=2, Area=24, W>=3, H>=2 -> 2
		},
		{
			name: "file_1.go", // Medium name
			w: 12, h: 2, lines: 10,
			expectedTier: 2,
		},
		{
			name: "extreme_tall",
			w: 3, h: 50, lines: 10,
			expectedTier: 2, // intended=2, Area=150, W=3, H=50 -> meets Area >= 24, W>=3, H>=2 -> 2
		},
		{
			name: "extreme_wide",
			w: 50, h: 3, lines: 10,
			expectedTier: 2, // intended=2, Area=150, W=50, H=3 -> meets Area >= 24, W>=3, H>=2 -> 2
		},
		{
			name: "short",
			w: 11, h: 2, lines: 10,
			expectedTier: 1, // intended=2, Area=22. Clamps to Tier 1.
		},
		{
			name: "very_long_name_that_gets_truncated",
			w: 11, h: 2, lines: 10,
			expectedTier: 1, 
		},
		{
			name: "x",
			w: 6, h: 1, lines: 10,
			expectedTier: 0, // intended=2, Area=6. Clamps to Tier 0.
		},
		{
			name: "long_name_small_box",
			w: 6, h: 1, lines: 10,
			expectedTier: 0, 
		},
		{
			name: "too_narrow",
			w: 2, h: 50, lines: 10,
			expectedTier: 0, // intended=2, Area=100, but W < 3 so clamps to Tier 0
		},
		{
			name: "intended_tier_1_clamp",
			w: 12, h: 2, lines: 2,
			expectedTier: 1, // intended=1, Area=24. Passes Tier 1 physical limits -> 1.
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tier := getLabelTier(tc.w, tc.h, tc.lines)
			if tier != tc.expectedTier {
				t.Errorf("expected tier %d for %dx%d box, got %d", tc.expectedTier, tc.w, tc.h, tier)
			}
		})
	}
}
