package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestLocalRoundTrip(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	n, sum, err := s.Put(context.Background(), "safe-key", bytes.NewBufferString("hello"))
	if err != nil || n != 5 || len(sum) != 64 {
		t.Fatalf("put: n=%d sum=%q err=%v", n, sum, err)
	}
	f, err := s.Open(context.Background(), "safe-key")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, _ := io.ReadAll(f)
	if string(b) != "hello" {
		t.Fatalf("got %q", b)
	}
}
func TestLocalRejectsTraversal(t *testing.T) {
	s, _ := NewLocal(t.TempDir())
	if _, _, err := s.Put(context.Background(), "../escape", bytes.NewReader(nil)); err == nil {
		t.Fatal("traversal accepted")
	}
}
