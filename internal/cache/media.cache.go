package cache

import (
	"context"
	"time"

	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/models"
	"github.com/redis/go-redis/v9"
)

func (r *Sredis) GetMedia(ctx context.Context, publicKey string) (*models.Media, error) {
	var media models.Media
	key := domain.RedirRedisMediaPrefix + publicKey
	s := r.rdb.HGetAll(ctx, key)
	m, _ := s.Result()
	if len(m) == 0 {
		return nil, redis.Nil
	}
	err := s.Scan(&media)
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (r *Sredis) SetMedia(ctx context.Context, media models.Media) error {
	key := domain.RedirRedisMediaPrefix + media.PublicKey
	exp := time.Minute * 5
	pipe := r.rdb.Pipeline()
	m := structToInterface(media)
	pipe.HSet(ctx, key, m)
	pipe.Expire(ctx, key, exp)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Sredis) SetPresignedUrl(ctx context.Context, publicKey string, url string, exp time.Duration) error {
	key := domain.RedirPreSignedUrlPrefix + publicKey
	err := r.rdb.Set(ctx, key, url, exp).Err()
	return err
}

func (r *Sredis) GetPresignedUrl(ctx context.Context, publicKey string) (string, error) {
	key := domain.RedirPreSignedUrlPrefix + publicKey
	p, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return p, nil
}
