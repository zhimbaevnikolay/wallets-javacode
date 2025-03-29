package herrors

import "errors"

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrUnknownOperation  = errors.New("unknown operation")
	ErrLockedWallet      = errors.New("locked wallet")
)
