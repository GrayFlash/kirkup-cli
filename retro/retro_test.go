package retro

import (
	"testing"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
)

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// -- inferSessions --

func TestInferSessions_SingleSession(t *testing.T) {
	events := []models.PromptEvent{
		{Timestamp: mustTime("2026-03-10T09:00:00Z"), Project: "alpha", Agent: "gemini-cli"},
		{Timestamp: mustTime("2026-03-10T09:10:00Z"), Project: "alpha", Agent: "gemini-cli"},
		{Timestamp: mustTime("2026-03-10T09:20:00Z"), Project: "alpha", Agent: "gemini-cli"},
	}
	sessions := inferSessions(events, 30)
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(sessions))
	}
	if sessions[0].PromptCount != 3 {
		t.Errorf("want 3 prompts, got %d", sessions[0].PromptCount)
	}
}

func TestInferSessions_SplitByGap(t *testing.T) {
	events := []models.PromptEvent{
		{Timestamp: mustTime("2026-03-10T09:00:00Z"), Project: "alpha", Agent: "gemini-cli"},
		{Timestamp: mustTime("2026-03-10T09:10:00Z"), Project: "alpha", Agent: "gemini-cli"},
		// 45 minute gap — exceeds 30 minute threshold
		{Timestamp: mustTime("2026-03-10T09:55:00Z"), Project: "alpha", Agent: "gemini-cli"},
		{Timestamp: mustTime("2026-03-10T10:05:00Z"), Project: "alpha", Agent: "gemini-cli"},
	}
	sessions := inferSessions(events, 30)
	if len(sessions) != 2 {
		t.Fatalf("want 2 sessions, got %d", len(sessions))
	}
	if sessions[0].PromptCount != 2 {
		t.Errorf("session 0: want 2 prompts, got %d", sessions[0].PromptCount)
	}
	if sessions[1].PromptCount != 2 {
		t.Errorf("session 1: want 2 prompts, got %d", sessions[1].PromptCount)
	}
}

func TestInferSessions_MultipleAgents(t *testing.T) {
	events := []models.PromptEvent{
		{Timestamp: mustTime("2026-03-10T09:00:00Z"), Project: "alpha", Agent: "gemini-cli"},
		{Timestamp: mustTime("2026-03-10T09:05:00Z"), Project: "alpha", Agent: "cursor"},
		{Timestamp: mustTime("2026-03-10T09:10:00Z"), Project: "alpha", Agent: "gemini-cli"},
	}
	// gemini-cli and cursor are different groups → 2 sessions
	sessions := inferSessions(events, 30)
	if len(sessions) != 2 {
		t.Fatalf("want 2 sessions (one per agent), got %d", len(sessions))
	}
}

// -- countContextSwitches --

func TestCountContextSwitches_NoSwitches(t *testing.T) {
	events := []models.PromptEvent{
		{Project: "alpha"},
		{Project: "alpha"},
		{Project: "alpha"},
	}
	if got := countContextSwitches(events); got != 0 {
		t.Errorf("want 0 switches, got %d", got)
	}
}

func TestCountContextSwitches_MultipleSwitches(t *testing.T) {
	events := []models.PromptEvent{
		{Project: "alpha"},
		{Project: "alpha"},
		{Project: "beta"},
		{Project: "alpha"},
		{Project: "beta"},
	}
	if got := countContextSwitches(events); got != 3 {
		t.Errorf("want 3 switches, got %d", got)
	}
}

// -- bar rendering --

func TestBar_ZeroPercent(t *testing.T) {
	b := bar(0)
	if len([]rune(b)) != barWidth {
		t.Errorf("bar width: want %d, got %d", barWidth, len([]rune(b)))
	}
	if b != "░░░░░░░░░░░░░░░░░░░░" {
		t.Errorf("0%% bar should be all empty blocks, got %q", b)
	}
}

func TestBar_FullPercent(t *testing.T) {
	b := bar(100)
	if b != "████████████████████" {
		t.Errorf("100%% bar should be all full blocks, got %q", b)
	}
}

func TestBar_HalfPercent(t *testing.T) {
	b := bar(50)
	filled := 0
	for _, r := range b {
		if r == '█' {
			filled++
		}
	}
	if filled != 10 {
		t.Errorf("50%% bar: want 10 filled blocks, got %d", filled)
	}
}

// -- fmtDuration --

func TestFmtDuration_Hours(t *testing.T) {
	got := fmtDuration(90 * time.Minute)
	if got != "~1.5h" {
		t.Errorf("want ~1.5h, got %q", got)
	}
}

func TestFmtDuration_Minutes(t *testing.T) {
	got := fmtDuration(25 * time.Minute)
	if got != "~25m" {
		t.Errorf("want ~25m, got %q", got)
	}
}
