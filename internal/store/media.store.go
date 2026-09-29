package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/models"
	"github.com/google/uuid"
)

func (s *Store) CreateMedia(ctx context.Context, logger *slog.Logger, user_id uuid.UUID, innerKey string, publicKey string) (*models.Media, error) {
	return s.repo.CreateMedia(ctx, logger, user_id, innerKey, publicKey)
}

func (s *Store) CreateMediaBatch(ctx context.Context, logger *slog.Logger, mediaBatch *[]models.Media) *[]models.Media {
	return s.repo.CreateMediaBatch(ctx, logger, mediaBatch)
}

func (s *Store) HandleBatchCommits(ctx context.Context, logger *slog.Logger, batchId uuid.UUID) error {
	return s.repo.HandleBatchCommits(ctx, logger, batchId)
}

func (s *Store) RetriveBatch(ctx context.Context, logger *slog.Logger, batchId uuid.UUID) (*[]models.Media, error) {
	return s.repo.RetriveBatch(ctx, logger, batchId)
}

func (s *Store) GetMedia(ctx context.Context, logger *slog.Logger, publicKey string) (*models.Media, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	m, err := s.r.GetMedia(ctx, publicKey)
	if err == nil {
		return m, nil
	}
	m, err = s.repo.GetMedia(ctx, logger, publicKey)
	if err != nil {
		logger.Error("failed to retrieve media from db", "public_key", publicKey, "error", err.Error())
		return nil, err
	}
	err = s.r.SetMedia(ctx, *m)
	if err != nil {
		logger.Warn("failed to cache media", "public_key", publicKey, "error", err.Error())
	}
	return m, nil
}

func (s *Store) SetPresigned(ctx context.Context, logger *slog.Logger, publicKey string, url string, exp time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.r.SetPresignedUrl(ctx, publicKey, url, exp)
}

func (s *Store) GetPresigned(ctx context.Context, logger *slog.Logger, publicKey string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.r.GetPresignedUrl(ctx, publicKey)
}

func (s *Store) ToggleAsset(ctx context.Context, logger *slog.Logger, publicKey string, public bool) (*models.Media, error) {
	return s.repo.ToggleAssetVisibility(ctx, logger, publicKey, public)
}
