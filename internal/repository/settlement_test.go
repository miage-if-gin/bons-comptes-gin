package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/miage-if-gin/bons-comptes-gin/internal/db"
	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
	"github.com/miage-if-gin/bons-comptes-gin/internal/repository"
)

// Test d'intégration : il faut une vraie base PostgreSQL, car c'est la
// requête SQL elle-même qu'on vérifie. La CI en fournit une ; en local,
// `make test` utilise celle du .env. Sans base joignable, le test est
// ignoré plutôt qu'en échec.
func TestPaidByAccount_DepenseSansPayeurNeCreditePersonne(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL non défini : test d'intégration ignoré")
	}
	gdb, err := db.Open(dsn, false)
	if err != nil {
		t.Skipf("base injoignable, test ignoré : %v", err)
	}
	t.Cleanup(func() { _ = db.Close(gdb) })
	if err := db.Migrate(gdb); err != nil {
		t.Fatalf("migrations : %v", err)
	}

	// Tout se passe dans une transaction annulée à la fin : la base reste
	// propre, même si c'est la base de développement.
	tx := gdb.Begin()
	t.Cleanup(func() { tx.Rollback() })

	suffixe := time.Now().UnixNano()
	creerCompte := func(nom string) domain.Account {
		a := domain.Account{
			Email:        fmt.Sprintf("%s-%d@test.local", nom, suffixe),
			PasswordHash: "x",
			DisplayName:  nom,
		}
		if err := tx.Create(&a).Error; err != nil {
			t.Fatalf("création du compte %s : %v", nom, err)
		}
		return a
	}
	alice, bob := creerCompte("alice"), creerCompte("bob")

	wallet := domain.Wallet{Name: "test", OwnerID: alice.ID, Status: "open"}
	if err := tx.Create(&wallet).Error; err != nil {
		t.Fatalf("création du porte-monnaie : %v", err)
	}

	// Alice saisit les trois dépenses, dont une sans payeur.
	depenses := []domain.Expense{
		{WalletID: wallet.ID, Label: "payée par Alice", AmountCents: 1200, PayerID: &alice.ID, CreatedBy: alice.ID, SpentAt: time.Now()},
		{WalletID: wallet.ID, Label: "sans payeur", AmountCents: 10000, CreatedBy: alice.ID, SpentAt: time.Now()},
		{WalletID: wallet.ID, Label: "payée par Bob", AmountCents: 5000, PayerID: &bob.ID, CreatedBy: alice.ID, SpentAt: time.Now()},
	}
	if err := tx.Create(&depenses).Error; err != nil {
		t.Fatalf("création des dépenses : %v", err)
	}

	payes, err := repository.NewSettlementRepository(tx).PaidByAccount(context.Background(), wallet.ID)
	if err != nil {
		t.Fatalf("PaidByAccount : %v", err)
	}

	obtenu := make(map[int64]int64)
	for _, p := range payes {
		obtenu[p.AccountID] = p.PaidCents
	}
	if obtenu[alice.ID] != 1200 {
		t.Errorf("Alice créditée de %d centimes, attendu 1200 : la dépense sans payeur ne doit créditer personne", obtenu[alice.ID])
	}
	if obtenu[bob.ID] != 5000 {
		t.Errorf("Bob crédité de %d centimes, attendu 5000", obtenu[bob.ID])
	}
	if len(payes) != 2 {
		t.Errorf("%d lignes, attendu 2 (une par payeur réel) : %+v", len(payes), payes)
	}
}
