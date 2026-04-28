package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/darkspock/gosnag/internal/database/db"
	projectcfg "github.com/darkspock/gosnag/internal/project"
)

// Config holds GitHub connection settings from a project.
type Config struct {
	Token  string
	Owner  string
	Repo   string
	Labels string // comma-separated label names
}

func ConfigFromSettings(settings projectcfg.ProjectSettings) Config {
	return Config{
		Token:  settings.GithubToken,
		Owner:  settings.GithubOwner,
		Repo:   settings.GithubRepo,
		Labels: settings.GithubLabels,
	}
}

// IsConfigured returns true if the project has GitHub configured.
func (c Config) IsConfigured() bool {
	return c.Token != "" && c.Owner != "" && c.Repo != ""
}

// CreateIssueResult holds the result of creating a GitHub issue.
type CreateIssueResult struct {
	Number int    `json:"number"`
	URL    string `json:"html_url"`
}

// CreateIssue creates a new issue in GitHub.
func CreateIssue(cfg Config, title, body string) (*CreateIssueResult, error) {
	payload := map[string]any{
		"title": title,
		"body":  body,
	}

	if cfg.Labels != "" {
		var labels []string
		for _, l := range strings.Split(cfg.Labels, ",") {
			l = strings.TrimSpace(l)
			if l != "" {
				labels = append(labels, l)
			}
		}
		if len(labels) > 0 {
			payload["labels"] = labels
		}
	}

	reqBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", cfg.Owner, cfg.Repo)
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result CreateIssueResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return &result, nil
}

// TestConnection verifies that the GitHub config is valid by fetching the repo.
func TestConnection(cfg Config) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", cfg.Owner, cfg.Repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("authentication failed: check your GitHub token")
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("repository '%s/%s' not found", cfg.Owner, cfg.Repo)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// BuildDescription creates a formatted description for a GitHub issue from a GoSnag issue.
func BuildDescription(issue db.Issue, baseURL, projectID string, latestStacktrace string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("**Level:** %s\n", issue.Level))
	b.WriteString(fmt.Sprintf("**Platform:** %s\n", issue.Platform))
	b.WriteString(fmt.Sprintf("**Events:** %d\n", issue.EventCount))
	b.WriteString(fmt.Sprintf("**First seen:** %s\n", issue.FirstSeen.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("**Last seen:** %s\n", issue.LastSeen.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("\n[View in GoSnag](%s/projects/%s/issues/%s)\n", baseURL, projectID, issue.ID))

	if latestStacktrace != "" {
		b.WriteString(fmt.Sprintf("\n<details><summary>Stacktrace</summary>\n\n```\n%s\n```\n</details>\n", latestStacktrace))
	}

	return b.String()
}
