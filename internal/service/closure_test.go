package service

import (
	"math/rand/v2"
	"testing"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

func membres(noms ...string) []domain.Account {
	var res []domain.Account
	for i, n := range noms {
		res = append(res, domain.Account{ID: int64(i + 1), DisplayName: n})
	}
	return res
}

func soldeDe(t *testing.T, soldes []Solde, id int64) int64 {
	t.Helper()
	for _, s := range soldes {
		if s.AccountID == id {
			return s.SoldeCents()
		}
	}
	t.Fatalf("compte %d absent des soldes", id)
	return 0
}

func sommeDesSoldes(soldes []Solde) int64 {
	var total int64
	for _, s := range soldes {
		total += s.SoldeCents()
	}
	return total
}

func TestCalculerSoldes_PartageEgal(t *testing.T) {
	// Alice paie 90 € pour trois : chacun doit 30 €.
	soldes := CalculerSoldes(
		membres("Alice", "Bob", "Chloé"),
		[]domain.AccountPaid{{AccountID: 1, DisplayName: "Alice", PaidCents: 9000}},
	)

	attendus := map[int64]int64{1: 6000, 2: -3000, 3: -3000}
	for id, attendu := range attendus {
		if obtenu := soldeDe(t, soldes, id); obtenu != attendu {
			t.Errorf("compte %d : solde %d, attendu %d", id, obtenu, attendu)
		}
	}
}

func TestCalculerSoldes_ResteAttribue(t *testing.T) {
	// 1 € pour trois : 34 + 33 + 33 centimes, pas de centime perdu.
	soldes := CalculerSoldes(
		membres("Alice", "Bob", "Chloé"),
		[]domain.AccountPaid{{AccountID: 2, DisplayName: "Bob", PaidCents: 100}},
	)

	parts := []int64{soldes[0].PartCents, soldes[1].PartCents, soldes[2].PartCents}
	if parts[0] != 34 || parts[1] != 33 || parts[2] != 33 {
		t.Fatalf("parts %v, attendu [34 33 33]", parts)
	}
	if s := sommeDesSoldes(soldes); s != 0 {
		t.Fatalf("somme des soldes %d, attendu 0", s)
	}
}

func TestCalculerSoldes_SansDepense(t *testing.T) {
	soldes := CalculerSoldes(membres("Alice", "Bob"), nil)
	for _, s := range soldes {
		if s.SoldeCents() != 0 {
			t.Fatalf("%s : solde %d, attendu 0", s.Nom, s.SoldeCents())
		}
	}
	if v := MinimiserVirements(soldes); len(v) != 0 {
		t.Fatalf("%d virements, attendu aucun", len(v))
	}
}

func TestCalculerSoldes_PayeurQuiNestPlusMembre(t *testing.T) {
	// David (id 9) a payé mais n'est pas membre accepté : il est remboursé,
	// sans payer de part.
	soldes := CalculerSoldes(
		membres("Alice", "Bob"),
		[]domain.AccountPaid{{AccountID: 9, DisplayName: "David", PaidCents: 1000}},
	)
	if s := soldeDe(t, soldes, 9); s != 1000 {
		t.Fatalf("David : solde %d, attendu 1000", s)
	}
	if s := sommeDesSoldes(soldes); s != 0 {
		t.Fatalf("somme des soldes %d, attendu 0", s)
	}
}

func TestMinimiserVirements_CasSimple(t *testing.T) {
	soldes := []Solde{
		{AccountID: 1, Nom: "Alice", PayeCents: 9000, PartCents: 3000},
		{AccountID: 2, Nom: "Bob", PartCents: 3000},
		{AccountID: 3, Nom: "Chloé", PartCents: 3000},
	}
	v := MinimiserVirements(soldes)

	if len(v) != 2 {
		t.Fatalf("%d virements, attendu 2 : %+v", len(v), v)
	}
	for _, x := range v {
		if x.Vers != 1 || x.MontantCents != 3000 {
			t.Errorf("virement inattendu : %+v", x)
		}
	}
}

func TestMinimiserVirements_PairesExactes(t *testing.T) {
	// Créanciers +7 et +3, débiteurs −6, −3 et −1.
	// Sans l'étape des paires exactes, le glouton seul ferait 4 virements.
	// En réglant d'abord −3 → +3, il n'en faut que 3.
	soldes := []Solde{
		{AccountID: 1, PayeCents: 700},
		{AccountID: 2, PayeCents: 300},
		{AccountID: 3, PartCents: 600},
		{AccountID: 4, PartCents: 300},
		{AccountID: 5, PartCents: 100},
	}
	v := MinimiserVirements(soldes)
	if len(v) != 3 {
		t.Fatalf("%d virements, attendu 3 : %+v", len(v), v)
	}
	verifierQueToutEstRegle(t, soldes, v)
}

// Sur des centaines de cas aléatoires, les virements doivent : tous être positifs, ne jamais aller d'un compte vers lui-même,

func TestMinimiserVirements_Aleatoire(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2)) // graine fixe : résultat reproductible

	for essai := 0; essai < 500; essai++ {
		n := 2 + r.IntN(30)
		var noms []string
		for i := 0; i < n; i++ {
			noms = append(noms, "membre")
		}
		var payes []domain.AccountPaid
		for i := 0; i < n; i++ {
			if r.IntN(3) > 0 {
				payes = append(payes, domain.AccountPaid{AccountID: int64(i + 1), PaidCents: r.Int64N(50_000)})
			}
		}

		soldes := CalculerSoldes(membres(noms...), payes)
		if s := sommeDesSoldes(soldes); s != 0 {
			t.Fatalf("essai %d : somme des soldes %d", essai, s)
		}

		v := MinimiserVirements(soldes)
		if len(v) > n-1 {
			t.Fatalf("essai %d : %d virements pour %d comptes", essai, len(v), n)
		}
		verifierQueToutEstRegle(t, soldes, v)
	}
}

func verifierQueToutEstRegle(t *testing.T, soldes []Solde, virements []Virement) {
	t.Helper()
	reste := make(map[int64]int64)
	for _, s := range soldes {
		reste[s.AccountID] = s.SoldeCents()
	}
	for _, v := range virements {
		if v.MontantCents <= 0 {
			t.Fatalf("virement non positif : %+v", v)
		}
		if v.De == v.Vers {
			t.Fatalf("virement vers soi-même : %+v", v)
		}
		reste[v.De] += v.MontantCents
		reste[v.Vers] -= v.MontantCents
	}
	for id, r := range reste {
		if r != 0 {
			t.Fatalf("compte %d : il reste %d centimes après les virements", id, r)
		}
	}
}
