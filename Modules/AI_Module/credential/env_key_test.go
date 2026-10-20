package credential

import (
	"context"
	"encoding/base64"
	"testing"
)

func TestEnvironmentKeySource(t *testing.T) {
	ctx := context.Background()

	key := []byte("01234567890123456789012345678901")
	encoded := base64.StdEncoding.EncodeToString(key)

	t.Setenv("PRIDE_TEST_CREDENTIAL_KEY", encoded)

	source := NewEnvironmentKeySource(
		"PRIDE_TEST_CREDENTIAL_KEY",
	)

	result, err := source.GetKey(ctx)
	if err != nil {
		t.Fatalf("GetKey() failed: %v", err)
	}

	if string(result) != string(key) {
		t.Fatalf(
			"unexpected key: got %q, want %q",
			string(result),
			string(key),
		)
	}
}

func TestEnvironmentKeySourceRejectsMissingKey(t *testing.T) {
	ctx := context.Background()

	source := NewEnvironmentKeySource(
		"PRIDE_MISSING_CREDENTIAL_KEY",
	)

	_, err := source.GetKey(ctx)
	if err == nil {
		t.Fatal("expected missing key to return an error")
	}
}

func TestEnvironmentKeySourceRejectsInvalidLength(t *testing.T) {
	ctx := context.Background()

	invalidKey := []byte("short-key")
	encoded := base64.StdEncoding.EncodeToString(invalidKey)

	t.Setenv("PRIDE_INVALID_CREDENTIAL_KEY", encoded)

	source := NewEnvironmentKeySource(
		"PRIDE_INVALID_CREDENTIAL_KEY",
	)

	_, err := source.GetKey(ctx)
	if err == nil {
		t.Fatal("expected invalid key length to return an error")
	}
}
