package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

const (
	maxBodyBytes  = 1 << 20
	maxErrorBytes = 4096
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
	Stream   bool          `json:"stream,omitempty"`
}

type streamChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta chatMessage `json:"delta"`
	} `json:"choices"`
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
		return Reply{}, fmt.Errorf("encoding chat request: %w", err)
	}

	var out chatResponse
	if err := o.post(ctx, "/chat/completions", "application/json", bytes.NewReader(body), &out); err != nil {
		return Reply{}, fmt.Errorf("chat completion: %w", err)
	}
	if len(out.Choices) == 0 {
		return Reply{}, fmt.Errorf("%s: no reply", o.host())
	}
	return Reply{Text: out.Choices[0].Message.Content, Model: out.Model}, nil
}

func (o OpenAI) Stream(ctx context.Context, instruction, input string, onDelta func(string)) (Reply, error) {
	body, err := json.Marshal(chatRequest{
		Model:  o.Model,
		Stream: true,
		Messages: []chatMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: input},
		},
	})
	if err != nil {
		return Reply{}, fmt.Errorf("encoding chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Reply{}, fmt.Errorf("creating request: %w", err)
	}
	o.setHeaders(req, "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Reply{}, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Reply{}, o.responseError(resp)
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 4096), maxBodyBytes)
	var text, model string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return Reply{}, fmt.Errorf("%s: unreadable stream chunk: %s", o.host(), clipError(data))
		}
		if chunk.Model != "" {
			model = chunk.Model
		}
		for _, c := range chunk.Choices {
			if c.Delta.Content != "" {
				text += c.Delta.Content
				onDelta(c.Delta.Content)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return Reply{}, fmt.Errorf("reading stream: %w", err)
	}
	if text == "" {
		return Reply{}, fmt.Errorf("%s: no reply", o.host())
	}
	return Reply{Text: text, Model: model}, nil
}

// Transcribe turns speech into text via /audio/transcriptions (Whisper).
// filename only tells the server the audio format, e.g. "speech.wav".
func (o OpenAI) Transcribe(ctx context.Context, filename string, audio io.Reader) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("creating upload form: %w", err)
	}
	if _, err := io.Copy(fw, audio); err != nil {
		return "", fmt.Errorf("reading audio: %w", err)
	}
	if err := w.WriteField("model", o.Model); err != nil {
		return "", fmt.Errorf("writing model field: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("closing upload form: %w", err)
	}

	var out struct {
		Text string `json:"text"`
	}
	if err := o.post(ctx, "/audio/transcriptions", w.FormDataContentType(), &body, &out); err != nil {
		return "", fmt.Errorf("transcription: %w", err)
	}
	return out.Text, nil
}

// post sends body to BaseURL+path and decodes the JSON reply into out,
// turning HTTP and API errors into one readable error.
func (o OpenAI) post(ctx context.Context, path, contentType string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	o.setHeaders(req, contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return o.responseError(resp)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	var apiErr struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &apiErr) == nil && apiErr.Error != nil {
		return fmt.Errorf("%s: HTTP %d: %s", o.host(), resp.StatusCode, clipError(apiErr.Error.Message))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: unreadable reply: %s", o.host(), clipError(string(raw)))
	}
	return nil
}

func (o OpenAI) setHeaders(req *http.Request, contentType string) {
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", contentType)
}

func (o OpenAI) responseError(resp *http.Response) error {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("reading error response: %w", err)
	}
	var apiErr struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &apiErr) == nil && apiErr.Error != nil {
		return fmt.Errorf("%s: HTTP %d: %s", o.host(), resp.StatusCode, clipError(apiErr.Error.Message))
	}
	return fmt.Errorf("%s: HTTP %d: %s", o.host(), resp.StatusCode, clipError(string(raw)))
}

func clipError(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxErrorBytes {
		return s[:maxErrorBytes] + " (truncated)"
	}
	return s
}

// host names the backend in errors, e.g. "api.groq.com".
func (o OpenAI) host() string {
	if u, err := url.Parse(o.BaseURL); err == nil && u.Host != "" {
		return u.Host
	}
	return o.BaseURL
}
