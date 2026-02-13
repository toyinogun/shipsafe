package vcs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// MockVCSProvider implements interfaces.VCSProvider for testing.
// It returns canned data and records all operations for assertion.
type MockVCSProvider struct {
	mu sync.Mutex

	// DiffToReturn is the canned diff returned by GetDiff.
	DiffToReturn *interfaces.Diff
	// DiffError is returned by GetDiff if non-nil.
	DiffError error

	// Comments records all PostComment calls.
	Comments []MockComment
	// CommentError is returned by PostComment if non-nil.
	CommentError error

	// Statuses records all SetStatus calls.
	Statuses []MockStatus
	// StatusError is returned by SetStatus if non-nil.
	StatusError error
}

// MockComment records a PostComment call.
type MockComment struct {
	PRRef string
	Body  string
}

// MockStatus records a SetStatus call.
type MockStatus struct {
	SHA         string
	State       interfaces.StatusState
	Description string
}

func (m *MockVCSProvider) GetDiff(ctx context.Context, prRef string) (*interfaces.Diff, error) {
	if m.DiffError != nil {
		return nil, m.DiffError
	}
	if m.DiffToReturn == nil {
		return nil, errors.New("mock: no diff configured")
	}
	return m.DiffToReturn, nil
}

func (m *MockVCSProvider) PostComment(ctx context.Context, prRef string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CommentError != nil {
		return m.CommentError
	}
	m.Comments = append(m.Comments, MockComment{PRRef: prRef, Body: body})
	return nil
}

func (m *MockVCSProvider) SetStatus(ctx context.Context, sha string, status interfaces.StatusState, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.StatusError != nil {
		return m.StatusError
	}
	m.Statuses = append(m.Statuses, MockStatus{SHA: sha, State: status, Description: description})
	return nil
}

// Compile-time interface check.
var _ interfaces.VCSProvider = (*MockVCSProvider)(nil)

func TestMockVCSProvider_GetDiff_ReturnsCannedDiff(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "main.go", Status: interfaces.FileModified},
		},
	}
	mock := &MockVCSProvider{DiffToReturn: diff}
	got, err := mock.GetDiff(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Files) != 1 || got.Files[0].Path != "main.go" {
		t.Errorf("unexpected diff: %+v", got)
	}
}

func TestMockVCSProvider_GetDiff_ReturnsError(t *testing.T) {
	mock := &MockVCSProvider{DiffError: errors.New("connection refused")}
	_, err := mock.GetDiff(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "connection refused" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockVCSProvider_GetDiff_NoDiffConfigured(t *testing.T) {
	mock := &MockVCSProvider{}
	_, err := mock.GetDiff(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error when no diff configured")
	}
}

func TestMockVCSProvider_PostComment_RecordsComment(t *testing.T) {
	mock := &MockVCSProvider{}
	err := mock.PostComment(context.Background(), "42", "ShipSafe: 85/100 GREEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(mock.Comments))
	}
	if mock.Comments[0].PRRef != "42" {
		t.Errorf("expected PR ref %q, got %q", "42", mock.Comments[0].PRRef)
	}
	if mock.Comments[0].Body != "ShipSafe: 85/100 GREEN" {
		t.Errorf("unexpected comment body: %q", mock.Comments[0].Body)
	}
}

func TestMockVCSProvider_PostComment_ReturnsError(t *testing.T) {
	mock := &MockVCSProvider{CommentError: errors.New("forbidden")}
	err := mock.PostComment(context.Background(), "1", "test")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMockVCSProvider_SetStatus_RecordsStatus(t *testing.T) {
	mock := &MockVCSProvider{}
	err := mock.SetStatus(context.Background(), "abc123", interfaces.StatusSuccess, "ShipSafe: 85/100 GREEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.Statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(mock.Statuses))
	}
	if mock.Statuses[0].SHA != "abc123" {
		t.Errorf("expected SHA %q, got %q", "abc123", mock.Statuses[0].SHA)
	}
	if mock.Statuses[0].State != interfaces.StatusSuccess {
		t.Errorf("expected state %q, got %q", interfaces.StatusSuccess, mock.Statuses[0].State)
	}
	if mock.Statuses[0].Description != "ShipSafe: 85/100 GREEN" {
		t.Errorf("unexpected description: %q", mock.Statuses[0].Description)
	}
}

func TestMockVCSProvider_SetStatus_ReturnsError(t *testing.T) {
	mock := &MockVCSProvider{StatusError: errors.New("unauthorized")}
	err := mock.SetStatus(context.Background(), "abc", interfaces.StatusSuccess, "test")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMockVCSProvider_MultipleComments(t *testing.T) {
	mock := &MockVCSProvider{}
	_ = mock.PostComment(context.Background(), "1", "first")  // best-effort test setup
	_ = mock.PostComment(context.Background(), "1", "second") // best-effort test setup
	_ = mock.PostComment(context.Background(), "2", "third")  // best-effort test setup

	if len(mock.Comments) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(mock.Comments))
	}
}

func TestMockVCSProvider_MultipleStatuses(t *testing.T) {
	mock := &MockVCSProvider{}
	_ = mock.SetStatus(context.Background(), "sha1", interfaces.StatusPending, "pending") // best-effort test setup
	_ = mock.SetStatus(context.Background(), "sha1", interfaces.StatusSuccess, "success") // best-effort test setup

	if len(mock.Statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(mock.Statuses))
	}
	if mock.Statuses[0].State != interfaces.StatusPending {
		t.Errorf("first status expected pending, got %q", mock.Statuses[0].State)
	}
	if mock.Statuses[1].State != interfaces.StatusSuccess {
		t.Errorf("second status expected success, got %q", mock.Statuses[1].State)
	}
}

func TestMockVCSProvider_WithParsedDiff(t *testing.T) {
	raw := []byte("diff --git a/hello.go b/hello.go\n" +
		"new file mode 100644\n" +
		"index 0000000..1234567\n" +
		"--- /dev/null\n" +
		"+++ b/hello.go\n" +
		"@@ -0,0 +1,5 @@\n" +
		"+package main\n" +
		"+\n" +
		"+func hello() string {\n" +
		"+\treturn \"world\"\n" +
		"+}\n")

	parser := NewDiffParser()
	diff, err := parser.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	mock := &MockVCSProvider{DiffToReturn: diff}
	got, err := mock.GetDiff(context.Background(), "99")
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}

	if len(got.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(got.Files))
	}
	if got.Files[0].Path != "hello.go" {
		t.Errorf("expected path %q, got %q", "hello.go", got.Files[0].Path)
	}
	if got.Files[0].Status != interfaces.FileAdded {
		t.Errorf("expected status %q, got %q", interfaces.FileAdded, got.Files[0].Status)
	}
}

func TestMockVCSProvider_ErrorDoesNotRecordComment(t *testing.T) {
	mock := &MockVCSProvider{CommentError: errors.New("server error")}
	err := mock.PostComment(context.Background(), "1", "should not be recorded")
	if err == nil {
		t.Fatal("expected error")
	}
	if len(mock.Comments) != 0 {
		t.Errorf("expected 0 comments after error, got %d", len(mock.Comments))
	}
}

func TestMockVCSProvider_ErrorDoesNotRecordStatus(t *testing.T) {
	mock := &MockVCSProvider{StatusError: errors.New("server error")}
	err := mock.SetStatus(context.Background(), "sha", interfaces.StatusSuccess, "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if len(mock.Statuses) != 0 {
		t.Errorf("expected 0 statuses after error, got %d", len(mock.Statuses))
	}
}
