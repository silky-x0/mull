package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// OpenAI talks to any OpenAI-compatible API (chat completions, audio
// transcriptions): Groq, OpenRouter, Ollama, LM Studio... selected by BaseURL.
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

	var out chatResponse
	if err := o.post(ctx, "/chat/completions", "application/json", bytes.NewReader(body), &out); err != nil {
		return Reply{}, err
	}
	if len(out.Choices) == 0 {
		return Reply{}, fmt.Errorf("%s: no reply", o.host())
	}
	return Reply{Text: out.Choices[0].Message.Content, Model: out.Model}, nil
}

// Transcribe turns speech into text via /audio/transcriptions (Whisper).
// filename only tells the server the audio format, e.g. "speech.wav".
func (o OpenAI) Transcribe(ctx context.Context, filename string, audio io.Reader) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, audio); err != nil {
		return "", err
	}
	if err := w.WriteField("model", o.Model); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	var out struct {
		Text string `json:"text"`
	}
	if err := o.post(ctx, "/audio/transcriptions", w.FormDataContentType(), &body, &out); err != nil {
		return "", err
	}
	return out.Text, nil
}

// post sends body to BaseURL+path and decodes the JSON reply into out,
// turning HTTP and API errors into one readable error.
func (o OpenAI) post(ctx context.Context, path, contentType string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var apiErr struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &apiErr) == nil && apiErr.Error != nil {
		return fmt.Errorf("%s: HTTP %d: %s", o.host(), resp.StatusCode, apiErr.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d: %s", o.host(), resp.StatusCode, raw)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: unreadable reply: %s", o.host(), raw)
	}
	return nil
}

// host names the backend in errors, e.g. "api.groq.com".
func (o OpenAI) host() string {
	if u, err := url.Parse(o.BaseURL); err == nil && u.Host != "" {
		return u.Host
	}
	return o.BaseURL
}
