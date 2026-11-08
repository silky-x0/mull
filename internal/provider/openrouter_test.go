package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A fake server stands in for OpenRouter, so the test needs no key or network.
func TestOpenRouterComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth header = %q", got)
		}
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "m" || len(req.Messages) != 2 || req.Messages[1].Content != "hi" {
			t.Errorf("bad request body: %+v", req)
		}
		w.Write([]byte(`{"model":"vendor/actual","choices":[{"message":{"role":"assistant","content":"Hello."}}]}`))
	}))
	defer srv.Close()

	o := OpenRouter{BaseURL: srv.URL, APIKey: "test-key", Model: "m"}
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

func TestOpenRouterError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"No auth credentials found"}}`))
	}))
	defer srv.Close()

	_, err := OpenRouter{BaseURL: srv.URL}.Complete(context.Background(), "x", "y")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestOpenRouterTimeout(t *testing.T) {
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
	_, err := OpenRouter{BaseURL: srv.URL}.Complete(ctx, "x", "y")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}
