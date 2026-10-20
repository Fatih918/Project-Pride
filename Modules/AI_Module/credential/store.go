package credential

import "context"

type Credential struct {
	Ref   string
	Value []byte
}

type Store interface {
	Get(ctx context.Context, ref string) (Credential, error)
	Set(ctx context.Context, credential Credential) error
	Delete(ctx context.Context, ref string) error
}
