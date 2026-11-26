package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

// A fake server stands in for the API, so the test needs no key or network.
func TestOpenAIComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth header = %q", got)
		}
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Model != "m" || len(req.Messages) != 2 || req.Messages[1].Content != "hi" {
			t.Errorf("bad request body: %+v", req)
		}
		w.Write([]byte(`{"model":"vendor/actual","choices":[{"message":{"role":"assistant","content":"Hello."}}]}`))
	}))
	defer srv.Close()

	o := OpenAI{BaseURL: srv.URL, APIKey: "test-key", Model: "m"}
	got, err := o.Complete(context.Background(), "rewrite", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Hello." {
		t.Errorf("text = %q, want %q", got.Text, "Hello.")
	}
	if got.Model != "vendor/actual" {
		t.Errorf("model = %q, want %q", got.Model, "vendor/actual")
	}
}

func TestOpenAIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"No auth credentials found"}}`))
	}))
	defer srv.Close()

	_, err := OpenAI{BaseURL: srv.URL}.Complete(context.Background(), "x", "y")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestOpenAITimeout(t *testing.T) {
	// The server only notices a client hang-up after reading the body, so the
	// handler waits on this channel instead of r.Context().
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := OpenAI{BaseURL: srv.URL}.Complete(ctx, "x", "y")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestOpenAITranscribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/transcriptions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer k" {
			t.Errorf("auth header = %q", got)
		}
		if got := r.FormValue("model"); got != "whisper-x" {
			t.Errorf("model = %q", got)
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			t.Errorf("read uploaded file: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		data, _ := io.ReadAll(f)
		if hdr.Filename != "speech.wav" || string(data) != "RIFFfake" {
			t.Errorf("file = %q %q", hdr.Filename, data)
		}
		w.Write([]byte(`{"text":" Hello there."}`))
	}))
	defer srv.Close()

	got, err := OpenAI{BaseURL: srv.URL, APIKey: "k", Model: "whisper-x"}.
		Transcribe(context.Background(), "speech.wav", strings.NewReader("RIFFfake"))
	if err != nil {
		t.Fatal(err)
	}
	if got != " Hello there." {
		t.Errorf("got %q", got)
	}
}

func TestOpenAIStream(t *testing.T) {
	var req chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"model\":\"vendor/stream\",\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo.\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var deltas []string
	got, err := OpenAI{BaseURL: srv.URL, APIKey: "k", Model: "m"}.
		Stream(context.Background(), "rewrite", "hi", func(s string) { deltas = append(deltas, s) })
	if err != nil {
		t.Fatal(err)
	}
	if !req.Stream {
		t.Error("request did not set stream: true")
	}
	if !slices.Equal(deltas, []string{"Hel", "lo."}) {
		t.Errorf("deltas = %q", deltas)
	}
	if got.Text != "Hello." || got.Model != "vendor/stream" {
		t.Errorf("reply = %+v", got)
	}
}

func TestOpenAIErrorBodyIsClipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(strings.Repeat("<html>broken gateway</html>", 1000)))
	}))
	defer srv.Close()

	_, err := OpenAI{BaseURL: srv.URL}.Complete(context.Background(), "x", "y")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if len(err.Error()) > maxErrorBytes+200 {
		t.Errorf("error is %d bytes, want it clipped near %d", len(err.Error()), maxErrorBytes)
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Errorf("error %q is missing the truncation marker", err)
	}
}

func TestOpenAIErrorMessageIsClipped(t *testing.T) {
	long := strings.Repeat("no auth ", 1000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": long}})
	}))
	defer srv.Close()

	_, err := OpenAI{BaseURL: srv.URL}.Complete(context.Background(), "x", "y")
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("err = %v, want a clipped API error", err)
	}
}
