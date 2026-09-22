package credentials

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/zalando/go-keyring"
)

// ErrNotConfigured is returned when no credentials are stored for the requested profile.
var ErrNotConfigured = errors.New("credentials not configured")

// ErrReadOnly is returned when Set or Delete is called on a read-only store.
var ErrReadOnly = errors.New("credential store is read-only")

// Credentials holds the API key, token, and auth mode for a profile.
type Credentials struct {
	APIKey   string `json:"apiKey"`
	Token    string `json:"token"`
	AuthMode string `json:"authMode"`
}

// Store defines the interface for credential persistence.
type Store interface {
	Get(profile string) (Credentials, error)
	Set(profile string, creds Credentials) error
	Delete(profile string) error
}

// MemoryStore is an in-memory Store implementation, primarily for testing.
type MemoryStore struct {
	creds map[string]Credentials
}

// NewMemoryStore creates a new in-memory credential store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{creds: make(map[string]Credentials)}
}

func (m *MemoryStore) Get(profile string) (Credentials, error) {
	cred, ok := m.creds[profile]
	if !ok {
		return Credentials{}, ErrNotConfigured
	}
	return cred, nil
}

func (m *MemoryStore) Set(profile string, creds Credentials) error {
	m.creds[profile] = creds
	return nil
}

func (m *MemoryStore) Delete(profile string) error {
	delete(m.creds, profile)
	return nil
}

const keyringServicePrefix = "trello-cli"

// KeyringServiceName returns the keyring service name for a profile.
func KeyringServiceName(profile string) string {
	return keyringServicePrefix + "/" + profile
}

// KeyringStore persists credentials in the OS keyring.
type KeyringStore struct{}

// NewKeyringStore creates a new keyring-backed credential store.
func NewKeyringStore() *KeyringStore {
	return &KeyringStore{}
}

func (k *KeyringStore) Get(profile string) (Credentials, error) {
	svc := KeyringServiceName(profile)
	data, err := keyring.Get(svc, "credentials")
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return Credentials{}, ErrNotConfigured
		}
		return Credentials{}, errors.New("cannot read credentials from the OS keyring; check keyring access or set both TRELLO_API_KEY and TRELLO_TOKEN")
	}
	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return Credentials{}, errors.New("cannot decode credentials from the OS keyring; set both TRELLO_API_KEY and TRELLO_TOKEN to use environment credentials")
	}
	return creds, nil
}

func (k *KeyringStore) Set(profile string, creds Credentials) error {
	svc := KeyringServiceName(profile)
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return keyring.Set(svc, "credentials", string(data))
}

func (k *KeyringStore) Delete(profile string) error {
	svc := KeyringServiceName(profile)
	err := keyring.Delete(svc, "credentials")
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// EnvStore reads credentials from environment variables. It is read-only.
type EnvStore struct{}

// NewEnvStore creates a new environment-variable-backed credential store.
func NewEnvStore() *EnvStore {
	return &EnvStore{}
}

// Get reads credentials from TRELLO_API_KEY and TRELLO_TOKEN environment variables.
// The profile parameter is ignored — env-based auth has no profile concept.
func (e *EnvStore) Get(profile string) (Credentials, error) {
	apiKey := os.Getenv("TRELLO_API_KEY")
	token := os.Getenv("TRELLO_TOKEN")
	if apiKey == "" || token == "" {
		return Credentials{}, ErrNotConfigured
	}
	return Credentials{
		APIKey:   apiKey,
		Token:    token,
		AuthMode: "env",
	}, nil
}

func (e *EnvStore) Set(profile string, creds Credentials) error {
	return ErrReadOnly
}

func (e *EnvStore) Delete(profile string) error {
	return ErrReadOnly
}

// FallbackStore tries the secondary Store when the primary cannot provide credentials.
type FallbackStore struct {
	primary   Store
	secondary Store
}

// NewFallbackStore creates a new fallback chain with the provided primary and secondary stores.
func NewFallbackStore(primary, secondary Store) *FallbackStore {
	return &FallbackStore{
		primary:   primary,
		secondary: secondary,
	}
}

func (f *FallbackStore) Get(profile string) (Credentials, error) {
	creds, err := f.primary.Get(profile)
	if err == nil {
		return creds, nil
	}
	fallback, fallbackErr := f.secondary.Get(profile)
	if fallbackErr == nil {
		return fallback, nil
	}
	if !errors.Is(err, ErrNotConfigured) {
		return Credentials{}, err
	}
	return Credentials{}, fallbackErr
}

func (f *FallbackStore) Set(profile string, creds Credentials) error {
	return f.primary.Set(profile, creds)
}

func (f *FallbackStore) Delete(profile string) error {
	return f.primary.Delete(profile)
}
