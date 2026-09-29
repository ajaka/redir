package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/models"
)

func (s *Store) CreatePrivateKey(ctx context.Context, logger *slog.Logger, productId int, hash string) (*models.Product, error) {
	return s.repo.CreatePrivateKey(ctx, logger, productId, hash)
}

func (s *Store) GetProductById(ctx context.Context, logger *slog.Logger, productId int) (*models.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p, err := s.r.GetProduct(ctx, productId)
	if err == nil {
		return p, nil
	}
	p, err = s.repo.GetProductById(ctx, logger, productId)
	if err != nil {
		return nil, err
	}
	s.r.SetProduct(ctx, *p)
	return p, nil
}

func (s *Store) CreateProduct(ctx context.Context, logger *slog.Logger, data *domain.CreateProductDetails) (*models.Product, error) {
	return s.repo.CreateProduct(ctx, logger, data)
}

func (s *Store) ToggleProductVisibility(ctx context.Context, logger *slog.Logger, public bool, productId int) (*models.Product, error) {
	return s.repo.ToggleProductVisibility(ctx, logger, productId, public)
}
