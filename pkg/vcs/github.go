package vcs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// GitHubProvider implements interfaces.VCSProvider for GitHub and GitHub Enterprise.
type GitHubProvider struct {
	baseURL    string
	token      string
	owner      string
	repo       string
	httpClient *http.Client
}

// NewGitHubProvider creates a GitHub VCS provider.
// owner/repo identifies the repository. Token is used for authentication.
// If baseURL is empty, it defaults to https://api.github.com.
func NewGitHubProvider(owner, repo, token, baseURL string) *GitHubProvider {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &GitHubProvider{
		baseURL:    baseURL,
		token:      token,
		owner:      owner,
		repo:       repo,
		httpClient: &http.Client{},
	}
}

// NewGitHubProviderFromEnv creates a GitHubProvider using standard environment variables.
func NewGitHubProviderFromEnv() (*GitHubProvider, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("vcs: GITHUB_TOKEN not set")
	}

	repository := os.Getenv("GITHUB_REPOSITORY")
	if repository == "" {
		return nil, fmt.Errorf("vcs: GITHUB_REPOSITORY not set")
	}

	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("vcs: invalid GITHUB_REPOSITORY format %q, expected owner/repo", repository)
	}

	baseURL := "https://api.github.com"
	if host := os.Getenv("GH_HOST"); host != "" && host != "github.com" {
		baseURL = fmt.Sprintf("https://%s/api/v3", host)
	}
	if serverURL := os.Getenv("GITHUB_API_URL"); serverURL != "" {
		baseURL = serverURL
	}

	return NewGitHubProvider(parts[0], parts[1], token, baseURL), nil
}

// GetDiff retrieves the diff for a pull request by its number.
func (g *GitHubProvider) GetDiff(ctx context.Context, prRef string) (*interfaces.Diff, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%s", g.baseURL, g.owner, g.repo, prRef)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("vcs: creating GitHub diff request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.diff")
	g.setAuth(req)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vcs: fetching GitHub diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vcs: GitHub diff request returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vcs: reading GitHub diff response: %w", err)
	}

	parser := NewDiffParser()
	return parser.Parse(ctx, body)
}

// PostComment posts a comment on a pull request.
func (g *GitHubProvider) PostComment(ctx context.Context, prRef string, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", g.baseURL, g.owner, g.repo, prRef)

	payload := map[string]string{"body": body}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("vcs: marshaling GitHub comment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("vcs: creating GitHub comment request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	g.setAuth(req)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vcs: posting GitHub comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vcs: GitHub comment returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SetStatus sets a commit status on a given SHA.
func (g *GitHubProvider) SetStatus(ctx context.Context, sha string, status interfaces.StatusState, description string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/statuses/%s", g.baseURL, g.owner, g.repo, sha)

	payload := map[string]string{
		"state":       githubStatusState(status),
		"description": description,
		"context":     "shipsafe",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("vcs: marshaling GitHub status: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("vcs: creating GitHub status request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	g.setAuth(req)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vcs: posting GitHub status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vcs: GitHub status returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (g *GitHubProvider) setAuth(req *http.Request) {
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
}

// githubStatusState maps interfaces.StatusState to GitHub API status strings.
func githubStatusState(s interfaces.StatusState) string {
	switch s {
	case interfaces.StatusPending:
		return "pending"
	case interfaces.StatusSuccess:
		return "success"
	case interfaces.StatusFailure:
		return "failure"
	case interfaces.StatusError:
		return "error"
	default:
		return "error"
	}
}
