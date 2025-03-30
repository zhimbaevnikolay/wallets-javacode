package redisworker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
	"wallets/internal/lib/sl"
	"wallets/internal/models"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	WALLETID  = "wallet_id"
	OPERATION = "operation"
	AMOUNT    = "amount"
)

type RedisWorker struct {
	stream string
	client *redis.Client
}

func New(stream string, client *redis.Client) *RedisWorker {

	return &RedisWorker{
		stream: stream,
		client: client,
	}
}

func (w *RedisWorker) AddToQueue(ctx context.Context, walletID uuid.UUID, operation string, amount int64) (string, error) {
	const op = "storage.redisworker.AddToQueue"

	tx, err := w.client.XAdd(ctx, &redis.XAddArgs{
		Stream: w.stream,
		Values: map[string]interface{}{
			WALLETID:  walletID.String(),
			OPERATION: operation,
			AMOUNT:    amount,
		},
	}).Result()

	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return tx, nil
}

func (w *RedisWorker) StartWorker(ctx context.Context, ch chan models.QueueTransaction, logger *slog.Logger) error {
	const op = "storage.redisworker.StartWorker"
	lastID := "0-0"

	log := logger.With("op", op)

	for {
		select {
		case <-ctx.Done():
			close(ch)
			log.Info("Worker stopped by context")
			return ctx.Err()

		default:
			messages, err := w.client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{w.stream, lastID},
				Count:   1,
				Block:   5 * time.Second,
			}).Result()

			if err == redis.Nil {
				continue
			} else if err != nil {
				log.Error("Failed to read from stream", sl.Err(err))
				continue
			}

			for _, stream := range messages {
				for _, msg := range stream.Messages {

					wID, ok := msg.Values[WALLETID].(string)
					if !ok {
						log.Error("Invalid wallet_id type", sl.Err(errors.New("can't parse wallet_id")))
						continue
					}

					opType, ok := msg.Values[OPERATION].(string)
					if !ok {
						log.Error("Invalid operation type", sl.Err(errors.New("can't parse operation")))
						continue
					}

					amountStr, ok := msg.Values[AMOUNT].(string)
					if !ok {
						log.Error("Invalid amount type", sl.Err(errors.New("can't parse amount")))
						continue
					}

					amount, err := strconv.Atoi(amountStr)
					if err != nil {
						log.Error("Failed to parse amount", sl.Err(err))
						continue
					}

					tx := models.Transactions{
						WalletID:      uuid.FromStringOrNil(wID),
						OperationType: models.OperationType(opType),
						Amount:        int64(amount),
					}

					ch <- models.QueueTransaction{RedisTxID: msg.ID, Transactions: tx}

					lastID = msg.ID

					w.DelFromQueue(ctx, msg.ID)
				}
			}
		}
	}
}

func (w *RedisWorker) DelFromQueue(ctx context.Context, TxID string) error {
	_, err := w.client.XDel(ctx, w.stream, TxID).Result()
	return err
}
