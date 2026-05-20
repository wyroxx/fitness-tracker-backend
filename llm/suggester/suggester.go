package suggester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// OllamaChatRequest is the payload sent to /api/chat endpoint of Ollama.
type OllamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaChatResponse captures the stream response; we care about the final content field.
type OllamaChatResponse struct {
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

// Suggester wraps parameters for talking to the Ollama server.
type Suggester struct {
	baseURL   string
	modelName string
	httpCli   *http.Client
}

// EnsureModel pulls the specified model from the Ollama server if it is not already present.
// It blocks until the pull completes (or returns an error).
func EnsureModel(baseURL, model string) error {
	reqBody := map[string]string{"name": model}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", baseURL+"/api/pull", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	for {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			break // stream finished
		}
		// when status == "success" or "exists" or "complete" we can finish
		if status, ok := m["status"].(string); ok {
			switch status {
			case "success", "exists", "complete", "already exists":
				return nil
			}
		}
	}
	return nil // default: assume success
}

// New creates a new Suggester.
func New(baseURL, model string) *Suggester {
	return &Suggester{baseURL: baseURL, modelName: model, httpCli: &http.Client{}}
}

// Suggest takes the user workout history description and returns a suggestion text.
func (s *Suggester) Suggest(history string) (string, error) {
	reqBody := OllamaChatRequest{
		Model: s.modelName,
		Messages: []ChatMessage{
			{Role: "user", Content: fmt.Sprintf("Based on this workout history, suggest the next workout: %s."+
				"\nYour response should be concise and include 3-5 sentences.", history)},
		},
	}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", s.baseURL+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpCli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var suggestion string
	for {
		var chunk OllamaChatResponse
		if err := dec.Decode(&chunk); err != nil {
			break
		}
		suggestion += chunk.Message.Content
		if chunk.Done {
			break
		}
	}
	return suggestion, nil
}
