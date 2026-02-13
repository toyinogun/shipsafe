package vcs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

func TestGitHubProvider_PostComment(t *testing.T) {
	var gotPath, gotAuth, gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	provider := NewGitHubProvider("myorg", "myrepo", "test-token", server.URL)
	err := provider.PostComment(context.Background(), "42", "Hello from ShipSafe")

	if err != nil {
		t.Fatalf("PostComment returned error: %v", err)
	}

	if gotPath != "/repos/myorg/myrepo/issues/42/comments" {
		t.Errorf("unexpected path: %s", gotPath)
	}

	if gotAuth != "Bearer test-token" {
		t.Errorf("unexpected auth header: %s", gotAuth)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if payload["body"] != "Hello from ShipSafe" {
		t.Errorf("unexpected comment body: %q", payload["body"])
	}
}

func TestGitHubProvider_PostComment_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer server.Close()

	provider := NewGitHubProvider("myorg", "myrepo", "bad-token", server.URL)
	err := provider.PostComment(context.Background(), "1", "test")

	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGitHubProvider_SetStatus(t *testing.T) {
	tests := []struct {
		name          string
		status        interfaces.StatusState
		expectedState string
	}{
		{"pending", interfaces.StatusPending, "pending"},
		{"success", interfaces.StatusSuccess, "success"},
		{"failure", interfaces.StatusFailure, "failure"},
		{"error", interfaces.StatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotAuth string
			var gotPayload map[string]string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAuth = r.Header.Get("Authorization")
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &gotPayload)
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()

			provider := NewGitHubProvider("myorg", "myrepo", "gh-token", server.URL)
			err := provider.SetStatus(context.Background(), "abc123", tt.status, "ShipSafe: 85/100 GREEN")

			if err != nil {
				t.Fatalf("SetStatus returned error: %v", err)
			}

			if gotPath != "/repos/myorg/myrepo/statuses/abc123" {
				t.Errorf("unexpected path: %s", gotPath)
			}

			if gotAuth != "Bearer gh-token" {
				t.Errorf("unexpected auth header: %s", gotAuth)
			}

			if gotPayload["state"] != tt.expectedState {
				t.Errorf("expected state %q, got %q", tt.expectedState, gotPayload["state"])
			}

			if gotPayload["description"] != "ShipSafe: 85/100 GREEN" {
				t.Errorf("unexpected description: %q", gotPayload["description"])
			}

			if gotPayload["context"] != "shipsafe" {
				t.Errorf("unexpected context: %q", gotPayload["context"])
			}
		})
	}
}

func TestGitHubProvider_SetStatus_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"validation failed"}`))
	}))
	defer server.Close()

	provider := NewGitHubProvider("myorg", "myrepo", "token", server.URL)
	err := provider.SetStatus(context.Background(), "abc123", interfaces.StatusSuccess, "test")

	if err == nil {
		t.Fatal("expected error for 422 response")
	}
}

func TestGitHubProvider_GetDiff(t *testing.T) {
	sampleDiff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main

+// added line
 func main() {}
`

	var gotAccept string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleDiff))
	}))
	defer server.Close()

	provider := NewGitHubProvider("myorg", "myrepo", "token", server.URL)
	diff, err := provider.GetDiff(context.Background(), "10")

	if err != nil {
		t.Fatalf("GetDiff returned error: %v", err)
	}

	if gotAccept != "application/vnd.github.diff" {
		t.Errorf("unexpected Accept header: %s", gotAccept)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}

	if diff.Files[0].Path != "main.go" {
		t.Errorf("unexpected file path: %s", diff.Files[0].Path)
	}
}

func TestGitHubProvider_NoAuth(t *testing.T) {
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	provider := NewGitHubProvider("myorg", "myrepo", "", server.URL)
	_ = provider.PostComment(context.Background(), "1", "test")

	if gotAuth != "" {
		t.Errorf("expected no auth header, got %q", gotAuth)
	}
}
