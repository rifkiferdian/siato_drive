package usermanagement

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"cloud-drive/internal/modules/auth"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("data pengguna tidak valid")
var ErrSelfAction = errors.New("akun yang sedang digunakan tidak dapat dinonaktifkan")

type Service struct {
	repo    *Repository
	orgSlug string
}

func NewService(repo *Repository, orgSlug string) *Service {
	return &Service{repo: repo, orgSlug: orgSlug}
}
func (s *Service) List(ctx context.Context) ([]User, error) { return s.repo.List(ctx) }
func (s *Service) Create(ctx context.Context, in CreateInput) error {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Role = strings.ToLower(strings.TrimSpace(in.Role))
	address, parseErr := mail.ParseAddress(in.Email)
	if parseErr != nil || address.Address != in.Email || in.DisplayName == "" || len(in.DisplayName) > 150 || !validRoles[in.Role] {
		return ErrInvalidInput
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return err
	}
	return s.repo.Create(ctx, in, hash, s.orgSlug)
}
func (s *Service) SetActive(ctx context.Context, actor, id uuid.UUID, active bool) error {
	if actor == id && !active {
		return ErrSelfAction
	}
	return s.repo.SetActive(ctx, id, active)
}
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.repo.ResetPassword(ctx, id, hash)
}
func (s *Service) SetRole(ctx context.Context, actor, id uuid.UUID, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if !validRoles[role] {
		return ErrInvalidInput
	}
	if actor == id && role != "admin" {
		return ErrSelfAction
	}
	return s.repo.SetRole(ctx, id, role, s.orgSlug)
}
