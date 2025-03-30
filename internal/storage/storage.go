package storage

import (
	"context"
	"fmt"
	"log/slog"
	"wallets/internal/lib/sl"
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

type Worker interface {
	AddToQueue(ctx context.Context, walletID uuid.UUID, operation string, amount int64) (string, error)
	StartWorker(ctx context.Context, ch chan models.QueueTransaction, logger *slog.Logger) error
	DelFromQueue(ctx context.Context, TxID string) error
}

type Storage struct {
	DB       DBRepos
	Redis    CacheRepos
	Worker   Worker
	WorkerCh chan models.QueueTransaction
}

func NewStorage(db DBRepos, cache CacheRepos, worker Worker) *Storage {
	return &Storage{
		DB:       db,
		Redis:    cache,
		Worker:   worker,
		WorkerCh: make(chan models.QueueTransaction),
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

	balance, err = r.DB.GetBalance(ctx, walletID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	_ = r.Redis.SetCachedBalance(ctx, walletID, balance) //TODO обработать ошибку выше, пока так

	return balance, nil
}

func (r *Storage) UpdateBalance(ctx context.Context, walletID uuid.UUID, operationType models.OperationType, amount int64) (models.Transactions, error) {
	const op = "storage.UpdateBalance"

	tx, err := r.DB.UpdateBalance(ctx, walletID, operationType, amount)
	if err != nil {
		return models.Transactions{}, fmt.Errorf("%s: %w", op, err)
	}

	r.Redis.InvalidateCache(ctx, walletID)

	return tx, nil
}

func (r *Storage) StartWorker(ctx context.Context, log *slog.Logger) {
	const op = "storage.StartWorker"

	logger := log.With("op", op)

	go r.Worker.StartWorker(ctx, r.WorkerCh, log)

	for tx := range r.WorkerCh {
		transaction, err := r.UpdateBalance(ctx, tx.Transactions.WalletID, tx.Transactions.OperationType, tx.Transactions.Amount)
		if err != nil {
			logger.Error("failed to update balance", sl.Err(err))

			continue
		}

		logger.Info("Balance updated", slog.Any("transaction", transaction))

	}

}

func (r *Storage) AddToQueue(ctx context.Context, walletID uuid.UUID, operation string, amount int64) (string, error) {
	return r.Worker.AddToQueue(ctx, walletID, operation, amount)
}

//redis-cli XADD update:wallet * wallet_id "7014a4fc-1e9c-47d2-9dd0-8bc8cb3aecb3" operation "WITHDRAW" amount "5000"
