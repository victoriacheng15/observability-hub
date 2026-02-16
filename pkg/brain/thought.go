package brain

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type AtomicThought struct {
	Date          string   `json:"date"`
	Content       string   `json:"content"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	ContextString string   `json:"context_string"`
	Checksum      string   `json:"checksum"`
	TokenCount    int      `json:"token_count"`
}

var tagsMap = map[string][]string{
	"ai":            {"ai", "llm", "rag", "pgvector", "embedding", "openai", "gemini", "agent", "deepseek"},
	"observability": {"grafana", "loki", "alloy", "opentelemetry", "otel", "metrics", "tracing", "logs", "prometheus"},
	"kubernetes":    {"kubernetes", "k3s", "pod", "pvc", "statefulset", "daemonset", "kubectl", "helm", "k8s"},
	"database":      {"postgres", "postgresql", "jsonb", "timescaledb", "postgis", "sql", "mongodb", "database", "db"},
	"devops":        {"github actions", "gitops", "reconciliation", "ci-cd", "docker", "terraform", "nix", "shell.nix"},
	"career":        {"impostor syndrome", "growth", "senior", "leadership", "reflection", "mentorship", "career", "brag", "win", "impact", "job", "application", "interview", "resume", "cv"},
	"platform":      {"openbao", "tailscale", "security", "infrastructure", "infra", "zero-trust", "secrets", "bao", "cloud", "platform", "homelab"},
	"cloud":         {"aws", "azure", "gcp", "digitalocean", "atlas", "cloudflare", "cloud-native", "serverless"},
	"sre":           {"incident", "rca", "post-mortem", "outage", "reliability", "slo", "sli", "error budget", "toil"},
	"language":      {"go", "golang", "python", "rust", "typescript", "javascript", "bash", "shell"},
	"linux":         {"linux", "systemd", "kernel", "gpu", "psu", "hardware", "cpu", "memory", "nixos"},
	"productivity":  {"para", "second brain", "zettelkasten", "notion", "obsidian", "journal", "workflow"},
}

// Atomize parses a journal entry body into a slice of AtomicThoughts.
func Atomize(date, body string) []AtomicThought {
	scanner := bufio.NewScanner(strings.NewReader(body))
	var atoms []AtomicThought
	var currentBlocks []string
	capturing := false
	inComment := false
	currentCategory := "resource"

	flush := func() {
		if len(currentBlocks) > 0 {
			text := strings.TrimSpace(strings.Join(currentBlocks, "\n"))
			// Remove leading dash if present
			if strings.HasPrefix(text, "- ") {
				text = strings.TrimPrefix(text, "- ")
			} else if text == "-" {
				text = ""
			}

			text = strings.TrimSpace(text)
			if text == "" || text == "*" {
				return
			}
			tags := GetTags(text)

			// If no tags are found, add "random" tag
			if len(tags) == 0 {
				tags = []string{"random"}
			}

			context := fmt.Sprintf("Date: %s | Category: %s | Tags: %s | Content: %s", date, currentCategory, strings.Join(tags, ", "), text)
			atoms = append(atoms, AtomicThought{
				Date:          date,
				Content:       text,
				Category:      currentCategory,
				Tags:          tags,
				ContextString: context,
				Checksum:      GetChecksum(text),
				TokenCount:    EstimateTokens(context),
			})
			currentBlocks = nil
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 1. Skip Comments
		if !inComment && strings.Contains(line, "<!--") {
			inComment = true
		}
		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}

		// 2. Detect Singular PARA Headers
		if strings.HasPrefix(trimmed, "## ") {
			if capturing {
				flush()
			}

			switch trimmed {
			case "## Project":
				currentCategory = "project"
				capturing = true
				continue
			case "## Area":
				currentCategory = "area"
				capturing = true
				continue
			case "## Resource":
				currentCategory = "resource"
				capturing = true
				continue
			case "## Archive":
				currentCategory = "archive"
				capturing = true
				continue
			case "## Thought":
				currentCategory = "resource"
				capturing = true
				continue
			default:
				// Today's Task and any other header should stop capturing
				capturing = false
			}
		}

		// 3. Stop capturing on separator
		if capturing && strings.HasPrefix(trimmed, "---") {
			flush()
			capturing = false
			continue
		}

		// 4. Capture Content (Skip actual task lines just in case they are nested)
		if capturing {
			if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") {
				continue
			}

			// Detect a dash/bullet at start of line
			isBullet := strings.HasPrefix(trimmed, "-")

			// If we see a new bullet point and we already have content, flush the previous one
			if isBullet && len(currentBlocks) > 0 {
				flush()
			}

			if trimmed != "" {
				// Clean the dash from the line immediately
				lineClean := trimmed
				if strings.HasPrefix(trimmed, "- ") {
					lineClean = strings.TrimPrefix(trimmed, "- ")
				} else if trimmed == "-" {
					lineClean = ""
				} else if strings.HasPrefix(trimmed, "-") {
					// Handle cases like "-Thought" (no space)
					lineClean = strings.TrimPrefix(trimmed, "-")
				}

				if lineClean != "" {
					currentBlocks = append(currentBlocks, lineClean)
				}
			}
		}
	}
	flush()
	return atoms
}

// GetTags returns a sorted list of categories based on keywords found in the text.
func GetTags(text string) []string {
	textLower := strings.ToLower(text)
	foundMap := make(map[string]bool)
	for tag, keywords := range tagsMap {
		for _, kw := range keywords {
			if strings.Contains(textLower, kw) {
				foundMap[tag] = true
				break
			}
		}
	}
	var found []string
	for tag := range foundMap {
		found = append(found, tag)
	}
	sort.Strings(found)
	return found
}

// GetChecksum calculates a SHA256 checksum of the provided text.
func GetChecksum(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// EstimateTokens provides a rough estimate of token count for LLM context.
func EstimateTokens(text string) int {
	words := strings.Fields(text)
	return int(float64(len(words))*1.33) + 1
}
