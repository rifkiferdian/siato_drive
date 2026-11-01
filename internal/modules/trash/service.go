package trash

import (
	"cloud-drive/internal/platform/storage"
	"context"
	"github.com/google/uuid"
)

type Service struct {
	repo    *Repository
	storage storage.Provider
}

func NewService(r *Repository, s storage.Provider) *Service { return &Service{repo: r, storage: s} }
func (s *Service) List(ctx context.Context, userID uuid.UUID, includePublic bool, q, kind string) ([]Item, error) {
	return s.repo.List(ctx, userID, includePublic, q, kind)
}
func (s *Service) Restore(ctx context.Context, userID, nodeID uuid.UUID, includePublic bool) error {
	return s.repo.Restore(ctx, userID, nodeID, includePublic)
}
func (s *Service) PermanentDelete(ctx context.Context, userID, nodeID uuid.UUID, includePublic bool) error {
	result, err := s.repo.PermanentDelete(ctx, userID, nodeID, includePublic)
	if err != nil {
		return err
	}
	for _, key := range result.StorageKeys {
		_ = s.storage.Delete(ctx, key)
	}
	return nil
}
