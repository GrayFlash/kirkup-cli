package classifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/GrayFlash/kirkup-cli/config"
	"github.com/GrayFlash/kirkup-cli/models"
)

type LLMClassifier struct {
	cfg config.LLMConfig
}

func NewLLMClassifier(cfg config.LLMConfig) *LLMClassifier {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.Endpoint == "" && cfg.Provider == "ollama" {
		cfg.Endpoint = "http://localhost:11434"
	}
	return &LLMClassifier{cfg: cfg}
}

func (c *LLMClassifier) Name() string {
	return fmt.Sprintf("llm-%s-%s", c.cfg.Provider, c.cfg.Model)
}

func (c *LLMClassifier) Classify(ctx context.Context, events []models.PromptEvent) ([]models.Classification, error) {
	fmt.Println("warning: LLM classification sends full prompt text to the configured provider.")
	
	var results []models.Classification

	// Process in batches
	for i := 0; i < len(events); i += c.cfg.BatchSize {
		end := i + c.cfg.BatchSize
		if end > len(events) {
			end = len(events)
		}
		batch := events[i:end]

		classifications, err := c.classifyBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		results = append(results, classifications...)
	}

	return results, nil
}

func (c *LLMClassifier) classifyBatch(ctx context.Context, batch []models.PromptEvent) ([]models.Classification, error) {
	prompt := c.buildPrompt(batch)

	var response string
	var err error

	switch c.cfg.Provider {
	case "ollama":
		response, err = c.callOllama(ctx, prompt)
	case "openai":
		response, err = c.callOpenAI(ctx, prompt)
	case "anthropic":
		response, err = c.callAnthropic(ctx, prompt)
	case "vertex":
		response, err = c.callVertexAI(ctx, prompt)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %q", c.cfg.Provider)
	}

	if err != nil {
		return nil, err
	}

	return c.parseResponse(batch, response)
}

func (c *LLMClassifier) callOpenAI(ctx context.Context, prompt string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	if c.cfg.Endpoint != "" {
		if strings.HasPrefix(c.cfg.Endpoint, "http://") && !strings.Contains(c.cfg.Endpoint, "localhost") && !strings.Contains(c.cfg.Endpoint, "127.0.0.1") {
			fmt.Printf("warning: using cleartext HTTP for OpenAI endpoint %q may leak your API key\n", c.cfg.Endpoint)
		}
		url = c.cfg.Endpoint
	}

	body := map[string]any{
		"model": c.cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(data))
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return res.Choices[0].Message.Content, nil
}

func (c *LLMClassifier) callAnthropic(ctx context.Context, prompt string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"
	if c.cfg.Endpoint != "" {
		if strings.HasPrefix(c.cfg.Endpoint, "http://") && !strings.Contains(c.cfg.Endpoint, "localhost") && !strings.Contains(c.cfg.Endpoint, "127.0.0.1") {
			fmt.Printf("warning: using cleartext HTTP for Anthropic endpoint %q may leak your API key\n", c.cfg.Endpoint)
		}
		url = c.cfg.Endpoint
	}

	body := map[string]any{
		"model":      c.cfg.Model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic error (status %d): %s", resp.StatusCode, string(data))
	}

	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Content) == 0 {
		return "", fmt.Errorf("anthropic returned no content")
	}

	return res.Content[0].Text, nil
}

func (c *LLMClassifier) buildPrompt(batch []models.PromptEvent) string {
	var sb bytes.Buffer
	sb.WriteString("You are an assistant that categorizes AI coding prompts into one of the following categories:\n")
	sb.WriteString("- coding: implementing features, writing logic\n")
	sb.WriteString("- debugging: fixing bugs, why is this broken, stacktraces\n")
	sb.WriteString("- testing: writing tests, mocking, benchmarking\n")
	sb.WriteString("- refactoring: renaming, restructuring, cleaning up code\n")
	sb.WriteString("- review: reviewing PRs, explaining diffs\n")
	sb.WriteString("- infra: docker, CI/CD, deployment, kubernetes\n")
	sb.WriteString("- spec-reading: understanding requirements, clarifying specs, explaining concepts\n")
	sb.WriteString("- documentation: writing readmes, docstrings, add comments\n")
	sb.WriteString("- exploration: spike, research, prototyping, trying out new libraries\n\n")
	sb.WriteString("For each prompt below, pick the best category and provide a confidence score (0.0 to 1.0).\n")
	sb.WriteString("Respond ONLY with a JSON array of objects, each having 'category' and 'confidence' fields.\n\n")

	for i, e := range batch {
		fmt.Fprintf(&sb, "Prompt %d: %s\n", i+1, e.Prompt)
	}

	sb.WriteString("\nOutput Format: [{\"category\": \"...\", \"confidence\": 0.9}, ...]")
	return sb.String()
}

func (c *LLMClassifier) callOllama(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", c.cfg.Endpoint)
	
	body := map[string]any{
		"model":  c.cfg.Model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	}
	
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(data))
	}
	
	var res struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	
	return res.Response, nil
}

func (c *LLMClassifier) parseResponse(batch []models.PromptEvent, response string) ([]models.Classification, error) {
	var items []struct {
		Category   string  `json:"category"`
		Confidence float64 `json:"confidence"`
	}
	
	if err := json.Unmarshal([]byte(response), &items); err != nil {
		// Try to find the JSON array if it's embedded in text
		start := strings.Index(response, "[")
		end := strings.LastIndex(response, "]")
		if start != -1 && end != -1 && end > start {
			if err := json.Unmarshal([]byte(response[start:end+1]), &items); err != nil {
				return nil, fmt.Errorf("failed to parse LLM response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse LLM response: %w", err)
		}
	}
	
	if len(items) != len(batch) {
		// If lengths don't match, we can't safely map them.
		// For now, return what we have or error.
		return nil, fmt.Errorf("LLM returned %d items, expected %d", len(items), len(batch))
	}
	
	var results []models.Classification
	for i, item := range items {
		results = append(results, models.Classification{
			PromptEventID: batch[i].ID,
			Category:      item.Category,
			Confidence:    item.Confidence,
			Classifier:    c.Name(),
			CreatedAt:     time.Now().UTC(),
		})
	}
	
	return results, nil
}

func (c *LLMClassifier) callVertexAI(ctx context.Context, prompt string) (string, error) {
	location := c.cfg.Location
	if location == "" {
		location = "us-central1"
	}
	project := c.cfg.ProjectID
	if project == "" {
		return "", fmt.Errorf("vertex ai requires project_id to be set in config")
	}
	model := c.cfg.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}

	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent", location, project, location, model)
	if c.cfg.Endpoint != "" {
		url = c.cfg.Endpoint
	}

	token := c.cfg.APIKey
	if token == "" {
		// Attempt to fetch ADC token via gcloud
		cmd := exec.CommandContext(ctx, "gcloud", "auth", "print-access-token")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get gcloud token: %w. Provide api_key or install gcloud CLI", err)
		}
		token = strings.TrimSpace(string(out))
	}

	body := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vertex ai error (status %d): %s", resp.StatusCode, string(data))
	}

	var res struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("vertex ai returned no content")
	}

	return res.Candidates[0].Content.Parts[0].Text, nil
}

