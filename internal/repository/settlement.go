package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

var ErrWalletAlreadyClosed = errors.New("porte-monnaie déjà clos")

type SettlementRepository struct {
	db *gorm.DB
}

func NewSettlementRepository(db *gorm.DB) *SettlementRepository {
	return &SettlementRepository{db: db}
}

func (r *SettlementRepository) PaidByAccount(ctx context.Context, walletID int64) ([]domain.AccountPaid, error) {
	var rows []domain.AccountPaid
	err := r.db.WithContext(ctx).
		Raw(`SELECT a.id AS account_id, a.display_name, SUM(e.amount_cents)::BIGINT AS paid_cents
			FROM expenses e
			JOIN accounts a ON a.id = COALESCE(e.payer_id, e.created_by)
			WHERE e.wallet_id = ?
			GROUP BY a.id, a.display_name
			ORDER BY a.id`, walletID).
		Scan(&rows).Error
	return rows, err
}

func (r *SettlementRepository) Close(ctx context.Context, walletID int64, virements []domain.Settlement) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.Wallet{}).
			Where("id = ? AND status = ?", walletID, "open").
			Updates(map[string]any{"status": "closed", "closed_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrWalletAlreadyClosed
		}

		if len(virements) == 0 {
			return nil
		}
		// Un seul INSERT par lot de 500 lignes plutôt qu'un INSERT par virement.
		return tx.CreateInBatches(&virements, 500).Error
	})
}

func (r *SettlementRepository) ListByWallet(ctx context.Context, walletID int64) ([]domain.SettlementLine, error) {
	var lines []domain.SettlementLine
	err := r.db.WithContext(ctx).
		Table("settlements s").
		Select(`s.id, s.wallet_id, s.from_account_id, s.to_account_id, s.amount_cents, s.settled_at,
			f.display_name AS from_name,
			t.display_name AS to_name`).
		Joins("JOIN accounts f ON f.id = s.from_account_id").
		Joins("JOIN accounts t ON t.id = s.to_account_id").
		Where("s.wallet_id = ?", walletID).
		Order("s.amount_cents DESC, s.id").
		Scan(&lines).Error
	return lines, err
}

func (r *SettlementRepository) FindInWallet(ctx context.Context, walletID, id int64) (*domain.Settlement, error) {
	var s domain.Settlement
	err := r.db.WithContext(ctx).
		Where("id = ? AND wallet_id = ?", id, walletID).
		First(&s).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettlementRepository) MarkSettled(ctx context.Context, walletID, id int64) error {
	return r.db.WithContext(ctx).
		Model(&domain.Settlement{}).
		Where("id = ? AND wallet_id = ? AND settled_at IS NULL", id, walletID).
		Update("settled_at", time.Now()).Error
}
