package models

import (
	"github.com/gofrs/uuid"
)

type OperationType string

type Wallet struct {
	ID      uuid.UUID `db:"id"`
	Balance int64     `db:"balance"`
}
