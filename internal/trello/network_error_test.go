package trello_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Scale-Flow/trello-cli/internal/trello"
)

func TestNetworkErrorsDoNotExposeCredentials(t *testing.T) {
	file := filepath.Join(t.TempDir(), "attachment.txt")
	if err := os.WriteFile(file, []byte("test attachment"), 0600); err != nil {
		t.Fatal(err)
	}
	client := trello.NewClient("https://api.trello.com", "private-key", "private-token", trello.DefaultClientOptions())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, call := range map[string]func() error{
		"get":    func() error { return client.Get(ctx, "/1/members/me", nil, nil) },
		"upload": func() error { return client.PostMultipart(ctx, "/1/cards/test/attachments", nil, file, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			if err == nil {
				t.Fatal("expected network error")
			}
			for _, secret := range []string{"private-key", "private-token", "key=", "token="} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("network error exposes %q", secret)
				}
			}
		})
	}
}

func TestRedirectErrorDoesNotExposeCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
	}))
	defer server.Close()
	client := trello.NewClient(server.URL, "private-key", "private-token", trello.DefaultClientOptions())
	err := client.Get(context.Background(), "/1/members/me", nil, nil)
	if err == nil {
		t.Fatal("expected redirect failure")
	}
	for _, secret := range []string{"private-key", "private-token", "key=", "token="} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("redirect error exposes %q", secret)
		}
	}
}

func TestDownloadRedirectErrorDoesNotExposeCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/1/") {
			json.NewEncoder(w).Encode(trello.Attachment{
				ID: "attachment1", Name: "test.txt",
				URL: "http://" + r.Host + "/download?key=private-key&token=private-token",
			})
			return
		}
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
	}))
	defer server.Close()
	client := trello.NewClient(server.URL, "private-key", "private-token", trello.DefaultClientOptions())
	_, err := client.DownloadAttachment(context.Background(), "card1", "attachment1", filepath.Join(t.TempDir(), "test.txt"), false)
	if err == nil {
		t.Fatal("expected redirect failure")
	}
	for _, secret := range []string{"private-key", "private-token", "key=", "token="} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("download error exposes %q", secret)
		}
	}
}

func TestMalformedRequestURLDoesNotExposeCredentials(t *testing.T) {
	for _, baseURL := range []string{
		"http://invalid%host?key=private-key&token=private-token",
		"http://user:private-token@invalid%host",
	} {
		client := trello.NewClient(baseURL, "private-key", "private-token", trello.DefaultClientOptions())
		err := client.Get(context.Background(), "/1/members/me", nil, nil)
		if err == nil {
			t.Fatal("expected malformed URL error")
		}
		for _, secret := range []string{"private-key", "private-token", "key=", "token="} {
			if strings.Contains(err.Error(), secret) {
				t.Errorf("malformed URL error exposes %q", secret)
			}
		}
	}
}
