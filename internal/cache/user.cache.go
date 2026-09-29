package cache

import (
	"context"
	"time"

	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/models"
	"github.com/redis/go-redis/v9"
)

func (r *Sredis) GetFullUser(ctx context.Context, identifier string, by string) (*models.User, error) {
	var user models.User
	key := domain.RedirRedisUserPrefix + by + ":" + identifier
	s := r.rdb.HGetAll(ctx, key)
	u, err := s.Result()
	if err != nil {
		return nil, err
	}
	if len(u) == 0 {
		return nil, redis.Nil
	}
	err = s.Scan(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Sredis) SetFullUser(ctx context.Context, identifier string, by string, u models.User) error {
	key := domain.RedirRedisUserPrefix + by + ":" + identifier
	exp := time.Minute * 5
	m := structToInterface(u)
	pipe := r.rdb.Pipeline()
	pipe.HSet(ctx, key, m)
	pipe.Expire(ctx, key, exp)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Sredis) RevokeFullUser(ctx context.Context, identifier string, by string) error {
	key := domain.RedirRedisUserPrefix + by + ":" + identifier
	return r.rdb.Del(ctx, key).Err()
}
