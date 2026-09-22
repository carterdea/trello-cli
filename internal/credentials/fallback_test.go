package credentials_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Scale-Flow/trello-cli/internal/credentials"
	"github.com/zalando/go-keyring"
)

type unavailableStore struct {
	credentials.Store
	err error
}

func (s unavailableStore) Get(string) (credentials.Credentials, error) {
	return credentials.Credentials{}, s.err
}

func TestFallbackStoreKeyringUnavailable(t *testing.T) {
	keyringErr := errors.New("keyring access denied: exit status 51")
	for _, tc := range []struct {
		name, key, token string
		configured       bool
	}{
		{"complete environment", "env-key", "env-token", true},
		{"key only", "env-key", "", false},
		{"token only", "", "env-token", false},
		{"empty environment", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRELLO_API_KEY", tc.key)
			t.Setenv("TRELLO_TOKEN", tc.token)
			store := credentials.NewFallbackStore(unavailableStore{err: keyringErr}, credentials.NewEnvStore())
			got, err := store.Get("default")
			if !tc.configured {
				if !errors.Is(err, keyringErr) || errors.Is(err, credentials.ErrNotConfigured) {
					t.Fatalf("Get() error = %v, want original keyring error", err)
				}
				if got != (credentials.Credentials{}) {
					t.Fatal("Get() returned partial credentials")
				}
				return
			}
			want := credentials.Credentials{APIKey: tc.key, Token: tc.token, AuthMode: "env"}
			if err != nil || got != want {
				t.Fatalf("Get() = %+v, %v; want complete environment credentials", got, err)
			}
		})
	}
}

func TestFallbackStorePreservesStoredCredentials(t *testing.T) {
	for _, mode := range []string{"manual", "interactive", "device", "key_only"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("TRELLO_API_KEY", "env-key")
			t.Setenv("TRELLO_TOKEN", "env-token")
			primary := credentials.NewMemoryStore()
			want := credentials.Credentials{APIKey: "stored-key", AuthMode: mode}
			if mode != "key_only" {
				want.Token = "stored-token"
			}
			if err := primary.Set("default", want); err != nil {
				t.Fatal(err)
			}
			got, err := credentials.NewFallbackStore(primary, credentials.NewEnvStore()).Get("default")
			if err != nil || got != want {
				t.Fatalf("Get() = %+v, %v; want stored credentials", got, err)
			}
		})
	}
}

func TestKeyringStoreReadErrorsAreSafe(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		t.Run(map[bool]string{false: "access denied", true: "malformed data"}[malformed], func(t *testing.T) {
			keyring.MockInit()
			t.Cleanup(keyring.MockInit)
			if malformed {
				if err := keyring.Set(credentials.KeyringServiceName("default"), "credentials", "private-token"); err != nil {
					t.Fatal(err)
				}
			} else {
				keyring.MockInitWithError(errors.New("backend error containing private-token"))
			}
			_, err := credentials.NewKeyringStore().Get("default")
			if err == nil || errors.Is(err, credentials.ErrNotConfigured) {
				t.Fatalf("Get() error = %v, want keyring read error", err)
			}
			if strings.Contains(err.Error(), "private-token") {
				t.Error("keyring error exposed secret data")
			}
			for _, hint := range []string{"OS keyring", "TRELLO_API_KEY", "TRELLO_TOKEN"} {
				if !strings.Contains(err.Error(), hint) {
					t.Errorf("keyring error missing %q", hint)
				}
			}
		})
	}
}
