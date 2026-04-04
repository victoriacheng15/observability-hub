package brain

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GitHubIssue represents a simplified issue structure from the gh CLI.
type GitHubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// BrainAPI defines the interface for fetching journal data from external sources.
type BrainAPI interface {
	FetchRecentJournals(repo string) ([]GitHubIssue, error)
	FetchIssueBody(repo string, number int) (string, error)
}

// RealBrainAPI is the production implementation using the GitHub REST API.
type RealBrainAPI struct {
	Token  string
	Client *http.Client
}

func NewBrainAPI(token string) BrainAPI {
	return &RealBrainAPI{
		Token: token,
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (r *RealBrainAPI) FetchRecentJournals(repo string) ([]GitHubIssue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues?labels=journal&state=all&per_page=50&sort=created&direction=desc", repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if r.Token != "" {
		req.Header.Set("Authorization", "token "+r.Token)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues from github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api returned status %d: %s", resp.StatusCode, string(body))
	}

	var issues []GitHubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("failed to decode github api response: %w", err)
	}

	return issues, nil
}

func (r *RealBrainAPI) FetchIssueBody(repo string, number int) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", repo, number)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if r.Token != "" {
		req.Header.Set("Authorization", "token "+r.Token)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch issue body from github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github api returned status %d: %s", resp.StatusCode, string(body))
	}

	var issue struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return "", fmt.Errorf("failed to decode github api response: %w", err)
	}

	return issue.Body, nil
}
