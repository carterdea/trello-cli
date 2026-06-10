package trello_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Scale-Flow/trello-cli/internal/trello"
)

func TestListAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/1/cards/c1/attachments" {
			t.Errorf("path = %s, want /1/cards/c1/attachments", r.URL.Path)
		}
		if err := json.NewEncoder(w).Encode([]map[string]any{
			{"id": "a1", "name": "file.txt", "url": "https://example.com/file.txt", "bytes": 4, "mimeType": "text/plain", "date": "2026-03-13T12:00:00Z", "isUpload": true},
		}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	attachments, err := client.ListAttachments(context.Background(), "c1")
	if err != nil {
		t.Fatalf("ListAttachments() error: %v", err)
	}
	if len(attachments) != 1 || attachments[0].ID != "a1" {
		t.Fatalf("attachments = %+v", attachments)
	}
}

func TestAddURLAttachment(t *testing.T) {
	var capturedQuery string
	name := "Reference"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/1/cards/c1/attachments" {
			t.Errorf("path = %s, want /1/cards/c1/attachments", r.URL.Path)
		}
		capturedQuery = r.URL.RawQuery
		if err := json.NewEncoder(w).Encode(map[string]any{
			"id": "a1", "name": name, "url": "https://example.com", "bytes": 0, "mimeType": "", "date": "2026-03-13T12:00:00Z", "isUpload": false,
		}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	attachment, err := client.AddURLAttachment(context.Background(), "c1", "https://example.com", &name)
	if err != nil {
		t.Fatalf("AddURLAttachment() error: %v", err)
	}
	if !strings.Contains(capturedQuery, "url=https%3A%2F%2Fexample.com") {
		t.Errorf("query missing url: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "name=Reference") {
		t.Errorf("query missing name: %s", capturedQuery)
	}
	if attachment.ID != "a1" {
		t.Errorf("ID = %q, want a1", attachment.ID)
	}
}

func TestAddFileAttachment(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "attachment-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp() error: %v", err)
	}
	if _, err := tempFile.WriteString("hello world"); err != nil {
		t.Fatalf("WriteString() error: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	var contentType string
	var fileName string
	var uploadedContent string
	var capturedQuery string
	var fieldName string
	var attachmentName string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/1/cards/c1/attachments" {
			t.Errorf("path = %s, want /1/cards/c1/attachments", r.URL.Path)
		}
		contentType = r.Header.Get("Content-Type")
		capturedQuery = r.URL.RawQuery

		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader() error: %v", err)
		}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("ReadAll() error: %v", err)
			}
			if part.FileName() != "" {
				fieldName = part.FormName()
				fileName = part.FileName()
				uploadedContent = string(data)
			} else if part.FormName() == "name" {
				attachmentName = string(data)
			}
		}

		if err := json.NewEncoder(w).Encode(map[string]any{
			"id": "a1", "name": "renamed.txt", "url": "https://example.com/file", "bytes": 11, "mimeType": "text/plain", "date": "2026-03-13T12:00:00Z", "isUpload": true,
		}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	name := "renamed.txt"
	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	attachment, err := client.AddFileAttachment(context.Background(), "c1", tempFile.Name(), &name)
	if err != nil {
		t.Fatalf("AddFileAttachment() error: %v", err)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Errorf("content type = %q", contentType)
	}
	if !strings.Contains(capturedQuery, "key=k") || !strings.Contains(capturedQuery, "token=t") {
		t.Errorf("query missing auth params: %s", capturedQuery)
	}
	if fieldName != "file" {
		t.Errorf("form file field = %q, want file", fieldName)
	}
	if uploadedContent != "hello world" {
		t.Errorf("uploaded content = %q, want hello world", uploadedContent)
	}
	if attachmentName != name {
		t.Errorf("attachment form name = %q, want %q", attachmentName, name)
	}
	if fileName == "" {
		t.Error("expected uploaded filename")
	}
	if attachment.ID != "a1" {
		t.Errorf("ID = %q, want a1", attachment.ID)
	}
}

func TestDeleteAttachment(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	if err := client.DeleteAttachment(context.Background(), "c1", "a1"); err != nil {
		t.Fatalf("DeleteAttachment() error: %v", err)
	}
	if capturedMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", capturedMethod)
	}
	if capturedPath != "/1/cards/c1/attachments/a1" {
		t.Errorf("path = %s, want /1/cards/c1/attachments/a1", capturedPath)
	}
}

func TestGetAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/1/cards/c1/attachments/a1" {
			t.Errorf("path = %s, want /1/cards/c1/attachments/a1", r.URL.Path)
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"id": "a1", "name": "Report", "url": "https://example.com/report.pdf", "fileName": "report.pdf", "bytes": 4, "mimeType": "application/pdf", "date": "2026-03-13T12:00:00Z", "isUpload": true,
		}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	attachment, err := client.GetAttachment(context.Background(), "c1", "a1")
	if err != nil {
		t.Fatalf("GetAttachment() error: %v", err)
	}
	if attachment.ID != "a1" || attachment.FileName != "report.pdf" {
		t.Fatalf("attachment = %+v", attachment)
	}
}

func TestDownloadAttachmentWritesFile(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1/cards/c1/attachments/a1":
			if err := json.NewEncoder(w).Encode(map[string]any{
				"id": "a1", "name": "file.txt", "url": serverURL + "/download/file.txt", "fileName": "file.txt", "bytes": 5, "mimeType": "text/plain", "date": "2026-03-13T12:00:00Z", "isUpload": true,
			}); err != nil {
				t.Fatalf("Encode() error: %v", err)
			}
		case "/download/file.txt":
			wantAuth := `OAuth oauth_consumer_key="k", oauth_token="t"`
			if r.Header.Get("Authorization") != wantAuth {
				t.Errorf("download authorization = %q, want %q", r.Header.Get("Authorization"), wantAuth)
			}
			if r.URL.RawQuery != "" {
				t.Errorf("download query = %q, want empty", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte("hello"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "saved.txt")
	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	result, err := client.DownloadAttachment(context.Background(), "c1", "a1", outputPath, false)
	if err != nil {
		t.Fatalf("DownloadAttachment() error: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("downloaded data = %q, want hello", string(data))
	}
	if result.Path != outputPath || result.Bytes != 5 {
		t.Fatalf("result = %+v", result)
	}
}

func TestDownloadAttachmentDirectoryOutputUsesSanitizedFileName(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1/cards/c1/attachments/a1":
			if err := json.NewEncoder(w).Encode(map[string]any{
				"id": "a1", "name": "ignored.txt", "url": serverURL + "/download/report.pdf", "fileName": "../report.pdf", "bytes": 3, "mimeType": "application/pdf", "date": "2026-03-13T12:00:00Z", "isUpload": false,
			}); err != nil {
				t.Fatalf("Encode() error: %v", err)
			}
		case "/download/report.pdf":
			_, _ = w.Write([]byte("pdf"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	outputDir := t.TempDir()
	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	result, err := client.DownloadAttachment(context.Background(), "c1", "a1", outputDir, false)
	if err != nil {
		t.Fatalf("DownloadAttachment() error: %v", err)
	}
	wantPath := filepath.Join(outputDir, "report.pdf")
	if result.Path != wantPath {
		t.Fatalf("result path = %q, want %q", result.Path, wantPath)
	}
	if data, err := os.ReadFile(wantPath); err != nil || string(data) != "pdf" {
		t.Fatalf("downloaded data = %q, err = %v", string(data), err)
	}
}

func TestDownloadAttachmentExternalURLDoesNotAppendAuth(t *testing.T) {
	var serverURL string
	var downloadQuery string
	var downloadAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1/cards/c1/attachments/a1":
			if err := json.NewEncoder(w).Encode(map[string]any{
				"id": "a1", "name": "external", "url": serverURL + "/external?existing=1", "bytes": 7, "mimeType": "text/plain", "date": "2026-03-13T12:00:00Z", "isUpload": false,
			}); err != nil {
				t.Fatalf("Encode() error: %v", err)
			}
		case "/external":
			downloadQuery = r.URL.RawQuery
			downloadAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte("content"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	if _, err := client.DownloadAttachment(context.Background(), "c1", "a1", filepath.Join(t.TempDir(), "external.txt"), false); err != nil {
		t.Fatalf("DownloadAttachment() error: %v", err)
	}
	if downloadQuery != "existing=1" {
		t.Fatalf("download query = %q, want existing=1", downloadQuery)
	}
	if downloadAuth != "" {
		t.Fatalf("download authorization = %q, want empty", downloadAuth)
	}
}

func TestDownloadAttachmentRefusesOverwrite(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/cards/c1/attachments/a1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"id": "a1", "name": "file.txt", "url": serverURL + "/download/file.txt", "bytes": 5, "mimeType": "text/plain", "date": "2026-03-13T12:00:00Z", "isUpload": true,
		}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "saved.txt")
	if err := os.WriteFile(outputPath, []byte("old"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	client := trello.NewClient(server.URL, "k", "t", trello.DefaultClientOptions())
	if _, err := client.DownloadAttachment(context.Background(), "c1", "a1", outputPath, false); err == nil {
		t.Fatal("DownloadAttachment() should reject existing file without force")
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("existing data = %q, want old", string(data))
	}
}
