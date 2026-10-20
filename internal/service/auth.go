package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
	"github.com/miage-if-gin/bons-comptes-gin/internal/repository"
)

// ErrIdentifiantsInvalides est volontairement unique : distinguer « email
// inconnu » de « mot de passe faux » permettrait d'énumérer les comptes
// existants.
var ErrIdentifiantsInvalides = errors.New("email ou mot de passe incorrect")

// DureeSession : au-delà, la session expire et l'utilisateur se reconnecte.
const DureeSession = 7 * 24 * time.Hour

type AuthService struct {
	accounts *repository.AccountRepository
	sessions *repository.SessionRepository
}

func NewAuthService(a *repository.AccountRepository, s *repository.SessionRepository) *AuthService {
	return &AuthService{accounts: a, sessions: s}
}

// Login vérifie les identifiants et ouvre une session.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	account, err := s.accounts.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if account == nil {
		// On hache quand même une valeur bidon : sans cela, un email inconnu
		// répondrait instantanément alors qu'un email connu prendrait le temps
		// d'un bcrypt. Cette différence de latence suffit à énumérer les
		// comptes existants.
		bcrypt.CompareHashAndPassword(
			[]byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"),
			[]byte(password),
		)
		return nil, ErrIdentifiantsInvalides
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return nil, ErrIdentifiantsInvalides
	}

	token, err := genererToken()
	if err != nil {
		return nil, err
	}

	session := &domain.Session{
		Token:     token,
		AccountID: account.ID,
		ExpiresAt: time.Now().Add(DureeSession),
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// Logout supprime la session côté serveur.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

// AccountFromToken renvoie le compte associé à une session valide, ou nil.
func (s *AuthService) AccountFromToken(ctx context.Context, token string) (*domain.Account, error) {
	if token == "" {
		return nil, nil
	}

	session, err := s.sessions.FindValid(ctx, token)
	if err != nil || session == nil {
		return nil, err
	}
	return s.accounts.FindByID(ctx, session.AccountID)
}

// genererToken produit 32 octets aléatoires encodés en base64 URL.
//
// crypto/rand puise dans l'entropie du système, contrairement à math/rand
// dont la suite est prédictible à partir de la graine. Un token de session
// prédictible permettrait d'usurper n'importe quelle connexion.
func genererToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
