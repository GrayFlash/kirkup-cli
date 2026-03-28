package classifier

import (
	"testing"
)

func TestCompileRule_InvalidRegex(t *testing.T) {
	// A mock raw rule with one valid and one invalid regex
	r := rawRule{
		Category: "test-cat",
		Patterns: []string{"^valid$", "[invalid"},
	}

	compiled := compileRule(r)
	
	// Should only compile the valid one and skip the invalid one without panicking
	if len(compiled.Patterns) != 1 {
		t.Fatalf("expected 1 valid pattern compiled, got %d", len(compiled.Patterns))
	}
}

func TestSortRules(t *testing.T) {
	rules := []Rule{
		{Category: "low", Priority: 1},
		{Category: "high", Priority: 10},
		{Category: "mid", Priority: 5},
	}
	sortRules(rules)
	
	if rules[0].Priority != 10 {
		t.Errorf("expected highest priority first, got %d", rules[0].Priority)
	}
	if rules[2].Priority != 1 {
		t.Errorf("expected lowest priority last, got %d", rules[2].Priority)
	}
}
