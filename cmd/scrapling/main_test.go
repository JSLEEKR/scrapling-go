package main

import (
	"testing"
)

func TestRunHelp(t *testing.T) {
	err := run([]string{"help"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunVersion(t *testing.T) {
	err := run([]string{"version"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunNoArgs(t *testing.T) {
	err := run([]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := run([]string{"unknown"})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestRunFetchNoURL(t *testing.T) {
	// fetch with no URL should give an error
	err := run([]string{"fetch"})
	if err == nil {
		t.Error("expected error for fetch without URL")
	}
}

func TestRunTrackNoArgs(t *testing.T) {
	err := run([]string{"track"})
	if err == nil {
		t.Error("expected error for track without args")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		n        int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		result := truncate(tt.input, tt.n)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.n, result, tt.expected)
		}
	}
}

func TestHelpFlags(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		err := run([]string{flag})
		if err != nil {
			t.Errorf("unexpected error for %s: %v", flag, err)
		}
	}
}
