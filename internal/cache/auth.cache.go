package cache

import (
	"context"
	"time"

	"github.com/ajaka/redir/internal/domain"
)

func (r *Sredis) Clean() {
	r.rdb.Close()
}

func (r *Sredis) SetUserOnline(ctx context.Context, sessionId string, u *domain.
	LightUser) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	key := domain.RedirRedisSessionPrefix + sessionId
	exp := time.Hour * 24
	m := structToInterface(*u)
	pipe := r.rdb.Pipeline()
	pipe.HSet(ctx, key, m)
	pipe.Expire(ctx, key, exp)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(exp), nil
}

func (r *Sredis) GetUser(ctx context.Context, sessionId string) (*domain.
	LightUser, bool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var user domain.LightUser
	key := domain.RedirRedisSessionPrefix + sessionId
	s := r.rdb.HGetAll(ctx, key)
	u, err := s.Result()
	if err != nil || len(u) == 0 {
		return nil, false
	}
	err = s.Scan(&user)
	if err != nil {
		return nil, false
	}
	return &user, true
}

func (r *Sredis) RevokeUser(ctx context.Context, sessionId string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	key := domain.RedirRedisSessionPrefix + sessionId
	err := r.rdb.Del(ctx, key).Err()
	return err
}

func (r *Sredis) SetVerifcationUser(ctx context.Context, email string, token string) error {
	key := domain.RedirVerificationTokenPrefix + token
	rev := domain.RedirVerificationEmailPrefix + email
	exp := time.Minute * 15
	pipe := r.rdb.Pipeline()
	pipe.SetNX(ctx, key, email, exp)
	pipe.SetNX(ctx, rev, token, exp)
	_, err := pipe.Exec(ctx)
	return err
}
func (r *Sredis) GetVerifcationUser(ctx context.Context, token string) (string, error) {
	key := domain.RedirVerificationTokenPrefix + token
	value, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	r.rdb.Del(ctx, key).Err()
	r.rdb.Del(ctx, domain.RedirVerificationEmailPrefix+value).Err()
	return value, nil
}
func (r *Sredis) GetVerificationTokenByEmail(ctx context.Context, email string) (string, error) {
	key := domain.RedirVerificationEmailPrefix + email
	v, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return v, nil
}
