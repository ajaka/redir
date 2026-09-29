package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/models"
	"github.com/google/uuid"
)

func (s *Store) CreateUser(ctx context.Context, logger *slog.Logger, u *domain.
	CreateUserDetails, cfg *configs.EnvData) error {
	return s.repo.CreateUser(ctx, logger, u, cfg)
}

func (s *Store) CreateOrLinkOauth(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, id_or_sub string, email string, name string, provider string) (*domain.LightUser, error) {
	return s.repo.CreateOrLinkOauth(ctx, logger, cfg, id_or_sub, email, name, provider)
}

func (s *Store) GetUserByEmail(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, email string) (*models.User, error) {
	const by string = "email"
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	u, err := s.r.GetFullUser(ctx, email, by)
	if err == nil {
		return u, nil
	}
	u, err = s.repo.GetUserByEmail(ctx, logger, cfg, email)
	if err != nil {
		return nil, err
	}
	s.r.SetFullUser(ctx, u.Email, by, *u)
	return u, nil
}

func (s *Store) GetUserById(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, id uuid.UUID) (*models.User, error) {
	const by string = "id"
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	u, err := s.r.GetFullUser(ctx, id.String(), by)
	if err == nil {
		return u, nil
	}
	u, err = s.repo.GetUserById(ctx, logger, cfg, id)
	if err != nil {
		return nil, err
	}
	s.r.SetFullUser(ctx, u.Id.String(), by, *u)
	return u, nil
}

func (s *Store) GetUserByProvider(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, provider string, sub string) (*domain.LightUser, error) {
	return s.repo.GetUserByProvider(ctx, logger, cfg, provider, sub)
}
