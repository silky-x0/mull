package main

import (
	"testing"
)

func TestAskQuestion(t *testing.T) {
	tests := []struct {
		args []string
		want string
		err  bool
	}{
		{[]string{"what", "is", "a", "goroutine?"}, "what is a goroutine?", false},
		{[]string{"why?"}, "why?", false},
		{nil, "", true},
	}
	for _, tt := range tests {
		got, err := askQuestion(tt.args)
		if got != tt.want || (err != nil) != tt.err {
			t.Errorf("askQuestion(%q) = %q, %v; want %q, err=%v", tt.args, got, err, tt.want, tt.err)
		}
	}
}
