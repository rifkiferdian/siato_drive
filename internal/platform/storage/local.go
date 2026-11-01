package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(root, "objects"), 0700); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(root, "temporary"), 0700); err != nil {
		return nil, err
	}
	return &Local{root: root}, nil
}
func (l *Local) safe(key string) (string, error) {
	if key == "" || filepath.IsAbs(key) || strings.Contains(key, "..") || strings.ContainsAny(key, "/\\") {
		return "", errors.New("invalid storage key")
	}
	p := filepath.Join(l.root, "objects", key)
	if filepath.Dir(p) != filepath.Join(l.root, "objects") {
		return "", errors.New("invalid storage key")
	}
	return p, nil
}
func (l *Local) Put(ctx context.Context, key string, src io.Reader) (int64, string, error) {
	dst, err := l.safe(key)
	if err != nil {
		return 0, "", err
	}
	tmp, err := os.CreateTemp(filepath.Join(l.root, "temporary"), "upload-*")
	if err != nil {
		return 0, "", err
	}
	name := tmp.Name()
	defer os.Remove(name)
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(tmp, h), &contextReader{ctx: ctx, r: src})
	closeErr := tmp.Close()
	if copyErr != nil {
		return n, "", copyErr
	}
	if closeErr != nil {
		return n, "", closeErr
	}
	if err = os.Rename(name, dst); err != nil {
		return n, "", err
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}
func (l *Local) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	p, err := l.safe(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}
func (l *Local) Delete(_ context.Context, key string) error {
	p, err := l.safe(key)
	if err != nil {
		return err
	}
	if err = os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *contextReader) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
		return c.r.Read(p)
	}
}
