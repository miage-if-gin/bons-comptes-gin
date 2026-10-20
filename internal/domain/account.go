package domain

import "time"

type Account struct {
	ID           int64 `gorm:"primaryKey"`
	Email        string
	PasswordHash string
	DisplayName  string
	CreatedAt    time.Time
}

func (Account) TableName() string { return "accounts" }
