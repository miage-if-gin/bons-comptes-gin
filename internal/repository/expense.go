package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

type ExpenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

// Create insère une dépense et renseigne son ID.
func (r *ExpenseRepository) Create(ctx context.Context, e *domain.Expense) error {
	return r.db.WithContext(ctx).Create(e).Error
}

// FindInWallet renvoie la dépense si elle appartient bien à ce porte-monnaie,
// sinon nil. Filtrer aussi sur wallet_id empêche de viser la dépense d'un
// autre porte-monnaie en changeant simplement l'ID dans l'URL.
func (r *ExpenseRepository) FindInWallet(ctx context.Context, walletID, id int64) (*domain.Expense, error) {
	var e domain.Expense
	err := r.db.WithContext(ctx).
		Where("id = ? AND wallet_id = ?", id, walletID).
		First(&e).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// Delete supprime une dépense du porte-monnaie.
func (r *ExpenseRepository) Delete(ctx context.Context, walletID, id int64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND wallet_id = ?", id, walletID).
		Delete(&domain.Expense{}).Error
}

// ListByWallet renvoie une page de dépenses, de la plus récente à la plus
// ancienne, avec les noms du payeur et du créateur.
//
// Une seule requête avec deux jointures, plutôt qu'une requête par ligne
// pour aller chercher les noms (le piège N+1 signalé dans l'ADR-001).
// LIMIT/OFFSET : avec beaucoup de dépenses (« Think HUGE »), on ne charge
// jamais tout en mémoire.
func (r *ExpenseRepository) ListByWallet(ctx context.Context, walletID int64, limit, offset int) ([]domain.ExpenseLine, error) {
	var lines []domain.ExpenseLine
	err := r.db.WithContext(ctx).
		Table("expenses e").
		Select(`e.id, e.wallet_id, e.label, e.amount_cents, e.payer_id, e.created_by, e.spent_at,
			COALESCE(p.display_name, '') AS payer_name,
			c.display_name AS creator_name`).
		Joins("LEFT JOIN accounts p ON p.id = e.payer_id").
		Joins("JOIN accounts c ON c.id = e.created_by").
		Where("e.wallet_id = ?", walletID).
		Order("e.spent_at DESC, e.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&lines).Error
	return lines, err
}

// Totals renvoie le nombre de dépenses et leur somme en centimes.
//
// Le calcul est fait par PostgreSQL (COUNT, SUM) : additionner en Go
// obligerait à charger toutes les lignes. SUM sur un BIGINT renvoie un
// NUMERIC, d'où la conversion explicite en BIGINT.
func (r *ExpenseRepository) Totals(ctx context.Context, walletID int64) (count int64, totalCents int64, err error) {
	var res struct {
		Count int64
		Total int64
	}
	err = r.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) AS count, COALESCE(SUM(amount_cents), 0)::BIGINT AS total
			FROM expenses WHERE wallet_id = ?`, walletID).
		Scan(&res).Error
	return res.Count, res.Total, err
}
