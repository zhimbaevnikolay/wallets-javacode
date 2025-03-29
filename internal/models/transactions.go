package models

import (
	"time"

	"github.com/gofrs/uuid"
)

const (
	DEPOSIT  OperationType = "DEPOSIT"
	WITHDRAW OperationType = "WITHDRAW"
)

type Transactions struct {
	ID            uuid.UUID `db:"id"`
	WalletID      uuid.UUID `db:"wallet_id"`
	OperationType `db:"operation_type"`
	Amount        int64     `db:"amount"`
	Created_at    time.Time `db:"created_at"`
}
