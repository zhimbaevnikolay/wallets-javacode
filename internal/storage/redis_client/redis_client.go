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
	pong                 = "PONG"
	expDuration          = 500 * time.Millisecond // TODO убрать в конфиг
	cacheExpDuration     = 10 * time.Minute       // TODO убрать в конфиг
	lockWalletKey        = "lock:wallet"
	walletKey            = "wallet"
	maxLockWalletRetries = 20                    // TODO убрать в конфиг
	lockWalletBaseDelay  = 50 * time.Millisecond // TODO убрать в конфиг
)

type RedisClient struct {
	client *redis.Client
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

	return &RedisClient{client: client}, nil
}

func (r *RedisClient) LockWallet(ctx context.Context, walletID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("%s:%s", lockWalletKey, walletID.String())
	locked, err := r.client.SetNX(ctx, key, 1, expDuration).Result()
	if err != nil {
		return false, err
	}

	return locked, nil

}

func (r *RedisClient) UnlockWallet(ctx context.Context, walletID uuid.UUID) {
	key := fmt.Sprintf("%s:%s", lockWalletKey, walletID.String())
	r.client.Del(ctx, key)

}

func (r *RedisClient) TryLockWallet(ctx context.Context, walletID uuid.UUID) (bool, error) {
	for i := 0; i < maxLockWalletRetries; i++ {
		locked, err := r.LockWallet(ctx, walletID)
		if err != nil {
			return false, err
		}

		if locked {
			return true, nil
		}

		delay := lockWalletBaseDelay * time.Duration(1<<i)
		if delay > 300*time.Millisecond {
			delay = 300 * time.Millisecond
		}
		time.Sleep(delay)
	}

	return false, nil
}

func (r *RedisClient) GetCachedBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("%s:%s", walletKey, walletID.String())
	balance, err := r.client.Get(ctx, key).Int64()
	if err != nil {
		return 0, err
	}

	return balance, nil

}

func (r *RedisClient) SetCachedBalance(ctx context.Context, walletID uuid.UUID, balance int64) error {
	key := fmt.Sprintf("%s:%s", walletKey, walletID)
	return r.client.Set(ctx, key, balance, cacheExpDuration).Err()
}

func (r *RedisClient) InvalidateCache(ctx context.Context, walletID uuid.UUID) {
	key := fmt.Sprintf("%s:%s", walletKey, walletID)
	r.client.Del(ctx, key)
}
