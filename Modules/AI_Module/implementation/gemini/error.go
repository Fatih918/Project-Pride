package gemini

import (
	"errors"

	prideerrors "pride-ai-module/errors"

	"google.golang.org/genai"
)

func normalizeProviderError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr genai.APIError

	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 401:
			return prideerrors.New(
				prideerrors.CodeAuthenticationError,
				"Gemini provider authentication failed",
			).WithCause(err)

		case 403:
			return prideerrors.New(
				prideerrors.CodePermissionError,
				"Gemini provider permission denied",
			).WithCause(err)

		case 429:
			return prideerrors.New(
				prideerrors.CodeRateLimit,
				"Gemini provider rate limit exceeded",
			).WithCause(err)

		case 400:
			if isInvalidAPIKeyError(apiErr) {
				return prideerrors.New(
					prideerrors.CodeAuthenticationError,
					"Gemini provider authentication failed",
				).WithCause(err)
			}
		}
	}

	return prideerrors.New(
		prideerrors.CodeRuntimeError,
		"Gemini provider execution failed",
	).WithCause(err)
}

func isInvalidAPIKeyError(err genai.APIError) bool {
	if err.Status == "UNAUTHENTICATED" ||
		err.Status == "INVALID_API_KEY" {
		return true
	}

	for _, detail := range err.Details {
		if reason, ok := detail["reason"].(string); ok {
			switch reason {
			case "API_KEY_INVALID", "INVALID_API_KEY":
				return true
			}
		}

		if reason, ok := detail["Reason"].(string); ok {
			switch reason {
			case "API_KEY_INVALID", "INVALID_API_KEY":
				return true
			}
		}
	}

	return false
}
