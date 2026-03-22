package classifier_test

import (
	"context"
	"testing"

	"github.com/GrayFlash/kirkup-cli/classifier"
	"github.com/GrayFlash/kirkup-cli/models"
)

func classify(t *testing.T, prompt string) string {
	t.Helper()
	rc := classifier.NewRuleClassifier()
	results, err := rc.Classify(context.Background(), []models.PromptEvent{
		{ID: "test", Prompt: prompt},
	})
	if err != nil {
		t.Fatalf("classify error: %v", err)
	}
	if len(results) == 0 {
		return ""
	}
	return results[0].Category
}

func TestDefaultTaxonomy(t *testing.T) {
	cases := []struct {
		prompt string
		want   string
	}{
		// debugging
		{"why is the output value incorrect", "debugging"},
		{"fix bug in the parser", "debugging"},
		{"the server is crashing on startup", "debugging"},
		// testing
		{"add table-driven tests for the validator", "testing"},
		{"write benchmark for the parser", "testing"},
		// refactoring
		{"refactor the auth middleware", "refactoring"},
		{"extract the price parser into its own package", "refactoring"},
		// review
		{"review this diff for concurrency issues", "review"},
		{"can you do a code review of this function", "review"},
		// infra
		{"fix the Dockerfile build", "infra"},
		{"update the github action workflow", "infra"},
		// spec-reading
		{"explain what this API field means", "spec-reading"},
		{"what does this function return", "spec-reading"},
		// documentation
		{"write godoc for this module", "documentation"},
		{"add comments to the handler", "documentation"},
		// exploration
		{"how to handle timezone conversion in Go", "exploration"},
		{"spike: investigate using redis for caching", "exploration"},
		// coding
		{"implement the parser", "coding"},
		{"create a new endpoint for user login", "coding"},
	}

	for _, c := range cases {
		got := classify(t, c.prompt)
		if got != c.want {
			t.Errorf("prompt %q: want %q, got %q", c.prompt, c.want, got)
		}
	}
}

func TestUnmatchedPromptOmitted(t *testing.T) {
	rc := classifier.NewRuleClassifier()
	results, err := rc.Classify(context.Background(), []models.PromptEvent{
		{ID: "1", Prompt: "xyz123 gibberish zzz"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results for unmatched prompt, got %d", len(results))
	}
}

func TestHigherPriorityWins(t *testing.T) {
	// "fix bug" contains both debugging keywords and "fix" could match coding
	// debugging (priority 10) should win over coding (priority 1)
	got := classify(t, "fix bug in the settlement calculator")
	if got != "debugging" {
		t.Errorf("want debugging (higher priority), got %q", got)
	}
}

func TestClassifierName(t *testing.T) {
	rc := classifier.NewRuleClassifier()
	if rc.Name() != "rules-v1" {
		t.Errorf("want rules-v1, got %q", rc.Name())
	}
}

func TestClassificationFields(t *testing.T) {
	rc := classifier.NewRuleClassifier()
	results, err := rc.Classify(context.Background(), []models.PromptEvent{
		{ID: "evt-1", Prompt: "implement the parser"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	c := results[0]
	if c.PromptEventID != "evt-1" {
		t.Errorf("PromptEventID: want evt-1, got %q", c.PromptEventID)
	}
	if c.Confidence != 1.0 {
		t.Errorf("Confidence: want 1.0, got %f", c.Confidence)
	}
	if c.Classifier != "rules-v1" {
		t.Errorf("Classifier: want rules-v1, got %q", c.Classifier)
	}
}

func TestAddCustomRule(t *testing.T) {
	rc := classifier.NewRuleClassifier()
	rc.AddRule("rework", []string{"spec changed", "requirement changed"}, nil, 10)

	results, err := rc.Classify(context.Background(), []models.PromptEvent{
		{ID: "test", Prompt: "the spec changed, need to update the handler"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("want a classification, got none")
	}
	if results[0].Category != "rework" {
		t.Errorf("want rework (custom rule), got %q", results[0].Category)
	}
}

func TestMultipleEvents(t *testing.T) {
	rc := classifier.NewRuleClassifier()
	events := []models.PromptEvent{
		{ID: "1", Prompt: "implement the parser"},
		{ID: "2", Prompt: "xyz gibberish"},
		{ID: "3", Prompt: "fix bug in the handler"},
	}
	results, err := rc.Classify(context.Background(), events)
	if err != nil {
		t.Fatal(err)
	}
	// unmatched event should be omitted
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if results[0].PromptEventID != "1" {
		t.Errorf("first result should be event 1, got %q", results[0].PromptEventID)
	}
	if results[1].PromptEventID != "3" {
		t.Errorf("second result should be event 3, got %q", results[1].PromptEventID)
	}
}
