package credential

import "context"

type KeySource interface {
	GetKey(ctx context.Context) ([]byte, error)
}
