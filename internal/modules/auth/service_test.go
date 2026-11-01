package auth

import "testing"

func TestPasswordHash(t *testing.T) {
	hash, err := HashPassword("Correct horse battery staple 1")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword("Correct horse battery staple 1", hash) {
		t.Fatal("valid password rejected")
	}
	if verifyPassword("wrong password", hash) {
		t.Fatal("invalid password accepted")
	}
}
func TestPasswordMinimum(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short password accepted")
	}
}

func TestPasswordComplexity(t *testing.T) {
	for _, password := range []string{"abcdef1", "Abcdef", "Ab1"} {
		if _, err := HashPassword(password); err == nil {
			t.Fatalf("password %q should be rejected", password)
		}
	}
	if _, err := HashPassword("Abcd12"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
}
