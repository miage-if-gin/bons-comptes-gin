package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

// WalletRepository regroupe les lectures sur les porte-monnaies et leurs
// membres dont ont besoin les autres lots (dépenses, clôture).
type WalletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

// FindByID renvoie le porte-monnaie, ou nil s'il n'existe pas.
func (r *WalletRepository) FindByID(ctx context.Context, id int64) (*domain.Wallet, error) {
	var w domain.Wallet
	err := r.db.WithContext(ctx).First(&w, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// IsAcceptedMember indique si le compte a accepté l'invitation au porte-monnaie.
// Une invitation en attente ou refusée ne donne aucun droit.
func (r *WalletRepository) IsAcceptedMember(ctx context.Context, walletID, accountID int64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Model(&domain.WalletMember{}).
		Where("wallet_id = ? AND account_id = ? AND status = ?", walletID, accountID, "accepted").
		Count(&n).Error
	return n > 0, err
}

// AcceptedMembers liste les membres acceptés, triés par nom.
// Seuls l'ID et le nom sont lus : le hash du mot de passe n'a rien à faire ici.
func (r *WalletRepository) AcceptedMembers(ctx context.Context, walletID int64) ([]domain.Account, error) {
	var accounts []domain.Account
	err := r.db.WithContext(ctx).
		Select("accounts.id, accounts.display_name").
		Joins("JOIN wallet_members wm ON wm.account_id = accounts.id").
		Where("wm.wallet_id = ? AND wm.status = ?", walletID, "accepted").
		Order("accounts.display_name").
		Find(&accounts).Error
	return accounts, err
}
