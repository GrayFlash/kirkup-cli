package classifier

import (
	"testing"

	"github.com/GrayFlash/kirkup-cli/config"
	"github.com/GrayFlash/kirkup-cli/models"
)

func TestLLMClassifier_ParseResponse(t *testing.T) {
	c := NewLLMClassifier(config.LLMConfig{Provider: "ollama", Model: "test"})
	
	batch := []models.PromptEvent{
		{ID: "id1", Prompt: "p1"},
		{ID: "id2", Prompt: "p2"},
	}
	
	// Valid JSON response
	resp := `[{"category": "coding", "confidence": 0.9}, {"category": "testing", "confidence": 0.8}]`
	res, err := c.parseResponse(batch, resp)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 results, got %d", len(res))
	}
	if res[0].Category != "coding" || res[0].PromptEventID != "id1" {
		t.Errorf("unexpected result 0: %+v", res[0])
	}

	// Response with embedded JSON
	resp2 := "Here is the JSON: ```json\n" + resp + "\n```"
	res2, err := c.parseResponse(batch, resp2)
	if err != nil {
		t.Fatalf("parseResponse error (embedded): %v", err)
	}
	if len(res2) != 2 {
		t.Errorf("expected 2 results, got %d", len(res2))
	}
	
	// Mismatched length
	resp3 := `[{"category": "coding", "confidence": 0.9}]`
	_, err = c.parseResponse(batch, resp3)
	if err == nil {
		t.Fatal("expected error for mismatched length, got nil")
	}
}
