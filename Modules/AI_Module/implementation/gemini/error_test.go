package gemini

import (
	"errors"
	"testing"

	prideerrors "pride-ai-module/errors"

	"google.golang.org/genai"
)

func TestNormalizeProviderErrorAuthentication401(t *testing.T) {
	original := genai.APIError{
		Code:    401,
		Status:  "UNAUTHENTICATED",
		Message: "authentication failed",
	}

	err := normalizeProviderError(original)

	if err == nil {
		t.Fatal("expected error")
	}

	var prideErr *prideerrors.Error

	if !errors.As(err, &prideErr) {
		t.Fatalf(
			"expected PRIDE error, got %T",
			err,
		)
	}

	if prideErr.Code != prideerrors.CodeAuthenticationError {
		t.Fatalf(
			"unexpected code: got %q, want %q",
			prideErr.Code,
			prideerrors.CodeAuthenticationError,
		)
	}

	if prideErr.Message !=
		"Gemini provider authentication failed" {
		t.Fatalf(
			"unexpected message: got %q",
			prideErr.Message,
		)
	}

	if prideErr.Cause == nil {
		t.Fatal("expected original provider error as cause")
	}
}

func TestNormalizeProviderErrorInvalidAPIKey400(t *testing.T) {
	original := genai.APIError{
		Code:    400,
		Status:  "INVALID_ARGUMENT",
		Message: "API key is invalid",
		Details: []map[string]any{
			{
				"reason": "API_KEY_INVALID",
			},
		},
	}

	err := normalizeProviderError(original)

	if err == nil {
		t.Fatal("expected error")
	}

	var prideErr *prideerrors.Error

	if !errors.As(err, &prideErr) {
		t.Fatalf(
			"expected PRIDE error, got %T",
			err,
		)
	}
	if prideErr.Code != prideerrors.CodeAuthenticationError {
		t.Fatalf(
			"unexpected code: got %q, want %q",
			prideErr.Code,
			prideerrors.CodeAuthenticationError,
		)
	}

}

func TestNormalizeProviderErrorPermission(t *testing.T) {
	original := genai.APIError{
		Code:    403,
		Status:  "PERMISSION_DENIED",
		Message: "permission denied",
	}

	err := normalizeProviderError(original)

	var prideErr *prideerrors.Error

	if !errors.As(err, &prideErr) {
		t.Fatalf(
			"expected PRIDE error, got %T",
			err,
		)
	}

	if prideErr.Code != prideerrors.CodePermissionError {
		t.Fatalf(
			"unexpected code: got %q, want %q",
			prideErr.Code,
			prideerrors.CodePermissionError,
		)
	}
}

func TestNormalizeProviderErrorRuntime(t *testing.T) {
	original := genai.APIError{
		Code:    500,
		Status:  "INTERNAL",
		Message: "internal server error",
	}

	err := normalizeProviderError(original)

	var prideErr *prideerrors.Error

	if !errors.As(err, &prideErr) {
		t.Fatalf(
			"expected PRIDE error, got %T",
			err,
		)
	}

	if prideErr.Code != prideerrors.CodeRuntimeError {
		t.Fatalf(
			"unexpected code: got %q, want %q",
			prideErr.Code,
			prideerrors.CodeRuntimeError,
		)
	}
}

func TestNormalizeProviderErrorDoesNotExposeProviderMessage(t *testing.T) {
	original := genai.APIError{
		Code:    401,
		Status:  "UNAUTHENTICATED",
		Message: "api-key=SUPER_SECRET",
	}

	err := normalizeProviderError(original)

	var prideErr *prideerrors.Error

	if !errors.As(err, &prideErr) {
		t.Fatalf(
			"expected PRIDE error, got %T",
			err,
		)
	}

	if prideErr.Message == original.Message {
		t.Fatal(
			"PRIDE error message must not expose provider message",
		)
	}

	if prideErr.Message !=
		"Gemini provider authentication failed" {
		t.Fatalf(
			"unexpected safe message: got %q",
			prideErr.Message,
		)
	}

	if prideErr.Cause == nil {
		t.Fatal(
			"expected original error to remain as internal cause",
		)
	}
}
