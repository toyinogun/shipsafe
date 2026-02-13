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

func TestForgejoProvider_PostComment(t *testing.T) {
	var gotPath, gotAuth, gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	provider := NewForgejoProvider("myorg", "myrepo", "forgejo-token", server.URL)
	err := provider.PostComment(context.Background(), "7", "ShipSafe report here")

	if err != nil {
		t.Fatalf("PostComment returned error: %v", err)
	}

	if gotPath != "/api/v1/repos/myorg/myrepo/issues/7/comments" {
		t.Errorf("unexpected path: %s", gotPath)
	}

	if gotAuth != "token forgejo-token" {
		t.Errorf("unexpected auth header: %s", gotAuth)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if payload["body"] != "ShipSafe report here" {
		t.Errorf("unexpected comment body: %q", payload["body"])
	}
}

func TestForgejoProvider_PostComment_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer server.Close()

	provider := NewForgejoProvider("myorg", "myrepo", "token", server.URL)
	err := provider.PostComment(context.Background(), "999", "test")

	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestForgejoProvider_SetStatus(t *testing.T) {
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

			provider := NewForgejoProvider("myorg", "myrepo", "fj-token", server.URL)
			err := provider.SetStatus(context.Background(), "def456", tt.status, "ShipSafe: 45/100 RED")

			if err != nil {
				t.Fatalf("SetStatus returned error: %v", err)
			}

			if gotPath != "/api/v1/repos/myorg/myrepo/statuses/def456" {
				t.Errorf("unexpected path: %s", gotPath)
			}

			if gotAuth != "token fj-token" {
				t.Errorf("unexpected auth header: %s", gotAuth)
			}

			if gotPayload["state"] != tt.expectedState {
				t.Errorf("expected state %q, got %q", tt.expectedState, gotPayload["state"])
			}

			if gotPayload["description"] != "ShipSafe: 45/100 RED" {
				t.Errorf("unexpected description: %q", gotPayload["description"])
			}

			if gotPayload["context"] != "shipsafe" {
				t.Errorf("unexpected context: %q", gotPayload["context"])
			}
		})
	}
}

func TestForgejoProvider_SetStatus_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer server.Close()

	provider := NewForgejoProvider("myorg", "myrepo", "token", server.URL)
	err := provider.SetStatus(context.Background(), "abc123", interfaces.StatusSuccess, "test")

	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestForgejoProvider_GetDiff(t *testing.T) {
	sampleDiff := `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -5,3 +5,4 @@
 func handler() {
+    log.Println("debug")
 }
`

	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleDiff))
	}))
	defer server.Close()

	provider := NewForgejoProvider("myorg", "myrepo", "token", server.URL)
	diff, err := provider.GetDiff(context.Background(), "3")

	if err != nil {
		t.Fatalf("GetDiff returned error: %v", err)
	}

	if gotPath != "/api/v1/repos/myorg/myrepo/pulls/3.diff" {
		t.Errorf("unexpected path: %s", gotPath)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}

	if diff.Files[0].Path != "handler.go" {
		t.Errorf("unexpected file path: %s", diff.Files[0].Path)
	}
}

func TestForgejoProvider_AuthFormat(t *testing.T) {
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	provider := NewForgejoProvider("myorg", "myrepo", "my-secret-token", server.URL)
	_ = provider.PostComment(context.Background(), "1", "test")

	// Forgejo uses "token <value>" format, not "Bearer <value>".
	if gotAuth != "token my-secret-token" {
		t.Errorf("expected 'token my-secret-token', got %q", gotAuth)
	}
}

func TestForgejoProvider_NoAuth(t *testing.T) {
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	provider := NewForgejoProvider("myorg", "myrepo", "", server.URL)
	_ = provider.PostComment(context.Background(), "1", "test")

	if gotAuth != "" {
		t.Errorf("expected no auth header, got %q", gotAuth)
	}
}
