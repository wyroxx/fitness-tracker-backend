package suggester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxSuggestionChars = 250

	suggestionPromptTemplate = `You are a fitness coach in a mobile workout app.

Based on the user's recent workout history, suggest the next workout in a practical coach-like style.

Decision rules:
- Choose a focus that fits recovery, balance, and recent training gaps.
- Do not repeat the exact same focus as the latest workout unless it makes sense as a light session.
- Prioritize undertrained muscles or useful balance work.
- Prefer exercises the user already performs, but add one helpful accessory if needed.
- Do not give exact weights, reps, or set numbers.
- Use qualitative guidance like "keep it controlled", "push slightly harder", "make it lighter", "focus on form", or "add rear delt work".
- Mention 2-4 exercises.
- Keep it realistic for one workout.

Workout history:
%s

Return exactly one plain-text sentence in English, max 250 characters.
No markdown, bullets, greetings, explanations, emojis, or disclaimers.
Use only these exercises:
Chest: Bench Press, Incline Bench Press, Dumbbell Bench Press, Incline Dumbbell Press, Hammer Chest Press, Push Up, Dips, Pec Deck Fly, Cable Chest Fly.
Back: Pull Up, Lat Pulldown, Seated Cable Row, Barbell Row, One Arm Dumbbell Row, Chest Supported Row, Back Extension.
Legs: Squat, Leg Press, Romanian Deadlift, Deadlift, Bulgarian Squat, Leg Extension, Leg Curl, Standing Calf Raise, Seated Calf Raise, Hip Abduction, Hip Thrust.
Abs: Crunch, Leg Raise, Hanging Leg Raise, Cable Crunch, Ab Machine Crunch, Russian Twist, Plank.
Shoulders: Dumbbell Shoulder Press, Machine Shoulder Press, Lateral Raise, Reverse Pec Deck, Face Pull, Front Raise.
Arms: Barbell Curl, Dumbbell Curl, Hammer Curl, Incline Dumbbell Curl, Preacher Curl, Overhead Triceps Extension, Skull Crusher, Triceps Pushdown, Rope Triceps Pushdown, Diamond Push Ups.`
)

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
		httpCli:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Suggest takes the user workout history description and returns a suggestion text.
func (s *Suggester) Suggest(ctx context.Context, history string) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("LLM API key is not configured")
	}

	reqBody := ChatRequest{
		Model: s.modelName,
		Messages: []ChatMessage{
			{Role: "user", Content: fmt.Sprintf(suggestionPromptTemplate, history)},
		},
		MaxTokens: 100,
		Stream:    false,
	}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewReader(b))
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
