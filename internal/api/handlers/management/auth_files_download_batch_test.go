package management

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

func newAuthFilesArchiveTestHandler(t *testing.T, files map[string]string) *Handler {
	t.Helper()

	authDir := t.TempDir()
	for name, payload := range files {
		if err := os.WriteFile(filepath.Join(authDir, name), []byte(payload), 0o600); err != nil {
			t.Fatalf("failed to write auth file %s: %v", name, err)
		}
	}
	return NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, nil)
}

func requestAuthFilesArchive(t *testing.T, h *Handler, query string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v8/management/credentials/download/archive"+query, nil)
	h.DownloadAuthFilesArchive(ctx)
	return rec
}

func TestDownloadAuthFilesArchivePackagesSelectedFiles(t *testing.T) {
	h := newAuthFilesArchiveTestHandler(t, map[string]string{
		"antigravity-a.json": `{"type":"antigravity","email":"a@example.com"}`,
		"antigravity-b.json": `{"type":"antigravity","email":"b@example.com"}`,
		"other.json":         `{"type":"codex"}`,
	})

	query := "?name=" + url.QueryEscape("antigravity-a.json") + "&name=" + url.QueryEscape("antigravity-b.json")
	rec := requestAuthFilesArchive(t, h, query)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("Content-Type = %q, want application/zip", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got == "" {
		t.Fatal("expected a Content-Disposition attachment header")
	}

	reader, errReader := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if errReader != nil {
		t.Fatalf("response is not a valid zip archive: %v", errReader)
	}
	if len(reader.File) != 2 {
		t.Fatalf("archive entry count = %d, want 2", len(reader.File))
	}

	entries := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		handle, errOpen := file.Open()
		if errOpen != nil {
			t.Fatalf("failed to open archive entry %s: %v", file.Name, errOpen)
		}
		content := new(bytes.Buffer)
		if _, errCopy := content.ReadFrom(handle); errCopy != nil {
			t.Fatalf("failed to read archive entry %s: %v", file.Name, errCopy)
		}
		if errClose := handle.Close(); errClose != nil {
			t.Fatalf("failed to close archive entry %s: %v", file.Name, errClose)
		}
		entries[file.Name] = content.String()
	}

	if got := entries["antigravity-a.json"]; got != `{"type":"antigravity","email":"a@example.com"}` {
		t.Fatalf("entry antigravity-a.json = %q", got)
	}
	if got := entries["antigravity-b.json"]; got != `{"type":"antigravity","email":"b@example.com"}` {
		t.Fatalf("entry antigravity-b.json = %q", got)
	}
	if _, exists := entries["other.json"]; exists {
		t.Fatal("archive must only contain the requested files")
	}
}

func TestDownloadAuthFilesArchiveRejectsInvalidInput(t *testing.T) {
	h := newAuthFilesArchiveTestHandler(t, map[string]string{"present.json": `{"type":"codex"}`})

	for _, tc := range []struct {
		name   string
		query  string
		status int
	}{
		{name: "missing name", query: "", status: http.StatusBadRequest},
		{name: "blank name", query: "?name=" + url.QueryEscape("   "), status: http.StatusBadRequest},
		{name: "path traversal", query: "?name=" + url.QueryEscape("../secret.json"), status: http.StatusBadRequest},
		{name: "nested path", query: "?name=" + url.QueryEscape("nested/present.json"), status: http.StatusBadRequest},
		{name: "not json", query: "?name=" + url.QueryEscape("present.txt"), status: http.StatusBadRequest},
		{name: "not found", query: "?name=" + url.QueryEscape("missing.json"), status: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := requestAuthFilesArchive(t, h, tc.query)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.status, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got == "application/zip" {
				t.Fatal("rejected request must not start a zip response")
			}
		})
	}
}

func TestDownloadAuthFilesArchiveDeduplicatesNames(t *testing.T) {
	h := newAuthFilesArchiveTestHandler(t, map[string]string{"dup.json": `{"type":"codex"}`})

	query := "?name=" + url.QueryEscape("dup.json") + "&name=" + url.QueryEscape("dup.json")
	rec := requestAuthFilesArchive(t, h, query)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	reader, errReader := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if errReader != nil {
		t.Fatalf("response is not a valid zip archive: %v", errReader)
	}
	if len(reader.File) != 1 {
		t.Fatalf("archive entry count = %d, want 1", len(reader.File))
	}
}
