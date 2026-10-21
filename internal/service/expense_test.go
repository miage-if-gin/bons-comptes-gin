package service

import "testing"

func TestPeutSupprimer(t *testing.T) {
	alice, bob, chloe := int64(1), int64(2), int64(3)

	cas := []struct {
		nom      string
		payeur   *int64
		createur int64
		visiteur int64
		attendu  bool
	}{
		{"le payeur peut supprimer", &bob, alice, bob, true},
		{"le créateur ne peut pas si un payeur existe", &bob, alice, alice, false},
		{"un tiers ne peut pas", &bob, alice, chloe, false},
		{"sans payeur, le créateur peut", nil, alice, alice, true},
		{"sans payeur, un tiers ne peut pas", nil, alice, bob, false},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if obtenu := PeutSupprimer(c.payeur, c.createur, c.visiteur); obtenu != c.attendu {
				t.Fatalf("obtenu %v, attendu %v", obtenu, c.attendu)
			}
		})
	}
}
