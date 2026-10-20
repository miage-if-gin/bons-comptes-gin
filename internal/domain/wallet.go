package domain

import "time"

type Wallet struct {
	ID        int64 `gorm:"primaryKey"`
	Name      string
	OwnerID   int64
	Status    string
	CreatedAt time.Time
	ClosedAt  *time.Time
}

func (Wallet) TableName() string { return "wallets" }

type WalletMember struct {
	WalletID    int64 `gorm:"primaryKey"`
	AccountID   int64 `gorm:"primaryKey"`
	Status      string
	InvitedBy   *int64
	InvitedAt   time.Time
	RespondedAt *time.Time
}

func (WalletMember) TableName() string { return "wallet_members" }
