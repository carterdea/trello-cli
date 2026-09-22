package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestAuthStatusWithUnavailableKeyring(t *testing.T) {
	setupTestAuth(t)
	credStore = nil
	keyring.MockInitWithError(errors.New("exit status 51"))
	t.Cleanup(keyring.MockInit)
	t.Setenv("TRELLO_API_KEY", "env-key")
	t.Setenv("TRELLO_TOKEN", "env-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "env-key" || r.URL.Query().Get("token") != "env-token" {
			t.Error("request did not use complete environment credentials")
		}
		json.NewEncoder(w).Encode(map[string]string{"id": "member1", "username": "test-user"})
	}))
	t.Cleanup(server.Close)
	oldURL := trelloBaseURL
	trelloBaseURL = server.URL
	t.Cleanup(func() { trelloBaseURL = oldURL })
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"auth", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("auth status failed with valid environment credentials: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Configured bool   `json:"configured"`
			AuthMode   string `json:"authMode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || !response.Data.Configured || response.Data.AuthMode != "env" {
		t.Fatalf("unexpected status: %s", out.String())
	}
}

func TestAuthStatusKeyringErrorIsActionable(t *testing.T) {
	setupTestAuth(t)
	credStore = nil
	keyring.MockInitWithError(errors.New("exit status 51"))
	t.Cleanup(keyring.MockInit)
	t.Setenv("TRELLO_API_KEY", "")
	t.Setenv("TRELLO_TOKEN", "")
	rootCmd.SetArgs([]string{"auth", "status"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected keyring access error")
	}
	var out bytes.Buffer
	handleError(&out, err)
	for _, hint := range []string{"keyring", "TRELLO_API_KEY", "TRELLO_TOKEN"} {
		if !strings.Contains(out.String(), hint) {
			t.Errorf("error does not include %q: %s", hint, out.String())
		}
	}
}
