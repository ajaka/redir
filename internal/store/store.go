package store

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ajaka/redir/internal/cache"
	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/repository"
)

type Store struct {
	r    *cache.Sredis
	repo *repository.Repository
	cfg  *configs.EnvData
}

func InitializeStore(r *cache.Sredis, repo *repository.Repository, cfg *configs.EnvData) *Store {
	return &Store{
		r:    r,
		repo: repo,
		cfg:  cfg,
	}
}

func (s *Store) CheckDependencies(ctx context.Context, logger *slog.Logger) error {
	logger.Info("Checking dependencies availability")

	if s == nil {
		return fmt.Errorf("store: dependency checker is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	errs := make([]string, 0, 3)
	if err := s.r.CheckHealth(ctx, logger); err != nil {
		errs = append(errs, err.Error())
	}
	if err := s.repo.CheckHealth(ctx, logger); err != nil {
		errs = append(errs, err.Error())
	}
	if err := configs.CheckStorageHealth(ctx, s.cfg, logger); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("dependency checks failed: %s", strings.Join(errs, "; "))
	}

	return nil
}
