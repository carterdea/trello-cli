package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Scale-Flow/trello-cli/internal/auth"
	"github.com/Scale-Flow/trello-cli/internal/contract"
	"github.com/Scale-Flow/trello-cli/internal/credentials"
)

func TestStatusNetworkErrorDoesNotExposeCredentials(t *testing.T) {
	store := credentials.NewMemoryStore()
	if err := store.Set("default", credentials.Credentials{APIKey: "private-key", Token: "private-token", AuthMode: "manual"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := auth.Status(ctx, store, "default", "https://api.trello.com")
	var ce *contract.ContractError
	if !errors.As(err, &ce) || ce.Code != contract.HTTPError {
		t.Fatalf("Status() error = %v, want HTTP_ERROR", err)
	}
	for _, secret := range []string{"private-key", "private-token", "key=", "token="} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("network error exposes %q", secret)
		}
	}
}
