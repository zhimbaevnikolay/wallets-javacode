package redis_client

import (
	"context"
	"errors"
	"fmt"
	"time"
	"wallets/internal/config"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	pong      = "PONG"
	LOCKKEY   = "lock:wallet"
	walletKey = "wallet"
)

type RedisClient struct {
	Client           *redis.Client
	lockExporation   time.Duration
	cacheExporation  time.Duration
	maxUnlockRetries int
	baseRetryDelay   time.Duration
}

func New(cfg config.Redis) (*RedisClient, error) {
	const op = "storage.redis_client.New"
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Addr, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	redis_pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if redis_pong != pong {
		return nil, errors.New("unexpected pong from redis")
	}

	return &RedisClient{
		Client:           client,
		lockExporation:   cfg.LockExporation,
		cacheExporation:  cfg.CacheExporation,
		maxUnlockRetries: cfg.MaxUnlockRetries,
		baseRetryDelay:   cfg.BaseRetryDelay,
	}, nil
}

func (r *RedisClient) LockWallet(ctx context.Context, walletID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("%s:%s", LOCKKEY, walletID.String())
	locked, err := r.Client.SetNX(ctx, key, 1, r.lockExporation).Result()
	if err != nil {
		return false, err
	}

	return locked, nil

}

func (r *RedisClient) UnlockWallet(ctx context.Context, walletID uuid.UUID) {
	key := fmt.Sprintf("%s:%s", LOCKKEY, walletID.String())
	r.Client.Del(ctx, key)

}

func (r *RedisClient) TryLockWallet(ctx context.Context, walletID uuid.UUID) (bool, error) {
	for i := 0; i < r.maxUnlockRetries; i++ {
		locked, err := r.LockWallet(ctx, walletID)
		if err != nil {
			return false, err
		}

		if locked {
			return true, nil
		}

		delay := r.baseRetryDelay * time.Duration(1<<i)
		if delay > 300*time.Millisecond {
			delay = 300 * time.Millisecond
		}
		time.Sleep(delay)
	}

	return false, nil
}

func (r *RedisClient) GetCachedBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("%s:%s", walletKey, walletID.String())
	balance, err := r.Client.Get(ctx, key).Int64()
	if err != nil {
		return 0, err
	}

	return balance, nil

}

func (r *RedisClient) SetCachedBalance(ctx context.Context, walletID uuid.UUID, balance int64) error {
	key := fmt.Sprintf("%s:%s", walletKey, walletID)
	return r.Client.Set(ctx, key, balance, r.cacheExporation).Err()
}

func (r *RedisClient) InvalidateCache(ctx context.Context, walletID uuid.UUID) {
	key := fmt.Sprintf("%s:%s", walletKey, walletID)
	r.Client.Del(ctx, key)
}
