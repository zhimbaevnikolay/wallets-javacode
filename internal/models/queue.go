package models

type QueueTransaction struct {
	RedisTxID string
	Transactions
}
