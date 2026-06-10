package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/Scale-Flow/trello-cli/internal/credentials"
	"github.com/Scale-Flow/trello-cli/internal/trello"
)

func TestAttachmentsListCommand(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	apiClient = &mockAPI{
		listAttachmentsFn: func(ctx context.Context, cardID string) ([]trello.Attachment, error) {
			if cardID != "c1" {
				t.Errorf("card ID = %q, want c1", cardID)
			}
			return []trello.Attachment{{ID: "a1", Name: "file.txt"}}, nil
		},
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"attachments", "list", "--card", "c1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("attachments list failed: %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	data := envelope["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(data))
	}
}

func TestAttachmentsAddFileCommand(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	file, err := os.CreateTemp(t.TempDir(), "attachment-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp() error: %v", err)
	}
	if _, err := file.WriteString("hello"); err != nil {
		t.Fatalf("WriteString() error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	name := "renamed.txt"
	apiClient = &mockAPI{
		addFileAttachmentFn: func(ctx context.Context, cardID, filePath string, gotName *string) (trello.Attachment, error) {
			if cardID != "c1" || filePath != file.Name() {
				t.Fatalf("card/path = %q/%q", cardID, filePath)
			}
			if gotName == nil || *gotName != name {
				t.Fatalf("name = %v", gotName)
			}
			return trello.Attachment{ID: "a1", Name: name}, nil
		},
	}

	if err := executeRootArgs("attachments", "add-file", "--card", "c1", "--path", file.Name(), "--name", name); err != nil {
		t.Fatalf("attachments add-file failed: %v", err)
	}
}

func TestAttachmentsAddFileMissingPath(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	assertContractCode(t, executeRootArgs("attachments", "add-file", "--card", "c1", "--path", "/nope"), "FILE_NOT_FOUND")
}

func TestAttachmentsAddURLCommand(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	name := "Reference"
	apiClient = &mockAPI{
		addURLAttachmentFn: func(ctx context.Context, cardID, urlStr string, gotName *string) (trello.Attachment, error) {
			if cardID != "c1" || urlStr != "https://example.com" {
				t.Fatalf("card/url = %q/%q", cardID, urlStr)
			}
			if gotName == nil || *gotName != name {
				t.Fatalf("name = %v", gotName)
			}
			return trello.Attachment{ID: "a1", Name: name}, nil
		},
	}

	if err := executeRootArgs("attachments", "add-url", "--card", "c1", "--url", "https://example.com", "--name", name); err != nil {
		t.Fatalf("attachments add-url failed: %v", err)
	}
}

func TestAttachmentsAddURLInvalidURL(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	assertContractCode(t, executeRootArgs("attachments", "add-url", "--card", "c1", "--url", "notaurl"), "VALIDATION_ERROR")
}

func TestAttachmentsDeleteCommand(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	apiClient = &mockAPI{
		deleteAttachmentFn: func(ctx context.Context, cardID, attachmentID string) error {
			if cardID != "c1" || attachmentID != "a1" {
				t.Fatalf("card/attachment = %q/%q", cardID, attachmentID)
			}
			return nil
		},
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"attachments", "delete", "--card", "c1", "--attachment", "a1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("attachments delete failed: %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	data := envelope["data"].(map[string]any)
	if data["deleted"] != true || data["id"] != "a1" {
		t.Fatalf("data = %+v", data)
	}
}

func TestAttachmentsDownloadCommand(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	outputPath := "/tmp/download.txt"
	apiClient = &mockAPI{
		downloadAttachmentFn: func(ctx context.Context, cardID, attachmentID, gotOutputPath string, force bool) (trello.AttachmentDownloadResult, error) {
			if cardID != "c1" || attachmentID != "a1" || gotOutputPath != outputPath || !force {
				t.Fatalf("download args = %q/%q/%q/%v", cardID, attachmentID, gotOutputPath, force)
			}
			return trello.AttachmentDownloadResult{
				ID:       "a1",
				CardID:   "c1",
				Name:     "file.txt",
				Path:     outputPath,
				Bytes:    5,
				MimeType: "text/plain",
			}, nil
		},
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"attachments", "download", "--card", "c1", "--attachment", "a1", "--output", outputPath, "--force"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("attachments download failed: %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	data := envelope["data"].(map[string]any)
	if data["id"] != "a1" || data["path"] != outputPath || data["bytes"] != float64(5) {
		t.Fatalf("data = %+v", data)
	}
}

func TestAttachmentsDownloadMissingOutput(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	assertContractCode(t, executeRootArgs("attachments", "download", "--card", "c1", "--attachment", "a1"), "VALIDATION_ERROR")
}

func TestAttachmentsDownloadMissingAttachment(t *testing.T) {
	setupTestAuth(t)
	credStore.Set("default", credentials.Credentials{APIKey: "k", Token: "t", AuthMode: "manual"})
	assertContractCode(t, executeRootArgs("attachments", "download", "--card", "c1", "--output", "/tmp/download.txt"), "VALIDATION_ERROR")
}
