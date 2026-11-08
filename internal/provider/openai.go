package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// OpenAI talks to any OpenAI-compatible chat completions endpoint:
// Groq, OpenRouter, Ollama, LM Studio... selected by BaseURL.
type OpenAI struct {
	BaseURL string // e.g. https://api.groq.com/openai/v1
	APIKey  string
	Model   string // e.g. google/gemma-4-31b-it:free
}

// Request/response shapes: only the fields mull uses.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o OpenAI) Complete(ctx context.Context, instruction, input string) (Reply, error) {
	body, err := json.Marshal(chatRequest{
		Model: o.Model,
		Messages: []chatMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: input},
		},
	})
	if err != nil {
		return Reply{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Reply{}, err
	}
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Reply{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Reply{}, err
	}

	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Reply{}, fmt.Errorf("%s: HTTP %d: %s", o.host(), resp.StatusCode, raw)
	}
	if out.Error != nil {
		return Reply{}, fmt.Errorf("%s: HTTP %d: %s", o.host(), resp.StatusCode, out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK || len(out.Choices) == 0 {
		return Reply{}, fmt.Errorf("%s: HTTP %d: no reply: %s", o.host(), resp.StatusCode, raw)
	}
	return Reply{Text: out.Choices[0].Message.Content, Model: out.Model}, nil
}

// host names the backend in errors, e.g. "api.groq.com".
func (o OpenAI) host() string {
	if u, err := url.Parse(o.BaseURL); err == nil && u.Host != "" {
		return u.Host
	}
	return o.BaseURL
}
