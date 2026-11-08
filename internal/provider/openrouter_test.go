package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Hello."}}]}`))
	}))
	defer srv.Close()

	o := OpenRouter{BaseURL: srv.URL, APIKey: "test-key", Model: "m"}
	got, err := o.Complete(context.Background(), "rewrite", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Hello." {
		t.Errorf("got %q, want %q", got, "Hello.")
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
