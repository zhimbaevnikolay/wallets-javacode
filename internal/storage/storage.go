package storage

import (
	"context"
	"fmt"
	"wallets/internal/herrors"
	"wallets/internal/models"

	"github.com/gofrs/uuid"
)

type DBRepos interface {
	CreateWallet(ctx context.Context, balance int64) (uuid.UUID, error)
	GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error)
	UpdateBalance(ctx context.Context, walletID uuid.UUID, operationType models.OperationType, amount int64) (models.Transactions, error)
}

type CacheRepos interface {
	LockWallet(ctx context.Context, walletID uuid.UUID) (bool, error)
	UnlockWallet(ctx context.Context, walletID uuid.UUID)
	TryLockWallet(ctx context.Context, walletID uuid.UUID) (bool, error)
	GetCachedBalance(ctx context.Context, walletID uuid.UUID) (int64, error)
	SetCachedBalance(ctx context.Context, walletID uuid.UUID, balance int64) error
	InvalidateCache(ctx context.Context, walletID uuid.UUID)
}

type Storage struct {
	DB    DBRepos
	Redis CacheRepos
}

func NewStorage(db DBRepos, cache CacheRepos) *Storage {
	return &Storage{
		DB:    db,
		Redis: cache,
	}
}

func (r *Storage) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	const op = "storage.GetBalance"

	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	balance, err := r.Redis.GetCachedBalance(ctx, walletID)
	if err == nil {
		return balance, nil
	}

	locked, err := r.Redis.TryLockWallet(ctx, walletID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if !locked {
		return 0, herrors.ErrLockedWallet
	}

	defer r.Redis.UnlockWallet(ctx, walletID)

	balance, err = r.DB.GetBalance(ctx, walletID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	_ = r.Redis.SetCachedBalance(ctx, walletID, balance) //TODO обработать ошибку выше, пока так

	return balance, nil
}

func (r *Storage) UpdateBalance(ctx context.Context, walletID uuid.UUID, operationType models.OperationType, amount int64) (models.Transactions, error) {
	const op = "storage.UpdateBalance"

	locked, err := r.Redis.TryLockWallet(ctx, walletID)
	if err != nil {
		return models.Transactions{}, fmt.Errorf("%s: %w", op, err)
	}

	if !locked {
		return models.Transactions{}, herrors.ErrLockedWallet
	}

	defer r.Redis.UnlockWallet(ctx, walletID)

	tx, err := r.DB.UpdateBalance(ctx, walletID, operationType, amount)
	if err != nil {
		return models.Transactions{}, fmt.Errorf("%s: %w", op, err)
	}

	r.Redis.InvalidateCache(ctx, walletID)

	return tx, nil
}
