package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrTokenNotFound = errors.New("refresh token not found")

type TokenRepo struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewTokenRepo(rdb *redis.Client, ttl time.Duration) *TokenRepo {
	return &TokenRepo{rdb: rdb, ttl: ttl}
}

func refreshKey(token string) string { return "refresh:" + token }
func sessionsKey(userID uint) string { return "user_sessions:" + strconv.FormatUint(uint64(userID), 10) }

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *TokenRepo) Issue(ctx context.Context, userID uint) (string, error) {
	tok, err := newToken()
	if err != nil {
		return "", err
	}
	pipe := r.rdb.TxPipeline()
	pipe.Set(ctx, refreshKey(tok), userID, r.ttl)
	pipe.SAdd(ctx, sessionsKey(userID), tok)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return tok, nil
}

func (r *TokenRepo) Validate(ctx context.Context, token string) (uint, error) {
	v, err := r.rdb.Get(ctx, refreshKey(token)).Result()
	if err != nil {
		return 0, ErrTokenNotFound
	}
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, ErrTokenNotFound
	}
	return uint(id), nil
}

func (r *TokenRepo) Revoke(ctx context.Context, token string) error {
	uid, err := r.Validate(ctx, token)
	if err != nil {
		return err
	}
	pipe := r.rdb.TxPipeline()
	pipe.Del(ctx, refreshKey(token))
	pipe.SRem(ctx, sessionsKey(uid), token)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *TokenRepo) RevokeAllForUser(ctx context.Context, userID uint) error {
	tokens, err := r.rdb.SMembers(ctx, sessionsKey(userID)).Result()
	if err != nil {
		return err
	}
	pipe := r.rdb.TxPipeline()
	for _, tok := range tokens {
		pipe.Del(ctx, refreshKey(tok))
	}
	pipe.Del(ctx, sessionsKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}
