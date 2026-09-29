package repository

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/errs"
	"github.com/ajaka/redir/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) CreateUser(ctx context.Context, logger *slog.Logger, user *domain.
	CreateUserDetails, cfg *configs.EnvData) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	INSERT INTO users (full_name,email,password)
	VALUES ($1, $2, $3)
	`
	_, err := r.pool.Exec(ctx, query, user.FullName, user.Email, user.Password)
	if err != nil {
		logger.Error("failed to create user", "email", user.Email, "error", err.Error())
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errs.ErrDuplicateEmail
		}
		return err
	}
	logger.Info("user created successfully", "email", user.Email)
	return nil
}

func (r *Repository) CreateOrLinkOauth(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, id_or_sub string, email string, name string, provider string) (*domain.LightUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	INSERT INTO users (provider_sub, email, full_name, provider,verified)
	VALUES ($1, $2, $3, $4, true)
	ON CONFLICT (email) DO UPDATE SET
	verified = true,
	provider_sub = EXCLUDED.provider_sub,
	provider = EXCLUDED.provider
	RETURNING id, email, admin, paid
	`
	rows, err := r.pool.Query(ctx, query, id_or_sub, email, name, provider)
	if err != nil {
		logger.Error("failed to create or link oauth user", "provider", provider, "email", email, "error", err.Error())
		return nil, err
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.LightUser])
	if err != nil {
		return nil, err
	}
	logger.Info("oauth user created or linked", "provider", provider, "email", email)
	return &user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	SELECT id, email, password, verified
	FROM users
	WHERE email = $1
	`
	rows, err := r.pool.Query(ctx, query, email)
	if err != nil {
		logger.Error("failed to get user by email", "email", email, "error", err.Error())
		return nil, err
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[models.User])

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserById(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, id uuid.UUID) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	SELECT id, email, verified, full_name, paid
	FROM users
	WHERE id = $1
	`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		logger.Error("failed to get user by id", "user_id", id.String(), "error", err.Error())
		return nil, err
	}

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[models.User])

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserByProvider(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, provider string, sub string) (*domain.LightUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	SELECT id, email, paid, admin
	FROM users
	WHERE provider = $1 AND provider_sub = $2
	`
	rows, err := r.pool.Query(ctx, query, provider, sub)
	if err != nil {
		logger.Error("failed to get user by provider", "provider", provider, "error", err.Error())
		return nil, err
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.LightUser])

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) SetUserVerified(ctx context.Context, logger *slog.Logger, email string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	UPDATE users
	SET verified = true
	WHERE email = $1 AND verified = false
	`
	tag, err := r.pool.Exec(ctx, query, email)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
