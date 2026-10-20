// Package service porte les règles métier. Il ne connaît ni Gin ni HTTP,
// ce qui le rend testable sans serveur.
package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
	"github.com/miage-if-gin/bons-comptes-gin/internal/repository"
)

var (
	ErrEmailInvalide    = errors.New("adresse email invalide")
	ErrMotDePasseFaible = errors.New("le mot de passe doit faire au moins 8 caractères")
	ErrNomVide          = errors.New("le nom ne peut pas être vide")
	ErrEmailPris        = repository.ErrEmailTaken
)

type AccountService struct {
	accounts *repository.AccountRepository
}

func NewAccountService(a *repository.AccountRepository) *AccountService {
	return &AccountService{accounts: a}
}

// Register valide les entrées, hache le mot de passe et crée le compte.
func (s *AccountService) Register(ctx context.Context, email, password, displayName string) (*domain.Account, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	if !strings.Contains(email, "@") || len(email) < 3 {
		return nil, ErrEmailInvalide
	}
	if len([]rune(password)) < 8 {
		return nil, ErrMotDePasseFaible
	}
	if displayName == "" {
		return nil, ErrNomVide
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	a := &domain.Account{
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
	}

	if err := s.accounts.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}
