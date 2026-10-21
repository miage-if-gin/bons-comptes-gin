package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
	"github.com/miage-if-gin/bons-comptes-gin/internal/repository"
)

var (
	ErrPorteMonnaieIntrouvable = errors.New("porte-monnaie introuvable")
	ErrAccesRefuse             = errors.New("vous n'êtes pas membre de ce porte-monnaie")
	ErrPorteMonnaieClos        = errors.New("ce porte-monnaie est clos : ses dépenses ne peuvent plus changer")
	ErrLibelleVide             = errors.New("le libellé ne peut pas être vide")
	ErrLibelleTropLong         = errors.New("le libellé ne doit pas dépasser 200 caractères")
	ErrDateInvalide            = errors.New("date invalide")
	ErrPayeurInvalide          = errors.New("le payeur doit être un membre du porte-monnaie")
	ErrDepenseIntrouvable      = errors.New("dépense introuvable")
	ErrSuppressionInterdite    = errors.New("seul le payeur (ou, à défaut, le créateur) peut supprimer cette dépense")
)

// DepensesParPage : taille d'une page du listing.
const DepensesParPage = 50

// LongueurMaxLibelle : nombre maximal de caractères d'un libellé.
const LongueurMaxLibelle = 200

// SaisieDepense contient les champs du formulaire, tels que tapés.
// Tout est en texte : c'est au service de valider et de convertir.
type SaisieDepense struct {
	Libelle  string
	Montant  string
	Date     string // format AAAA-MM-JJ, celui de <input type="date">
	PayeurID string // vide = dépense non attribuée
}

// LigneDepense est une ligne du listing, enrichie de ce que la page doit
// savoir : le montant formaté et le droit de suppression du visiteur.
type LigneDepense struct {
	domain.ExpenseLine
	PeutSupprimer bool
}

// Montant est appelé depuis le template : {{ .Montant }}.
func (l LigneDepense) Montant() string { return FormatMontant(l.AmountCents) }

// PageDepenses rassemble tout ce qu'affiche la page des dépenses.
type PageDepenses struct {
	Wallet     domain.Wallet
	Lignes     []LigneDepense
	Membres    []domain.Account
	NbDepenses int64
	TotalCents int64
	Page       int
	NbPages    int
}

func (p PageDepenses) Total() string       { return FormatMontant(p.TotalCents) }
func (p PageDepenses) Ouvert() bool        { return p.Wallet.Status == "open" }
func (p PageDepenses) APrecedente() bool   { return p.Page > 1 }
func (p PageDepenses) ASuivante() bool     { return p.Page < p.NbPages }
func (p PageDepenses) PagePrecedente() int { return p.Page - 1 }
func (p PageDepenses) PageSuivante() int   { return p.Page + 1 }

// PeutSupprimer applique la règle de l'énoncé : « supprimer une dépense par
// le compte dépenseur uniquement (ou créateur sinon) ».
// S'il y a un payeur, lui seul peut supprimer ; sinon, seul le créateur.
func PeutSupprimer(payerID *int64, createdBy, accountID int64) bool {
	if payerID != nil {
		return *payerID == accountID
	}
	return createdBy == accountID
}

type ExpenseService struct {
	expenses *repository.ExpenseRepository
	wallets  *repository.WalletRepository
}

func NewExpenseService(e *repository.ExpenseRepository, w *repository.WalletRepository) *ExpenseService {
	return &ExpenseService{expenses: e, wallets: w}
}

// Lister renvoie une page de dépenses du porte-monnaie, le total et les membres.
func (s *ExpenseService) Lister(ctx context.Context, walletID, accountID int64, page int) (*PageDepenses, error) {
	wallet, err := s.verifierAcces(ctx, walletID, accountID)
	if err != nil {
		return nil, err
	}

	nb, total, err := s.expenses.Totals(ctx, walletID)
	if err != nil {
		return nil, err
	}

	// Nombre de pages arrondi au supérieur, au moins une.
	nbPages := int((nb + DepensesParPage - 1) / DepensesParPage)
	if nbPages < 1 {
		nbPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > nbPages {
		page = nbPages
	}

	lignes, err := s.expenses.ListByWallet(ctx, walletID, DepensesParPage, (page-1)*DepensesParPage)
	if err != nil {
		return nil, err
	}

	membres, err := s.wallets.AcceptedMembers(ctx, walletID)
	if err != nil {
		return nil, err
	}

	resultat := &PageDepenses{
		Wallet:     *wallet,
		Membres:    membres,
		NbDepenses: nb,
		TotalCents: total,
		Page:       page,
		NbPages:    nbPages,
	}
	for _, l := range lignes {
		resultat.Lignes = append(resultat.Lignes, LigneDepense{
			ExpenseLine: l,
			// Rien ne se supprime dans un porte-monnaie clos.
			PeutSupprimer: resultat.Ouvert() && PeutSupprimer(l.PayerID, l.CreatedBy, accountID),
		})
	}
	return resultat, nil
}

// Declarer valide la saisie et enregistre une nouvelle dépense.
func (s *ExpenseService) Declarer(ctx context.Context, walletID, accountID int64, saisie SaisieDepense) (*domain.Expense, error) {
	wallet, err := s.verifierAcces(ctx, walletID, accountID)
	if err != nil {
		return nil, err
	}
	if wallet.Status != "open" {
		return nil, ErrPorteMonnaieClos
	}

	libelle := strings.TrimSpace(saisie.Libelle)
	if libelle == "" {
		return nil, ErrLibelleVide
	}
	// utf8.RuneCountInString compte les caractères ; len() compterait les
	// octets, et un libellé accentué ou en cyrillique serait mal mesuré.
	if utf8.RuneCountInString(libelle) > LongueurMaxLibelle {
		return nil, ErrLibelleTropLong
	}

	montant, err := ParseMontant(saisie.Montant)
	if err != nil {
		return nil, err
	}

	date := time.Now()
	if d := strings.TrimSpace(saisie.Date); d != "" {
		date, err = time.ParseInLocation("2006-01-02", d, time.Local)
		if err != nil {
			return nil, ErrDateInvalide
		}
	}

	var payeur *int64
	if p := strings.TrimSpace(saisie.PayeurID); p != "" {
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, ErrPayeurInvalide
		}
		membre, err := s.wallets.IsAcceptedMember(ctx, walletID, id)
		if err != nil {
			return nil, err
		}
		if !membre {
			return nil, ErrPayeurInvalide
		}
		payeur = &id
	}

	depense := &domain.Expense{
		WalletID:    walletID,
		Label:       libelle,
		AmountCents: montant,
		PayerID:     payeur,
		CreatedBy:   accountID,
		SpentAt:     date,
	}
	if err := s.expenses.Create(ctx, depense); err != nil {
		return nil, err
	}
	return depense, nil
}

// Supprimer efface une dépense si le compte en a le droit.
func (s *ExpenseService) Supprimer(ctx context.Context, walletID, expenseID, accountID int64) error {
	wallet, err := s.verifierAcces(ctx, walletID, accountID)
	if err != nil {
		return err
	}
	if wallet.Status != "open" {
		return ErrPorteMonnaieClos
	}

	depense, err := s.expenses.FindInWallet(ctx, walletID, expenseID)
	if err != nil {
		return err
	}
	if depense == nil {
		return ErrDepenseIntrouvable
	}
	if !PeutSupprimer(depense.PayerID, depense.CreatedBy, accountID) {
		return ErrSuppressionInterdite
	}
	return s.expenses.Delete(ctx, walletID, expenseID)
}

// verifierAcces renvoie le porte-monnaie si le compte en est membre accepté.
func (s *ExpenseService) verifierAcces(ctx context.Context, walletID, accountID int64) (*domain.Wallet, error) {
	wallet, err := s.wallets.FindByID(ctx, walletID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrPorteMonnaieIntrouvable
	}

	membre, err := s.wallets.IsAcceptedMember(ctx, walletID, accountID)
	if err != nil {
		return nil, err
	}
	if !membre {
		return nil, ErrAccesRefuse
	}
	return wallet, nil
}
