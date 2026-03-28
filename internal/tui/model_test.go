package tui

import (
	"testing"

	"github.com/GrayFlash/kirkup-cli/internal/retro"
)

func TestModel_ViewNarrow(t *testing.T) {
	// A terminal width of 10 should not cause a panic
	m := Model{
		width: 10,
		height: 20,
		projects: []projectEntry{{name: "test", prompts: 5}},
		summary: &retro.Summary{
			TotalPrompts: 10,
			Categories: []retro.CategoryStat{{Category: "test", Percent: 50.0}},
			Daily: []retro.DayStat{{Prompts: 5}},
		},
	}
	
	// Just make sure it doesn't panic!
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("View() panicked on narrow width: %v", r)
		}
	}()
	_ = m.View()
}
