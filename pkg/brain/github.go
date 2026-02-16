package brain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type GitHubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// FetchRecentJournals retrieves the 50 most recent issues labeled 'journal' from the specified repo.
func FetchRecentJournals(repo string) ([]GitHubIssue, error) {
	cmd := exec.Command("gh", "issue", "list", "--repo", repo, "--label", "journal", "--state", "all", "--limit", "50", "--json", "number,title")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list issues via gh cli: %w", err)
	}

	var issues []GitHubIssue
	if err := json.Unmarshal(out.Bytes(), &issues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gh cli output: %w", err)
	}
	return issues, nil
}

// FetchIssueBody retrieves the body of a specific issue.
func FetchIssueBody(repo string, number int) (string, error) {
	cmd := exec.Command("gh", "issue", "view", fmt.Sprintf("%d", number), "--repo", repo, "--json", "body", "--jq", ".body")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to view issue via gh cli: %w", err)
	}
	return out.String(), nil
}
