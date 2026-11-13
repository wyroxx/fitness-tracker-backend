package suggester

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSuggestCallsOpenAICompatibleChatCompletions(t *testing.T) {
	var gotPath, gotAuth, gotModel string
	sg := New("https://api.groq.com/openai/v1/", "llama-3.1-8b-instant", "test-key")
	sg.httpCli = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		gotModel = req.Model
		if req.MaxTokens < 2048 {
			t.Fatal("reasoning models need room to produce the final answer")
		}

		body := `{"choices":[{"message":{"role":"assistant","content":"Push focus: bench press 3x8, incline dumbbell press 3x10, triceps dips 3x12."}}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}

	got, err := sg.Suggest(context.Background(), "Training 1: Chest day - bench press")
	if err != nil {
		t.Fatalf("Suggest() error = %v", err)
	}

	if gotPath != "/openai/v1/chat/completions" {
		t.Fatalf("request path = %q, want /openai/v1/chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want Bearer token", gotAuth)
	}
	if gotModel != "llama-3.1-8b-instant" {
		t.Fatalf("model = %q, want llama-3.1-8b-instant", gotModel)
	}
	if !strings.Contains(got, "Push focus") {
		t.Fatalf("Suggest() = %q, want parsed assistant content", got)
	}
}

func TestSuggestRequiresAPIKey(t *testing.T) {
	sg := New("https://api.groq.com/openai/v1", "llama-3.1-8b-instant", "")

	if _, err := sg.Suggest(context.Background(), "history"); err == nil {
		t.Fatal("Suggest() error = nil, want API key error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestCompactSuggestionCollapsesWhitespace(t *testing.T) {
	got := compactSuggestion(" Push day:\n\nbench press   3x8,\trows 3x10. ")
	want := "Push day: bench press 3x8, rows 3x10."

	if got != want {
		t.Fatalf("compactSuggestion() = %q, want %q", got, want)
	}
}

func TestCompactSuggestionLimitsCharacters(t *testing.T) {
	got := compactSuggestion(strings.Repeat("a", maxSuggestionChars+20))

	if len([]rune(got)) > maxSuggestionChars {
		t.Fatalf("compactSuggestion() length = %d, want <= %d", len([]rune(got)), maxSuggestionChars)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("compactSuggestion() = %q, want truncation suffix", got)
	}
}

func TestCompactSuggestionCountsRunes(t *testing.T) {
	got := compactSuggestion(strings.Repeat("\u0436", maxSuggestionChars+20))

	if len([]rune(got)) > maxSuggestionChars {
		t.Fatalf("compactSuggestion() length = %d, want <= %d", len([]rune(got)), maxSuggestionChars)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("compactSuggestion() = %q, want truncation suffix", got)
	}
}

func TestCoachIndependentRequest(t *testing.T) {
	sg := New("https://example.invalid", "test", "key")
	sg.httpCli = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var request ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Messages) != 3 || request.Messages[0].Role != "system" || request.MaxTokens != 500 {
			t.Errorf("unexpected coach request: %+v", request)
		}
		if request.Messages[2].Content == "" {
			t.Error("missing default question")
		}
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": strings.Repeat("Progress. ", 40)}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}
	result, err := sg.Coach(context.Background(), `{"workouts":2}`, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) <= 250 {
		t.Fatal("coach answer was truncated to legacy suggestion limit")
	}
}
