package domain

import "time"

type Settlement struct {
	ID            int64 `gorm:"primaryKey"`
	WalletID      int64
	FromAccountID int64
	ToAccountID   int64
	AmountCents   int64
	SettledAt     *time.Time
	CreatedAt     time.Time
}

func (Settlement) TableName() string { return "settlements" }

type SettlementLine struct {
	ID            int64
	WalletID      int64
	FromAccountID int64
	ToAccountID   int64
	AmountCents   int64
	SettledAt     *time.Time
	FromName      string
	ToName        string
}

type AccountPaid struct {
	AccountID   int64
	DisplayName string
	PaidCents   int64
}
