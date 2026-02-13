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

// ForgejoProvider implements interfaces.VCSProvider for Forgejo and Gitea instances.
type ForgejoProvider struct {
	baseURL    string
	token      string
	owner      string
	repo       string
	httpClient *http.Client
}

// NewForgejoProvider creates a Forgejo/Gitea VCS provider.
// owner/repo identifies the repository. Token is used for authentication.
// baseURL should be the Forgejo/Gitea server URL (e.g., https://codeberg.org).
func NewForgejoProvider(owner, repo, token, baseURL string) *ForgejoProvider {
	baseURL = strings.TrimRight(baseURL, "/")

	return &ForgejoProvider{
		baseURL:    baseURL,
		token:      token,
		owner:      owner,
		repo:       repo,
		httpClient: &http.Client{},
	}
}

// NewForgejoProviderFromEnv creates a ForgejoProvider using standard environment variables.
func NewForgejoProviderFromEnv() (*ForgejoProvider, error) {
	token := os.Getenv("FORGEJO_TOKEN")
	if token == "" {
		token = os.Getenv("GITEA_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("vcs: FORGEJO_TOKEN or GITEA_TOKEN not set")
	}

	serverURL := os.Getenv("CI_SERVER_URL")
	if serverURL == "" {
		serverURL = os.Getenv("GITEA_SERVER_URL")
	}
	if serverURL == "" {
		return nil, fmt.Errorf("vcs: CI_SERVER_URL or GITEA_SERVER_URL not set")
	}

	repository := os.Getenv("GITHUB_REPOSITORY")
	if repository == "" {
		return nil, fmt.Errorf("vcs: GITHUB_REPOSITORY not set (Forgejo Actions uses GitHub-compatible env vars)")
	}

	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("vcs: invalid GITHUB_REPOSITORY format %q, expected owner/repo", repository)
	}

	return NewForgejoProvider(parts[0], parts[1], token, serverURL), nil
}

// GetDiff retrieves the diff for a pull request by its index number.
func (f *ForgejoProvider) GetDiff(ctx context.Context, prRef string) (*interfaces.Diff, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%s.diff", f.baseURL, f.owner, f.repo, prRef)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("vcs: creating Forgejo diff request: %w", err)
	}

	f.setAuth(req)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vcs: fetching Forgejo diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vcs: Forgejo diff request returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vcs: reading Forgejo diff response: %w", err)
	}

	parser := NewDiffParser()
	return parser.Parse(ctx, body)
}

// PostComment posts a comment on a pull request (issue endpoint).
func (f *ForgejoProvider) PostComment(ctx context.Context, prRef string, body string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%s/comments", f.baseURL, f.owner, f.repo, prRef)

	payload := map[string]string{"body": body}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("vcs: marshaling Forgejo comment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("vcs: creating Forgejo comment request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	f.setAuth(req)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vcs: posting Forgejo comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vcs: Forgejo comment returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SetStatus sets a commit status on a given SHA.
func (f *ForgejoProvider) SetStatus(ctx context.Context, sha string, status interfaces.StatusState, description string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/statuses/%s", f.baseURL, f.owner, f.repo, sha)

	payload := map[string]string{
		"state":       forgejoStatusState(status),
		"description": description,
		"context":     "shipsafe",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("vcs: marshaling Forgejo status: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("vcs: creating Forgejo status request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	f.setAuth(req)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vcs: posting Forgejo status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vcs: Forgejo status returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (f *ForgejoProvider) setAuth(req *http.Request) {
	if f.token != "" {
		req.Header.Set("Authorization", "token "+f.token)
	}
}

// forgejoStatusState maps interfaces.StatusState to Forgejo/Gitea API status strings.
func forgejoStatusState(s interfaces.StatusState) string {
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
