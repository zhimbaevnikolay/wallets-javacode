package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"wallets/internal/config"
	"wallets/internal/herrors"
	"wallets/internal/models"

	"github.com/gofrs/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

const (
	tableWallets     = "wallets"
	tableTransaction = "transactions"
)

type PostgresRepos struct {
	db *sqlx.DB
}

func New(storage config.Storage) (*PostgresRepos, error) {
	const op = "storage.Postgres.New"

	connStr := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
		storage.User, storage.Password, storage.Host, storage.Port, storage.Name, storage.SSLMode)
	db, err := sqlx.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &PostgresRepos{db: db}, nil

}

func (r *PostgresRepos) CreateWallet(ctx context.Context, balance int64) (uuid.UUID, error) {
	const op = "storage.Postgres.CreateWallet"
	var walletID uuid.UUID

	query := fmt.Sprintf("INSERT INTO %s (balance) VALUES ($1) RETURNING id", tableWallets)
	row := r.db.QueryRowContext(ctx, query, balance)

	if err := row.Scan(&walletID); err != nil {
		return uuid.UUID{}, fmt.Errorf("%s: %w", op, err)
	}

	return walletID, nil

}

func (r *PostgresRepos) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	const op = "storage.Postgres.GetBalance"
	var balance int64

	query := fmt.Sprintf("SELECT balance FROM %s WHERE id=$1", tableWallets)
	row := r.db.QueryRowContext(ctx, query, walletID)

	if err := row.Scan(&balance); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			err = herrors.ErrNXUUID
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return balance, nil

}

func (r *PostgresRepos) UpdateBalance(ctx context.Context, walletID uuid.UUID, operationType models.OperationType, amount int64) (models.Transactions, error) {
	const op = "storage.Postgres.UpdateBalance"
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return models.Transactions{}, fmt.Errorf("%s: %w", op, err)
	}

	defer tx.Rollback()

	var balance int64
	getQuery := fmt.Sprintf("SELECT balance FROM %s WHERE id = $1 FOR UPDATE", tableWallets)
	row := tx.QueryRowContext(ctx, getQuery, walletID)

	if err := row.Scan(&balance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = herrors.ErrNXUUID
		}
		return models.Transactions{}, fmt.Errorf("%s: %w", op, err)
	}

	switch operationType {
	case models.DEPOSIT:
		balance += amount

	case models.WITHDRAW:
		if balance < amount {
			return models.Transactions{}, fmt.Errorf("%s: %w", op, herrors.ErrInsufficientFunds)
		}

		balance -= amount
	default:
		return models.Transactions{}, fmt.Errorf("%s: %w", op, herrors.ErrUnknownOperation)
	}

	updateQuery := fmt.Sprintf("UPDATE %s SET balance = $1 WHERE id = $2", tableWallets)
	row = tx.QueryRowContext(ctx, updateQuery, balance, walletID)
	if err := row.Err(); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			err = herrors.ErrNXUUID
		}

		return models.Transactions{}, fmt.Errorf("%s: %w", op, err)
	}

	transaction := models.Transactions{}

	transationQuery := fmt.Sprintf("INSERT INTO %s (wallet_id, operation_type, amount) VALUES ($1, $2, $3) RETURNING id, wallet_id, operation_type, amount, created_at", tableTransaction)
	row = tx.QueryRowContext(ctx, transationQuery, walletID, operationType, amount)

	if err := row.Scan(&transaction.ID, &transaction.WalletID, &transaction.OperationType, &transaction.Amount, &transaction.Created_at); err != nil {

		return models.Transactions{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.Transactions{}, err
	}

	return transaction, nil
}
