package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

// ErrEmailTaken signale une violation de la contrainte d'unicité sur l'email.
var ErrEmailTaken = errors.New("email déjà utilisé")

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// Create insère un compte et renseigne son ID.
func (r *AccountRepository) Create(ctx context.Context, a *domain.Account) error {
	err := r.db.WithContext(ctx).Create(a).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrEmailTaken
	}
	return err
}

// FindByEmail renvoie un compte, ou nil s'il n'existe pas.
func (r *AccountRepository) FindByEmail(ctx context.Context, email string) (*domain.Account, error) {
	var a domain.Account
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&a).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AccountRepository) FindByID(ctx context.Context, id int64) (*domain.Account, error) {
	var a domain.Account
	err := r.db.WithContext(ctx).First(&a, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
