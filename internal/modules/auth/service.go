package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrInvalidCredentials = errors.New("email atau kata sandi salah")
var ErrCurrentPassword = errors.New("password saat ini tidak sesuai")
var ErrProfileInput = errors.New("nama tampilan tidak valid")

type Service struct {
	repo *Repository
	ttl  time.Duration
}

func NewService(repo *Repository, ttl time.Duration) *Service { return &Service{repo: repo, ttl: ttl} }
func (s *Service) Login(ctx context.Context, email, password, ip, ua string) (string, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	u, hash, err := s.repo.Credentials(ctx, email)
	if err != nil || !u.Active || !verifyPassword(password, hash) {
		s.repo.LogLogin(ctx, email, nil, false, ip)
		return "", "", ErrInvalidCredentials
	}
	token, err := randomToken(32)
	if err != nil {
		return "", "", err
	}
	csrf, err := randomToken(24)
	if err != nil {
		return "", "", err
	}
	if err = s.repo.CreateSession(ctx, u.ID, token, csrf, ip, ua, time.Now().Add(s.ttl)); err != nil {
		return "", "", err
	}
	s.repo.LogLogin(ctx, email, &u.ID, true, ip)
	return token, csrf, nil
}
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	if token == "" {
		return Session{}, ErrInvalidCredentials
	}
	return s.repo.Session(ctx, token)
}
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.repo.DeleteSession(ctx, token)
}
func (s *Service) UpdateDisplayName(ctx context.Context, userID uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 150 {
		return ErrProfileInput
	}
	return s.repo.UpdateDisplayName(ctx, userID, name)
}
func (s *Service) ChangePassword(ctx context.Context, session Session, current, next string) error {
	_, hash, err := s.repo.Credentials(ctx, session.User.Email)
	if err != nil || !verifyPassword(current, hash) {
		return ErrCurrentPassword
	}
	nextHash, err := HashPassword(next)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, session.UserID, session.ID, nextHash)
}
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func HashPassword(p string) (string, error) {
	var hasUpper, hasDigit bool
	for _, char := range p {
		hasUpper = hasUpper || unicode.IsUpper(char)
		hasDigit = hasDigit || unicode.IsDigit(char)
	}
	if utf8.RuneCountInString(p) < 6 || !hasUpper || !hasDigit {
		return "", errors.New("password minimal 6 karakter serta wajib memiliki huruf besar dan angka")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	h := argon2.IDKey([]byte(p), salt, 3, 64*1024, 2, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(h)), nil
}
func verifyPassword(p, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var m, t, pn uint64
	for _, item := range strings.Split(parts[3], ",") {
		kv := strings.SplitN(item, "=", 2)
		if len(kv) != 2 {
			return false
		}
		n, err := strconv.ParseUint(kv[1], 10, 32)
		if err != nil {
			return false
		}
		switch kv[0] {
		case "m":
			m = n
		case "t":
			t = n
		case "p":
			pn = n
		}
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(p), salt, uint32(t), uint32(m), uint8(pn), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}
