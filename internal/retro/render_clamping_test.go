package retro

import (
	"testing"
)

func TestBar_Clamping(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
	}{
		{"negative percent", -10.0},
		{"zero percent", 0.0},
		{"over 100 percent", 150.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will panic if strings.Repeat receives a negative number
			result := bar(tt.percent)
			
			// We just want to ensure it doesn't panic and returns a string of the correct total length
			// barWidth is hardcoded to 20 in render.go
			if len([]rune(result)) != 20 {
				t.Errorf("expected bar length of 20, got %d", len([]rune(result)))
			}
		})
	}
}
