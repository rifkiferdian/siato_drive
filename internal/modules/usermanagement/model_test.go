package usermanagement

import "testing"

func TestUserStorageGB(t *testing.T) {
	user := User{StorageBytes: 1536 * 1024 * 1024}
	if got, want := user.StorageGB(), "1.500 GB"; got != want {
		t.Fatalf("StorageGB() = %q, want %q", got, want)
	}
}
