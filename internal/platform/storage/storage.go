package storage

import (
	"context"
	"io"
)

type Provider interface {
	Put(ctx context.Context, key string, src io.Reader) (int64, string, error)
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}
