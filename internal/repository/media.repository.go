package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) CreateMedia(ctx context.Context, logger *slog.Logger, user_id uuid.UUID, innerKey string, publicKey string) (*models.Media, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	INSERT INTO medias (public_key, inner_key,user_id)
	VALUES ($1, $2, $3)
	RETURNING public_key, inner_key, user_id, file_size, status, file_type, active, file_name, created_at, updated_at, batch_id,seq_id,public,mime_type,hits,
	`
	rows, err := r.pool.Query(ctx, query, publicKey, innerKey, user_id)
	if err != nil {
		logger.Error("failed to create media", "public_key", publicKey, "error", err.Error())
		return nil, err
	}
	media, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Media])
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (r *Repository) CreateMediaBatch(ctx context.Context, logger *slog.Logger, mediaBatch *[]models.Media) *[]models.Media {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var allSavedMedia []models.Media
	batch := pgx.Batch{}

	for _, f := range *mediaBatch {
		batch.Queue(
			`
				INSERT INTO medias (public_key, inner_key, user_id, file_name, mime_type, batch_id, seq_id, public)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				RETURNING public_key, inner_key, user_id, file_size, status, file_type, active, file_name, created_at, updated_at, batch_id,seq_id, public, mime_type, hits
				`, f.PublicKey, f.InnerKey, f.UserId, f.FileName, f.MimeType, f.BatchId, f.SeqId, f.Public,
		)
	}
	results := r.pool.SendBatch(ctx, &batch)
	defer results.Close()

	for range *mediaBatch {
		rows, err := results.Query()

		if err != nil {
			logger.Error("error querying media batch", "error", err.Error())
			continue
		}
		m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Media])
		if err != nil {
			logger.Error("error collecting media row from batch", "error", err.Error())
			continue
		}
		allSavedMedia = append(allSavedMedia, m)
	}
	return &allSavedMedia
}

func (r *Repository) HandleBatchCommits(ctx context.Context, logger *slog.Logger, batchId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
	UPDATE medias
	SET status = 'completed'
	WHERE batch_id = $1
	`
	tag, err := r.pool.Exec(ctx, query, batchId)
	if err != nil {
		logger.Error("failed to commit batch", "batch_id", batchId.String(), "error", err.Error())
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) RetriveBatch(ctx context.Context, logger *slog.Logger, batchId uuid.UUID) (*[]models.Media, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
	SELECT *
	FROM medias
	WHERE batch_id = $1 and status = 'pending'
	`
	rows, err := r.pool.Query(ctx, query, batchId)
	if err != nil {
		logger.Error("failed to retrieve batch", "batch_id", batchId.String(), "error", err.Error())
		return nil, err
	}
	medias, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Media])
	if err != nil {
		return nil, err
	}
	return &medias, nil
}

func (r *Repository) GetMedia(ctx context.Context, logger *slog.Logger, publicKey string) (*models.Media, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
	SELECT *
	FROM medias
	WHERE public_key = $1 AND status = 'completed'
	`
	rows, err := r.pool.Query(ctx, query, publicKey)
	if err != nil {
		logger.Error("failed to get media", "public_key", publicKey, "error", err.Error())
		return nil, err
	}
	media, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Media])
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (r *Repository) ToggleAssetVisibility(ctx context.Context, logger *slog.Logger, publicKey string, public bool) (*models.Media, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
	UPDATE medias
	SET public = $1
	WHERE public_key = $2
	RETURNING public_key, inner_key, user_id, file_size, status, file_type, active, file_name, created_at, updated_at, batch_id,seq_id, public, mime_type, hits
	`
	rows, err := r.pool.Query(ctx, query, public, publicKey)
	if err != nil {
		logger.Error("failed to toggle asset visibility", "public_key", publicKey, "public", public, "error", err.Error())
		return nil, err
	}
	media, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Media])
	if err != nil {
		return nil, err
	}
	logger.Info("asset visibility toggled in repository", "public_key", publicKey, "public", public)
	return &media, nil
}
