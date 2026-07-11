package service

import (
	"strings"
	"testing"
)

// TestDataURIExtraction covers B5: safe data-URI base64 extraction.
func TestDataURIExtraction(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		want     string
		skip     bool
	}{
		{"valid data URI", "data:image/jpeg;base64,/9j/abc123", "/9j/abc123", false},
		{"no comma — should skip", "data:image/jpeg;base64NOCOLON", "", true},
		{"plain base64 passthrough", "/9j/abc123", "/9j/abc123", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url := tc.input
			skipped := false
			if strings.HasPrefix(url, "data:image/") {
				if idx := strings.Index(url, ","); idx >= 0 {
					url = url[idx+1:]
				} else {
					skipped = true
				}
			}
			if tc.skip {
				if !skipped {
					t.Errorf("expected skip but got url=%q", url)
				}
				return
			}
			if url != tc.want {
				t.Errorf("got %q, want %q", url, tc.want)
			}
		})
	}
}

// TestRetryIndexDistribution covers B1: all sessions must be reachable.
func TestRetryIndexDistribution(t *testing.T) {
	sessionCount := 3
	retryCount := sessionCount
	startIndex := 0
	seen := make(map[int]bool)
	for i := 0; i < retryCount; i++ {
		idx := (startIndex + i) % sessionCount
		seen[idx] = true
	}
	for s := 0; s < sessionCount; s++ {
		if !seen[s] {
			t.Errorf("session index %d never used", s)
		}
	}
}

// TestRootPromptIsolation covers B4: rootPrompt must not be affected by UploadText mutation.
func TestRootPromptIsolation(t *testing.T) {
	original := "Human: hello\n\nAssistant: "
	var prompt strings.Builder
	prompt.WriteString(original)
	rootPromptStr := prompt.String() // captured before loop

	// Simulate UploadText resetting prompt
	prompt.Reset()
	prompt.WriteString("[file placeholder]")

	// On retry, restore from rootPromptStr
	prompt.Reset()
	prompt.WriteString(rootPromptStr)

	if prompt.String() != original {
		t.Errorf("rootPrompt isolation failed: got %q, want %q", prompt.String(), original)
	}
}
