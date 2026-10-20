package credential

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
)

type EnvironmentKeySource struct {
	VariableName string
}

func NewEnvironmentKeySource(variableName string) *EnvironmentKeySource {
	return &EnvironmentKeySource{
		VariableName: variableName,
	}
}

func (s *EnvironmentKeySource) GetKey(
	ctx context.Context,
) ([]byte, error) {
	value, ok := os.LookupEnv(s.VariableName)
	if !ok || value == "" {
		return nil, fmt.Errorf(
			"credential encryption key environment variable %q is not set",
			s.VariableName,
		)
	}

	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf(
			"decode credential encryption key: %w",
			err,
		)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf(
			"credential encryption key must be 32 bytes, got %d",
			len(key),
		)
	}

	return key, nil
}

var _ KeySource = (*EnvironmentKeySource)(nil)
