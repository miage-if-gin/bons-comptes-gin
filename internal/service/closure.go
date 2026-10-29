package service

import (
	"context"
	"errors"
	"sort"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
	"github.com/miage-if-gin/bons-comptes-gin/internal/repository"
)

var (
	ErrClotureReservee       = errors.New("seul le créateur du porte-monnaie peut le clôturer")
	ErrDejaClos              = errors.New("ce porte-monnaie est déjà clos")
	ErrPasEncoreClos         = errors.New("le porte-monnaie doit être clos avant de rembourser")
	ErrVirementIntrouvable   = errors.New("virement introuvable")
	ErrRemboursementInterdit = errors.New("seul le compte qui doit l'argent peut effectuer ce remboursement")
	ErrDejaRembourse         = errors.New("ce virement a déjà été remboursé")
)

type Solde struct {
	AccountID int64
	Nom       string
	PayeCents int64
	PartCents int64
}

func (s Solde) SoldeCents() int64 { return s.PayeCents - s.PartCents }

func (s Solde) Paye() string { return FormatMontant(s.PayeCents) }
func (s Solde) Part() string { return FormatMontant(s.PartCents) }
func (s Solde) Net() string  { return FormatMontant(s.SoldeCents()) }

func (s Solde) Montant() string {
	if s.SoldeCents() < 0 {
		return FormatMontant(-s.SoldeCents())
	}
	return FormatMontant(s.SoldeCents())
}

type Virement struct {
	De           int64
	DeNom        string
	Vers         int64
	VersNom      string
	MontantCents int64
}

func (v Virement) Montant() string { return FormatMontant(v.MontantCents) }

func CalculerSoldes(membres []domain.Account, payes []domain.AccountPaid) []Solde {
	parCompte := make(map[int64]*Solde)

	ids := make([]int64, 0, len(membres))
	for _, m := range membres {
		parCompte[m.ID] = &Solde{AccountID: m.ID, Nom: m.DisplayName}
		ids = append(ids, m.ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var total int64
	for _, p := range payes {
		total += p.PaidCents
		s, ok := parCompte[p.AccountID]
		if !ok {
			s = &Solde{AccountID: p.AccountID, Nom: p.DisplayName}
			parCompte[p.AccountID] = s
		}
		s.PayeCents += p.PaidCents
	}

	if n := int64(len(ids)); n > 0 {
		base, reste := total/n, total%n
		for i, id := range ids {
			parCompte[id].PartCents = base
			if int64(i) < reste {
				parCompte[id].PartCents++
			}
		}
	}

	soldes := make([]Solde, 0, len(parCompte))
	for _, s := range parCompte {
		soldes = append(soldes, *s)
	}
	sort.Slice(soldes, func(i, j int) bool { return soldes[i].AccountID < soldes[j].AccountID })
	return soldes
}

func MinimiserVirements(soldes []Solde) []Virement {
	type position struct {
		id      int64
		nom     string
		montant int64 // toujours positif
	}

	var debiteurs, crediteurs []position
	for _, s := range soldes {
		switch v := s.SoldeCents(); {
		case v < 0:
			debiteurs = append(debiteurs, position{s.AccountID, s.Nom, -v})
		case v > 0:
			crediteurs = append(crediteurs, position{s.AccountID, s.Nom, v})
		}
	}

	trier := func(p []position) {
		sort.Slice(p, func(i, j int) bool {
			if p[i].montant != p[j].montant {
				return p[i].montant > p[j].montant
			}
			return p[i].id < p[j].id
		})
	}
	trier(debiteurs)
	trier(crediteurs)

	var virements []Virement
	payer := func(d, c *position, montant int64) {
		virements = append(virements, Virement{
			De: d.id, DeNom: d.nom,
			Vers: c.id, VersNom: c.nom,
			MontantCents: montant,
		})
		d.montant -= montant
		c.montant -= montant
	}

	// Étape 1 : paires exactes. On range les créanciers par montant.
	parMontant := make(map[int64][]int)
	for j, c := range crediteurs {
		parMontant[c.montant] = append(parMontant[c.montant], j)
	}
	for i := range debiteurs {
		d := &debiteurs[i]
		if candidats := parMontant[d.montant]; len(candidats) > 0 {
			j := candidats[0]
			parMontant[d.montant] = candidats[1:]
			payer(d, &crediteurs[j], d.montant)
		}
	}

	// Étape 2 : glouton sur ce qui reste.
	i, j := 0, 0
	for i < len(debiteurs) && j < len(crediteurs) {
		d, c := &debiteurs[i], &crediteurs[j]
		switch {
		case d.montant == 0:
			i++
		case c.montant == 0:
			j++
		default:
			payer(d, c, min(d.montant, c.montant))
		}
	}
	return virements
}

// Service : vérifie les droits, lit la base, appelle les calculs purs.

// LigneVirement est un virement enregistré  tel qu'affiché après clôture.
type LigneVirement struct {
	domain.SettlementLine
	PeutRembourser bool
}

func (l LigneVirement) Montant() string { return FormatMontant(l.AmountCents) }
func (l LigneVirement) Rembourse() bool { return l.SettledAt != nil }

// PageCloture rassemble tout ce qu'affiche la page de clôture.
type PageCloture struct {
	Wallet          domain.Wallet
	Soldes          []Solde
	Apercu          []Virement      // porte-monnaie ouvert : ce que donnerait la clôture
	Virements       []LigneVirement // porte-monnaie clos : virements enregistrés
	TotalCents      int64
	EstProprietaire bool
}

func (p PageCloture) Ouvert() bool  { return p.Wallet.Status == "open" }
func (p PageCloture) Total() string { return FormatMontant(p.TotalCents) }

// Debiteurs : la liste « qui doit donner de l'argent, et combien ».
func (p PageCloture) Debiteurs() []Solde {
	var res []Solde
	for _, s := range p.Soldes {
		if s.SoldeCents() < 0 {
			res = append(res, s)
		}
	}
	return res
}

// Crediteurs : la liste « qui doit recevoir de l'argent, et combien ».
func (p PageCloture) Crediteurs() []Solde {
	var res []Solde
	for _, s := range p.Soldes {
		if s.SoldeCents() > 0 {
			res = append(res, s)
		}
	}
	return res
}

type ClosureService struct {
	wallets     *repository.WalletRepository
	settlements *repository.SettlementRepository
}

func NewClosureService(w *repository.WalletRepository, s *repository.SettlementRepository) *ClosureService {
	return &ClosureService{wallets: w, settlements: s}
}

// Consulter renvoie les soldes du porte-monnaie, avec l'aperçu des virements
func (s *ClosureService) Consulter(ctx context.Context, walletID, accountID int64) (*PageCloture, error) {
	wallet, err := s.verifierAcces(ctx, walletID, accountID)
	if err != nil {
		return nil, err
	}

	soldes, err := s.soldes(ctx, walletID)
	if err != nil {
		return nil, err
	}

	page := &PageCloture{
		Wallet:          *wallet,
		Soldes:          soldes,
		EstProprietaire: wallet.OwnerID == accountID,
	}
	for _, so := range soldes {
		page.TotalCents += so.PayeCents
	}

	if page.Ouvert() {
		page.Apercu = MinimiserVirements(soldes)
		return page, nil
	}

	lignes, err := s.settlements.ListByWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}
	for _, l := range lignes {
		page.Virements = append(page.Virements, LigneVirement{
			SettlementLine: l,
			PeutRembourser: l.SettledAt == nil && l.FromAccountID == accountID,
		})
	}
	return page, nil
}

// Cloturer fige le porte-monnaie et enregistre les virements à effectuer
func (s *ClosureService) Cloturer(ctx context.Context, walletID, accountID int64) error {
	wallet, err := s.verifierAcces(ctx, walletID, accountID)
	if err != nil {
		return err
	}
	if wallet.OwnerID != accountID {
		return ErrClotureReservee
	}
	if wallet.Status != "open" {
		return ErrDejaClos
	}

	soldes, err := s.soldes(ctx, walletID)
	if err != nil {
		return err
	}

	var aEnregistrer []domain.Settlement
	for _, v := range MinimiserVirements(soldes) {
		aEnregistrer = append(aEnregistrer, domain.Settlement{
			WalletID:      walletID,
			FromAccountID: v.De,
			ToAccountID:   v.Vers,
			AmountCents:   v.MontantCents,
		})
	}

	err = s.settlements.Close(ctx, walletID, aEnregistrer)
	if errors.Is(err, repository.ErrWalletAlreadyClosed) {
		return ErrDejaClos
	}
	return err
}

func (s *ClosureService) Rembourser(ctx context.Context, walletID, settlementID, accountID int64) error {
	wallet, err := s.verifierAcces(ctx, walletID, accountID)
	if err != nil {
		return err
	}
	if wallet.Status != "closed" {
		return ErrPasEncoreClos
	}

	virement, err := s.settlements.FindInWallet(ctx, walletID, settlementID)
	if err != nil {
		return err
	}
	if virement == nil {
		return ErrVirementIntrouvable
	}
	if virement.FromAccountID != accountID {
		return ErrRemboursementInterdit
	}
	if virement.SettledAt != nil {
		return ErrDejaRembourse
	}
	return s.settlements.MarkSettled(ctx, walletID, settlementID)
}

func (s *ClosureService) soldes(ctx context.Context, walletID int64) ([]Solde, error) {
	membres, err := s.wallets.AcceptedMembers(ctx, walletID)
	if err != nil {
		return nil, err
	}
	payes, err := s.settlements.PaidByAccount(ctx, walletID)
	if err != nil {
		return nil, err
	}
	return CalculerSoldes(membres, payes), nil
}

// verifierAcces renvoie le porte-monnaie si le compte en est membre accepté.
// Mêmes erreurs que pour les dépenses : le handler les traduit pareil.
func (s *ClosureService) verifierAcces(ctx context.Context, walletID, accountID int64) (*domain.Wallet, error) {
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
