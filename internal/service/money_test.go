package service

import (
	"errors"
	"testing"
)

func TestParseMontant(t *testing.T) {
	cas := []struct {
		saisie  string
		attendu int64
		erreur  error
	}{
		{"12,50", 1250, nil},
		{"12.50", 1250, nil},
		{"12,5", 1250, nil},
		{"12", 1200, nil},
		{"0,05", 5, nil},
		{"  7,30  ", 730, nil},
		{"12,50 €", 1250, nil},
		{"1 234,56", 123456, nil},
		{"1000000", MontantMaxCentimes, nil},

		{"", 0, ErrMontantInvalide},
		{"abc", 0, ErrMontantInvalide},
		{"0", 0, ErrMontantInvalide},
		{"0,00", 0, ErrMontantInvalide},
		{"-5", 0, ErrMontantInvalide},
		{"12,345", 0, ErrMontantInvalide},
		{"12,", 0, ErrMontantInvalide},
		{",50", 0, ErrMontantInvalide},
		{"1e3", 0, ErrMontantInvalide},
		{"1,2,3", 0, ErrMontantInvalide},
		{"1000000,01", 0, ErrMontantTropEleve},
		{"99999999999999999999", 0, ErrMontantInvalide},
	}

	for _, c := range cas {
		t.Run(c.saisie, func(t *testing.T) {
			obtenu, err := ParseMontant(c.saisie)
			if !errors.Is(err, c.erreur) {
				t.Fatalf("ParseMontant(%q) : erreur %v, attendue %v", c.saisie, err, c.erreur)
			}
			if obtenu != c.attendu {
				t.Fatalf("ParseMontant(%q) = %d, attendu %d", c.saisie, obtenu, c.attendu)
			}
		})
	}
}

func TestFormatMontant(t *testing.T) {
	cas := map[int64]string{
		1250:   "12,50 €",
		5:      "0,05 €",
		0:      "0,00 €",
		123456: "1234,56 €",
		-730:   "-7,30 €",
	}
	for centimes, attendu := range cas {
		if obtenu := FormatMontant(centimes); obtenu != attendu {
			t.Errorf("FormatMontant(%d) = %q, attendu %q", centimes, obtenu, attendu)
		}
	}
}

// Aller-retour : formater puis relire doit redonner le même nombre de centimes.
func TestMontantAllerRetour(t *testing.T) {
	for _, centimes := range []int64{1, 10, 99, 100, 1250, 999999} {
		texte := FormatMontant(centimes)
		relu, err := ParseMontant(texte)
		if err != nil || relu != centimes {
			t.Errorf("%d → %q → %d (%v)", centimes, texte, relu, err)
		}
	}
}
