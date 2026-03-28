package classifier

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
)

// Rule defines a single classification rule. The first rule (by Priority desc)
// whose keyword or pattern matches the prompt text wins.
type Rule struct {
	Category string
	Keywords []string
	Patterns []*regexp.Regexp
	Priority int
}

// RuleClassifier classifies prompt events using keyword and regex rules.
type RuleClassifier struct {
	rules []Rule
}

// NewRuleClassifier returns a RuleClassifier loaded with the default taxonomy.
// Additional rules can be appended via AddRule.
func NewRuleClassifier() *RuleClassifier {
	rc := &RuleClassifier{}
	for _, r := range defaultRules {
		rc.rules = append(rc.rules, compileRule(r))
	}
	return rc
}

func (rc *RuleClassifier) Name() string { return "rules-v1" }

// AddRule appends a custom rule. Higher priority rules are checked first.
func (rc *RuleClassifier) AddRule(category string, keywords []string, patterns []string, priority int) {
	r := rawRule{category, keywords, patterns, priority}
	rc.rules = append(rc.rules, compileRule(r))
	// Keep rules sorted by priority descending.
	sortRules(rc.rules)
}

// Classify assigns a category to each event. Events with no matching rule are
// omitted from the result.
func (rc *RuleClassifier) Classify(_ context.Context, events []models.PromptEvent) ([]models.Classification, error) {
	now := time.Now().UTC()
	var out []models.Classification
	for _, e := range events {
		category, ok := rc.classify(e.Prompt)
		if !ok {
			continue
		}
		out = append(out, models.Classification{
			PromptEventID: e.ID,
			Category:      category,
			Confidence:    1.0,
			Classifier:    rc.Name(),
			CreatedAt:     now,
		})
	}
	return out, nil
}

// classify returns the category for a single prompt text.
func (rc *RuleClassifier) classify(prompt string) (string, bool) {
	lower := strings.ToLower(prompt)
	for _, r := range rc.rules {
		if matchesRule(r, lower) {
			return r.Category, true
		}
	}
	return "", false
}

func matchesRule(r Rule, lowerPrompt string) bool {
	for _, kw := range r.Keywords {
		if strings.Contains(lowerPrompt, kw) {
			return true
		}
	}
	for _, pat := range r.Patterns {
		if pat.MatchString(lowerPrompt) {
			return true
		}
	}
	return false
}

// -- default taxonomy --

type rawRule struct {
	Category string
	Keywords []string
	Patterns []string
	Priority int
}

var defaultRules = []rawRule{
	{
		Category: "debugging",
		Keywords: []string{"debug", "fix bug", "why is", "why does", "not working", "broken", "failing", "error", "exception", "crash", "panic", "stacktrace", "traceback"},
		Priority: 10,
	},
	{
		Category: "testing",
		Keywords: []string{"test", "spec", "assert", "coverage", "mock", "stub", "fixture", "benchmark"},
		Priority: 9,
	},
	{
		Category: "refactoring",
		Keywords: []string{"refactor", "extract", "rename", "restructure", "clean up", "cleanup", "reorganize", "simplify", "decouple", "move"},
		Priority: 8,
	},
	{
		Category: "review",
		Keywords: []string{"review", "code review", "pull request", "diff", "lgtm", "approve"},
		Priority: 7,
	},
	{
		Category: "infra",
		Keywords: []string{"docker", "dockerfile", "ci", "cd", "pipeline", "deploy", "kubernetes", "k8s", "nginx", "terraform", "ansible", "github action", "workflow"},
		Priority: 6,
	},
	{
		Category: "spec-reading",
		Keywords: []string{"explain", "what is", "what does", "how does", "what are", "describe", "understand", "clarify", "definition of"},
		Priority: 5,
	},
	{
		Category: "documentation",
		Keywords: []string{"readme", "godoc", "jsdoc", "docstring", "write doc", "document", "add comment", "add comments"},
		Priority: 4,
	},
	{
		Category: "exploration",
		Keywords: []string{"spike", "prototype", "explore", "experiment", "try out", "research", "investigate", "how to"},
		Priority: 3,
	},
	{
		Category: "coding",
		Keywords: []string{"implement", "create", "build", "add feature", "write", "add", "generate", "make"},
		Priority: 1,
	},
}

func compileRule(r rawRule) Rule {
	compiled := Rule{
		Category: r.Category,
		Keywords: r.Keywords,
		Priority: r.Priority,
	}
	for _, p := range r.Patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled.Patterns = append(compiled.Patterns, re)
		} else {
			log.Printf("warning: invalid regex pattern %q in category %q: %v\n", p, r.Category, err)
		}
	}
	return compiled
}

func sortRules(rules []Rule) {
	// insertion sort — rule lists are small
	for i := 1; i < len(rules); i++ {
		for j := i; j > 0 && rules[j].Priority > rules[j-1].Priority; j-- {
			rules[j], rules[j-1] = rules[j-1], rules[j]
		}
	}
}
