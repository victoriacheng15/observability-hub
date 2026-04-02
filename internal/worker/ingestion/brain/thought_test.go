package brain

import (
	"reflect"
	"testing"
)

func TestAtomize(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		body     string
		expected []AtomicThought
	}{
		{
			name: "Basic Para Headers",
			date: "2026-02-16",
			body: `
## Thought
- This is a test thought.
- Another thought.

## Project
- Completed task (should be captured)
`,
			expected: []AtomicThought{
				{Content: "This is a test thought.", Category: "resource"},
				{Content: "Another thought.", Category: "resource"},
				{Content: "Completed task (should be captured)", Category: "project"},
			},
		},
		{
			name: "Skip Tasks and Comments",
			date: "2026-02-16",
			body: `
## Thought
<!-- This is a comment -->
- Valid thought
- [ ] Active task (skip)
- [x] Done task (skip)
<!-- 
Multiline
Comment
-->
- Another valid thought
`,
			expected: []AtomicThought{
				{Content: "Valid thought", Category: "resource"},
				{Content: "Another valid thought", Category: "resource"},
			},
		},
		{
			name: "Separator and All PARA Headers",
			date: "2026-02-16",
			body: `
## Area
- Area content
---
## Resource
- Resource content
## Archive
- Archive content
## Project
- Project content
`,
			expected: []AtomicThought{
				{Content: "Area content", Category: "area"},
				{Content: "Resource content", Category: "resource"},
				{Content: "Archive content", Category: "archive"},
				{Content: "Project content", Category: "project"},
			},
		},
		{
			name: "Bullet Formats",
			date: "2026-02-16",
			body: `
## Thought
-No space
-  Extra space
- - Nested dash
- * Bullet and star
`,
			expected: []AtomicThought{
				{Content: "No space", Category: "resource"},
				{Content: "Extra space", Category: "resource"},
				{Content: "Nested dash", Category: "resource"},
				{Content: "* Bullet and star", Category: "resource"},
			},
		},
		{
			name: "Empty and Invalid",
			date: "2026-02-16",
			body: `
## Thought
- 
- *
- -
- Just text without bullet
`,
			expected: []AtomicThought{
				{Content: "Just text without bullet", Category: "resource"},
			},
		},
		{
			name: "Non-PARA Header stops capturing",
			date: "2026-02-16",
			body: `
## Thought
- Captured
## Random Header
- Not captured
`,
			expected: []AtomicThought{
				{Content: "Captured", Category: "resource"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Atomize(tt.date, tt.body)
			if len(got) != len(tt.expected) {
				for i, a := range got {
					t.Logf("Got atom %d: %q", i, a.Content)
				}
				t.Fatalf("Expected %d atoms, got %d", len(tt.expected), len(got))
			}
			for i := range got {
				if got[i].Content != tt.expected[i].Content {
					t.Errorf("Atom %d Content mismatch: got %q, want %q", i, got[i].Content, tt.expected[i].Content)
				}
				if got[i].Category != tt.expected[i].Category {
					t.Errorf("Atom %d Category mismatch: got %q, want %q", i, got[i].Category, tt.expected[i].Category)
				}
			}
		})
	}
}

func TestGetTags(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Multiple Tags",
			text:     "I am learning about OpenTelemetry and Kubernetes in my Homelab.",
			expected: []string{"kubernetes", "observability", "platform"},
		},
		{
			name:     "AI Tags",
			text:     "Using Gemini and OpenAI for RAG.",
			expected: []string{"ai"},
		},
		{
			name:     "No Tags",
			text:     "Just some random text without keywords.",
			expected: []string{},
		},
		{
			name:     "Career and SRE",
			text:     "Reflection on incident RCA and mentorship impact.",
			expected: []string{"career", "sre"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTags(tt.text)
			if len(got) == 0 && len(tt.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GetTags() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetChecksum(t *testing.T) {
	tests := []struct {
		name      string
		t1        string
		t2        string
		wantMatch bool
	}{
		{
			name:      "Identical Text",
			t1:        "hello world",
			t2:        "hello world",
			wantMatch: true,
		},
		{
			name:      "Different Text",
			t1:        "hello world",
			t2:        "different",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (GetChecksum(tt.t1) == GetChecksum(tt.t2)) != tt.wantMatch {
				if tt.wantMatch {
					t.Errorf("Checksums for %q and %q should match", tt.t1, tt.t2)
				} else {
					t.Errorf("Checksums for %q and %q should not match", tt.t1, tt.t2)
				}
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "Basic Sentence",
			text: "This is a simple test case.",
		},
		{
			name: "Empty String",
			text: "",
		},
		{
			name: "Long String",
			text: "This is a much longer string to test the token estimation logic with more words.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if tt.text != "" && got <= 0 {
				t.Errorf("EstimateTokens(%q) returned non-positive value: %d", tt.text, got)
			}
			if tt.text == "" && got != 1 {
				t.Errorf("EstimateTokens(%q) expected 1, got %d", tt.text, got)
			}
		})
	}
}
