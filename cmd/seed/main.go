// Commande seed : peuple la base de données de démonstration.
//
// Utilité : le « Think HUGE » de l'énoncé ne se vérifie pas sur trois lignes.
// Avec quelques milliers de dépenses, on mesure les temps de réponse, on
// repère les requêtes lentes et on vérifie que la pagination tient.
//
//	go run ./cmd/seed
//	go run ./cmd/seed -comptes 100 -portemonnaies 50 -depenses 20000
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/config"
	"github.com/miage-if-gin/bons-comptes-gin/internal/db"
	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

// MotDePasseDemo : le même pour tous les comptes générés, pour pouvoir se
// connecter à n'importe lequel pendant une démonstration.
const MotDePasseDemo = "demo1234"

var libelles = []string{
	"Courses", "Essence", "Restaurant", "Péage", "Hôtel", "Train",
	"Cinéma", "Pharmacie", "Boulangerie", "Taxi", "Musée", "Pizza",
	"Café", "Parking", "Location vélo", "Pique-nique", "Camping",
}

var prenoms = []string{
	"Alice", "Bob", "Chloé", "David", "Emma", "Félix", "Gabrielle",
	"Hugo", "Inès", "Jules", "Kenza", "Léo", "Manon", "Noah", "Olivia",
	"Paul", "Quentin", "Rachel", "Samuel", "Théo", "Uma", "Victor",
}

func main() {
	nbComptes := flag.Int("comptes", 40, "nombre de comptes à créer")
	nbPortefeuilles := flag.Int("portemonnaies", 25, "nombre de porte-monnaies")
	nbDepenses := flag.Int("depenses", 5000, "nombre de dépenses")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration invalide", "error", err)
		os.Exit(1)
	}

	// verbose=false : sans cela, GORM journaliserait les milliers d'insertions.
	gdb, err := db.Open(cfg.DatabaseURL, false)
	if err != nil {
		slog.Error("connexion à la base impossible", "error", err)
		os.Exit(1)
	}
	defer db.Close(gdb)

	if err := db.Migrate(gdb); err != nil {
		slog.Error("migrations en échec", "error", err)
		os.Exit(1)
	}

	debut := time.Now()
	// Graine fixe : deux exécutions produisent le même jeu de données.
	r := rand.New(rand.NewPCG(42, 1024))

	comptes, err := creerComptes(gdb, *nbComptes, r)
	if err != nil {
		slog.Error("création des comptes", "error", err)
		os.Exit(1)
	}
	slog.Info("comptes créés", "n", len(comptes))

	portefeuilles, membres, err := creerPortefeuilles(gdb, comptes, *nbPortefeuilles, r)
	if err != nil {
		slog.Error("création des porte-monnaies", "error", err)
		os.Exit(1)
	}
	slog.Info("porte-monnaies créés", "n", len(portefeuilles))

	n, err := creerDepenses(gdb, portefeuilles, membres, *nbDepenses, r)
	if err != nil {
		slog.Error("création des dépenses", "error", err)
		os.Exit(1)
	}
	slog.Info("dépenses créées", "n", n)

	slog.Info("terminé",
		"duree", time.Since(debut).Round(time.Millisecond),
		"connexion", fmt.Sprintf("n'importe quel email généré / %s", MotDePasseDemo),
	)
}

func creerComptes(gdb *gorm.DB, n int, r *rand.Rand) ([]domain.Account, error) {
	// Un seul hachage bcrypt réutilisé : en calculer un par compte prendrait
	// plusieurs secondes pour rien, tous ayant le même mot de passe.
	hash, err := bcrypt.GenerateFromPassword([]byte(MotDePasseDemo), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	suffixe := time.Now().Unix()
	comptes := make([]domain.Account, 0, n)
	for i := 0; i < n; i++ {
		prenom := prenoms[i%len(prenoms)]
		comptes = append(comptes, domain.Account{
			// Minuscules et sans accent : le service d'inscription normalise
			// les emails, et le seed doit produire des données cohérentes
			// avec ce que l'application aurait créé.
			Email:        fmt.Sprintf("%s%d-%d@demo.local", strings.ToLower(sansAccents(prenom)), i, suffixe),
			PasswordHash: string(hash),
			DisplayName:  fmt.Sprintf("%s %c.", prenom, 'A'+rune(r.IntN(26))),
		})
	}

	// CreateInBatches : un INSERT pour 100 lignes plutôt que 100 INSERT.
	if err := gdb.CreateInBatches(&comptes, 100).Error; err != nil {
		return nil, err
	}
	return comptes, nil
}

func creerPortefeuilles(gdb *gorm.DB, comptes []domain.Account, n int, r *rand.Rand) ([]domain.Wallet, map[int64][]int64, error) {
	portefeuilles := make([]domain.Wallet, 0, n)
	for i := 0; i < n; i++ {
		proprietaire := comptes[r.IntN(len(comptes))]
		portefeuilles = append(portefeuilles, domain.Wallet{
			Name:    fmt.Sprintf("%s %d", []string{"Vacances", "Week-end", "Colocation", "Soirée", "Voyage"}[r.IntN(5)], i+1),
			OwnerID: proprietaire.ID,
			Status:  "open",
		})
	}
	if err := gdb.CreateInBatches(&portefeuilles, 100).Error; err != nil {
		return nil, nil, err
	}

	var membres []domain.WalletMember
	// Qui est membre accepté de quel porte-monnaie : servira aux dépenses,
	// un payeur devant être membre accepté.
	acceptes := make(map[int64][]int64, len(portefeuilles))
	maintenant := time.Now()

	for i, w := range portefeuilles {
		// Le propriétaire est toujours membre accepté.
		membres = append(membres, domain.WalletMember{
			WalletID: w.ID, AccountID: w.OwnerID,
			Status: "accepted", InvitedAt: maintenant, RespondedAt: &maintenant,
		})
		acceptes[w.ID] = []int64{w.OwnerID}

		// Entre 2 et 6 invités, choisis sans doublon.
		dejaInvites := map[int64]bool{w.OwnerID: true}
		for j := 0; j < 2+r.IntN(5); j++ {
			invite := comptes[r.IntN(len(comptes))]
			if dejaInvites[invite.ID] {
				continue
			}
			dejaInvites[invite.ID] = true

			// Un invité sur dix refuse, un sur cinq n'a pas encore répondu :
			// les trois statuts sont représentés dans le jeu de données.
			statut := "accepted"
			switch {
			case r.IntN(10) == 0:
				statut = "declined"
			case r.IntN(5) == 0:
				statut = "pending"
			}

			m := domain.WalletMember{
				WalletID: w.ID, AccountID: invite.ID,
				Status: statut, InvitedBy: &w.OwnerID, InvitedAt: maintenant,
			}
			if statut != "pending" {
				m.RespondedAt = &maintenant
			}
			membres = append(membres, m)

			if statut == "accepted" {
				acceptes[w.ID] = append(acceptes[w.ID], invite.ID)
			}
		}
		_ = i
	}

	if err := gdb.CreateInBatches(&membres, 200).Error; err != nil {
		return nil, nil, err
	}
	return portefeuilles, acceptes, nil
}

func creerDepenses(gdb *gorm.DB, portefeuilles []domain.Wallet, acceptes map[int64][]int64, n int, r *rand.Rand) (int, error) {
	depenses := make([]domain.Expense, 0, n)

	for i := 0; i < n; i++ {
		w := portefeuilles[r.IntN(len(portefeuilles))]
		membres := acceptes[w.ID]
		if len(membres) == 0 {
			continue
		}

		auteur := membres[r.IntN(len(membres))]

		// Une dépense sur six n'a pas de payeur (ADR-011) : elle compte dans
		// le total sans entrer dans le partage.
		var payeur *int64
		if r.IntN(6) > 0 {
			p := membres[r.IntN(len(membres))]
			payeur = &p
		}

		depenses = append(depenses, domain.Expense{
			WalletID:    w.ID,
			Label:       libelles[r.IntN(len(libelles))],
			AmountCents: int64(100 + r.IntN(20_000)), // de 1 € à 200 €
			PayerID:     payeur,
			CreatedBy:   auteur,
			SpentAt:     time.Now().Add(-time.Duration(r.IntN(90*24)) * time.Hour),
		})
	}

	if err := gdb.CreateInBatches(&depenses, 500).Error; err != nil {
		return 0, err
	}
	return len(depenses), nil
}

// sansAccents remplace les caractères accentués des prénoms pour construire
// des adresses email valides.
func sansAccents(s string) string {
	remplacements := strings.NewReplacer(
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"à", "a", "â", "a", "ä", "a",
		"î", "i", "ï", "i",
		"ô", "o", "ö", "o",
		"ù", "u", "û", "u", "ü", "u",
		"ç", "c",
	)
	return remplacements.Replace(s)
}

var _ = context.Background // évite un import inutilisé si le code évolue
