package domain

import "time"

// Expense est une dépense déclarée dans un porte-monnaie.
//
// Le montant est en centimes (ADR-004) : 12,50 € est stocké 1250.
// PayerID est un pointeur car le compte dépenseur est facultatif :
// nil correspond à NULL en base, « dépense non attribuée ».
type Expense struct {
	ID          int64 `gorm:"primaryKey"`
	WalletID    int64
	Label       string
	AmountCents int64
	PayerID     *int64
	CreatedBy   int64
	SpentAt     time.Time
	CreatedAt   time.Time
}

func (Expense) TableName() string { return "expenses" }

// ExpenseLine est une ligne du listing : la dépense et les noms
// du payeur et du créateur, obtenus par jointure en une seule requête.
type ExpenseLine struct {
	ID          int64
	WalletID    int64
	Label       string
	AmountCents int64
	PayerID     *int64
	CreatedBy   int64
	SpentAt     time.Time
	PayerName   string
	CreatorName string
}
