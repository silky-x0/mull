package main

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestRewriteInstruction(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"default is fix", nil, presets["fix"]},
		{"preset", []string{"shorten"}, presets["shorten"]},
		{"quoted free-form", []string{"make it sound less angry"}, "as instructed: make it sound less angry."},
		{"unquoted free-form", []string{"translate", "to", "Hindi"}, "as instructed: translate to Hindi."},
		{"unknown single word", []string{"formal"}, "as instructed: formal."},
		{"custom preset", []string{"pirate"}, "Talk like a pirate."},
		{"custom preset with dash", []string{"git-msg"}, "Write a commit message."},
		{"custom overrides built-in", []string{"casual"}, "Use slang."},
	}
	t.Setenv("MULL_PRESET_PIRATE", "Talk like a pirate.")
	t.Setenv("MULL_PRESET_GIT_MSG", "Write a commit message.")
	t.Setenv("MULL_PRESET_CASUAL", "Use slang.")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteInstruction(tt.args)
			if !strings.Contains(got, tt.want) || !strings.HasSuffix(got, outputRule) {
				t.Errorf("rewriteInstruction(%q) = %q, want it to contain %q and end with the output rule", tt.args, got, tt.want)
			}
		})
	}
}

func TestPresetList(t *testing.T) {
	t.Setenv("MULL_PRESET_GIT_MSG", "x")
	t.Setenv("MULL_PRESET_CASUAL", "override, not a new entry")
	t.Setenv("MULL_PRESET_EMPTY", "")

	want := []string{"fix", "shorten", "polite", "casual", "translate", "git-msg"}
	if got := presetList(); !slices.Equal(got, want) {
		t.Errorf("presetList() = %q, want %q", got, want)
	}
}

func TestUnwrapText(t *testing.T) {
	for in, want := range map[string]string{
		"plain":                    "plain",
		"\n\n  padded \n":          "padded",
		"<text>\nechoed\n</text>":  "echoed",
		"keeps <text> inside text": "keeps <text> inside text",
	} {
		if got := unwrapText(in); got != want {
			t.Errorf("unwrapText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckRewrite(t *testing.T) {
	question := "Can you explain how React hooks work and give me an example?"
	tutorial := strings.Repeat("React hooks are functions that let you use state. ", 10)
	para := strings.Repeat("word ", 40)

	tests := []struct {
		name    string
		args    []string
		in, out string
		off     bool // want errOffTask
		err     bool // want any error
	}{
		{"fix ok", nil, "teh cat sat", "The cat sat.", false, false},
		{"fix answered", nil, question, tutorial, true, true},
		{"shorten ok", []string{"shorten"}, para, "words", false, false},
		{"shorten answered", []string{"shorten"}, question, tutorial, true, true},
		{"shorten grew", []string{"shorten"}, para, para + para, true, true},
		{"fix gutted", nil, para, "ok", true, true},
		{"short input slack", nil, "ok", "OK, sounds good to me.", false, false},
		{"free-form unchecked", []string{"make it a poem"}, "hi", tutorial, false, false},
		{"multi-word unchecked", []string{"translate", "to", "Hindi"}, "hi", tutorial, false, false},
		{"empty reply", []string{"shorten"}, para, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRewrite(tt.args, tt.in, tt.out)
			if (err != nil) != tt.err || errors.Is(err, errOffTask) != tt.off {
				t.Errorf("checkRewrite = %v, want err=%v offTask=%v", err, tt.err, tt.off)
			}
		})
	}

	t.Run("overridden built-in unchecked", func(t *testing.T) {
		t.Setenv("MULL_PRESET_FIX", "Expand into an essay.")
		if err := checkRewrite(nil, "hi", tutorial); err != nil {
			t.Errorf("got %v, want nil", err)
		}
	})
}
