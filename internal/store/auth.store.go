package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/domain"
)

func (s *Store) SetUserOnline(ctx context.Context, logger *slog.Logger, sessionId string, u *domain.
	LightUser) (time.Time, error) {
	return s.r.SetUserOnline(ctx, sessionId, u)
}

func (s *Store) GetUser(ctx context.Context, logger *slog.Logger, sessionId string) (*domain.
	LightUser, bool) {
	return s.r.GetUser(ctx, sessionId)
}

func (s *Store) RevokeUser(ctx context.Context, logger *slog.Logger, sessionId string) error {
	return s.r.RevokeUser(ctx, sessionId)
}
func (s *Store) SetVerificationUser(ctx context.Context, logger *slog.Logger, email string, token string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	return s.r.SetVerifcationUser(ctx, email, token)
}

func (s *Store) GetVerificationUser(ctx context.Context, logger *slog.Logger, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	return s.r.GetVerifcationUser(ctx, token)
}

func (s *Store) SetUserVerified(ctx context.Context, logger *slog.Logger, email string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	if err := s.repo.SetUserVerified(ctx, logger, email); err != nil {
		return err
	}
	_ = s.r.RevokeFullUser(ctx, email, "email")
	return nil
}
