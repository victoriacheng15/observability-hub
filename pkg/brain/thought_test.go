package brain

import (
	"testing"
)

func TestAtomize(t *testing.T) {
	date := "2026-02-16"
	body := `
## Thought
- This is a test thought.
- Another thought.

## Project
- [ ] Active task (should be skipped)
- Completed task (should be captured)

---
Ignored content after separator.
`
	atoms := Atomize(date, body)

	// Expected:
	// 1. "This is a test thought." (Category: resource)
	// 2. "Another thought." (Category: resource)
	// 3. "Completed task (should be captured)" (Category: project)

	if len(atoms) != 3 {
		t.Fatalf("Expected 3 atoms, got %d", len(atoms))
	}

	if atoms[0].Content != "This is a test thought." || atoms[0].Category != "resource" {
		t.Errorf("Atom 0 mismatch: %+v", atoms[0])
	}

	if atoms[2].Content != "Completed task (should be captured)" || atoms[2].Category != "project" {
		t.Errorf("Atom 2 mismatch: %+v", atoms[2])
	}
}

func TestGetTags(t *testing.T) {
	text := "I am learning about OpenTelemetry and Kubernetes in my Homelab."
	tags := GetTags(text)

	// Expected: "observability", "kubernetes", "platform"
	expected := map[string]bool{
		"observability": true,
		"kubernetes":    true,
		"platform":      true,
	}

	for _, tag := range tags {
		delete(expected, tag)
	}

	if len(expected) > 0 {
		t.Errorf("Missing expected tags: %v", expected)
	}
}

func TestGetChecksum(t *testing.T) {
	t1 := "hello world"
	t2 := "hello world"
	t3 := "different"

	if GetChecksum(t1) != GetChecksum(t2) {
		t.Error("Checksums for identical text should match")
	}

	if GetChecksum(t1) == GetChecksum(t3) {
		t.Error("Checksums for different text should not match")
	}
}
