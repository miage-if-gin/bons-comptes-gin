package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// FindValid renvoie la session si elle existe et n'est pas expirée, sinon nil.
//
// L'expiration est filtrée en SQL plutôt que comparée en Go : une session
// périmée ne doit jamais remonter jusqu'à la couche applicative.
func (r *SessionRepository) FindValid(ctx context.Context, token string) (*domain.Session, error) {
	var s domain.Session
	err := r.db.WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, time.Now()).
		First(&s).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) Delete(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Where("token = ?", token).
		Delete(&domain.Session{}).Error
}

// DeleteExpired purge les sessions périmées. Sans ménage, la table grossit
// indéfiniment. À appeler périodiquement.
func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&domain.Session{})
	return res.RowsAffected, res.Error
}
