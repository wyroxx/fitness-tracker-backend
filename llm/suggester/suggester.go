package suggester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxSuggestionChars = 250

// ChatRequest is the payload sent to an OpenAI-compatible chat completions API.
type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
	Stream    bool          `json:"stream"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse captures an OpenAI-compatible chat completions response.
type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// Suggester wraps parameters for talking to an OpenAI-compatible LLM API.
type Suggester struct {
	baseURL   string
	modelName string
	apiKey    string
	httpCli   *http.Client
}

// New creates a new Suggester.
func New(baseURL, model, apiKey string) *Suggester {
	return &Suggester{
		baseURL:   strings.TrimRight(baseURL, "/"),
		modelName: model,
		apiKey:    apiKey,
		httpCli:   &http.Client{},
	}
}

// Suggest takes the user workout history description and returns a suggestion text.
func (s *Suggester) Suggest(history string) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("LLM API key is not configured")
	}

	reqBody := ChatRequest{
		Model: s.modelName,
		Messages: []ChatMessage{
			{Role: "user", Content: fmt.Sprintf("You are a fitness coach in a mobile app. Based on this workout history, suggest the next workout: %s."+
				"\nReturn one plain-text sentence in English, max 250 characters. Include the workout focus and 2-4 concrete exercises or set targets. No markdown, bullets, greetings, or explanations.", history)},
		},
		MaxTokens: 100,
		Stream:    false,
	}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", s.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpCli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("LLM API request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM API response did not include choices")
	}
	return compactSuggestion(chatResp.Choices[0].Message.Content), nil
}

func compactSuggestion(suggestion string) string {
	suggestion = strings.Join(strings.Fields(suggestion), " ")
	runes := []rune(suggestion)
	if len(runes) <= maxSuggestionChars {
		return suggestion
	}

	const suffix = "..."
	limit := maxSuggestionChars - len([]rune(suffix))
	if limit <= 0 {
		return string(runes[:maxSuggestionChars])
	}
	trimmed := strings.TrimRight(strings.TrimSpace(string(runes[:limit])), " .,;:-")
	if trimmed == "" {
		return string(runes[:maxSuggestionChars])
	}
	return trimmed + suffix
}
