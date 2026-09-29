package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/models"
	"github.com/redis/go-redis/v9"
)

func (r *Sredis) GetProduct(ctx context.Context, productId int) (*models.Product, error) {
	var product models.Product
	key := fmt.Sprintf("%s%d", domain.RedirRedisProductPrefix, productId)
	s := r.rdb.HGetAll(ctx, key)
	p, err := s.Result()
	if err != nil {
		return nil, err
	}
	if len(p) == 0 {
		return nil, redis.Nil
	}
	err = s.Scan(&product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *Sredis) SetProduct(ctx context.Context, product models.Product) error {
	key := fmt.Sprintf("%s%d", domain.RedirRedisProductPrefix, product.ProductId)
	exp := 20 * time.Minute
	m := structToInterface(product)
	pipe := r.rdb.Pipeline()
	pipe.HSet(ctx, key, m)
	pipe.Expire(ctx, key, exp)
	_, err := pipe.Exec(ctx)
	return err
}
