package store

import (
	"context"
	"log/slog"

	"github.com/ajaka/redir/internal/models"
)

func (s *Store) SaveMetrics(ctx context.Context, logger *slog.Logger, metric *models.Metrics) error {
	return s.repo.SaveMetrics(ctx, logger, metric)
}
