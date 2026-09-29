package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/models"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) CreatePrivateKey(ctx context.Context, logger *slog.Logger, productId int, hash string) (*models.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	update products
	SET private_key = $2,updated_at = CURRENT_TIMESTAMP
	WHERE product_id = $1
	RETURNING id, product_id, product_name,user_id, created_at, updated_at, public, private_key
	`
	rows, err := r.pool.Query(ctx, query, productId, hash)
	if err != nil {
		logger.Error("failed to create private key", "product_id", productId, "error", err.Error())
		return nil, err
	}
	product, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Product])
	if err != nil {
		return nil, err
	}
	logger.Info("private key created", "product_id", productId)
	return &product, nil
}

func (r *Repository) GetProductById(ctx context.Context, logger *slog.Logger, productId int) (*models.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
	SELECT id, product_id, user_id, COALESCE(private_key,'') as private_key, created_at, updated_at, public, product_name
	FROM products
	WHERE product_id = $1
	`
	rows, err := r.pool.Query(ctx, query, productId)
	if err != nil {
		logger.Error("failed to get product by id", "product_id", productId, "error", err.Error())
		return nil, err
	}

	product, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Product])
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *Repository) CreateProduct(ctx context.Context, logger *slog.Logger, data *domain.CreateProductDetails) (*models.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
	INSERT INTO products (product_name, user_id, public)
	VALUES($1, $2, $3)
	RETURNING id,product_id,product_name,user_id,created_at,updated_at
	`
	rows, err := r.pool.Query(ctx, query, data.ProductName, data.UserId, data.Public)
	if err != nil {
		logger.Error("failed to create product", "product_name", data.ProductName, "error", err.Error())
		return nil, err
	}
	product, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[models.Product])
	if err != nil {
		return nil, err
	}
	logger.Info("product created", "product_name", data.ProductName)
	return &product, nil
}

func (r *Repository) ToggleProductVisibility(ctx context.Context, logger *slog.Logger, productId int, public bool) (*models.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
	UPDATE products
	SET public = $1,updated_at = CURRENT_TIMESTAMP
	WHERE product_id = $2
	RETURNING id, product_id, product_name,user_id, created_at, updated_at, public
	`
	rows, err := r.pool.Query(ctx, query, public, productId)
	if err != nil {
		logger.Error("failed to toggle product visibility", "product_id", productId, "public", public, "error", err.Error())
		return nil, err
	}
	product, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[models.Product])
	if err != nil {
		return nil, err
	}
	logger.Info("product visibility toggled in repository", "product_id", productId, "public", public)
	return &product, nil

}
