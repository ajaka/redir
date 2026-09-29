package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/models"
)

type AuthStore interface {
	CreateUser(ctx context.Context, logger *slog.Logger, user *domain.CreateUserDetails, cfg *configs.EnvData) error
	GetUserByEmail(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, email string) (*models.User, error)
	SetUserOnline(ctx context.Context, logger *slog.Logger, sessionId string, u *domain.LightUser) (time.Time, error)
	RevokeUser(ctx context.Context, logger *slog.Logger, sessionId string) error
	GetUserByProvider(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, provider string, sub string) (*domain.LightUser, error)
	CreateOrLinkOauth(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, id_or_sub string, email string, name string, provider string) (*domain.LightUser, error)
	GetUser(ctx context.Context, logger *slog.Logger, sessionId string) (*domain.LightUser, bool)
	SetVerificationUser(ctx context.Context, logger *slog.Logger, email string, token string) error
	GetVerificationUser(ctx context.Context, logger *slog.Logger, token string) (string, error)
	SetUserVerified(ctx context.Context, logger *slog.Logger, email string) error
}
