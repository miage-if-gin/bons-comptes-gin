package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrMontantInvalide  = errors.New("montant invalide : saisir un nombre positif, par exemple 12,50")
	ErrMontantTropEleve = errors.New("montant trop élevé : 1 000 000 € maximum")
)

// MontantMaxCentimes plafonne une dépense à un million d'euros.
// Sans plafond, une saisie absurde pourrait faire déborder les sommes.
const MontantMaxCentimes int64 = 100_000_000

// ParseMontant convertit une saisie en euros (« 12,50 », « 12.5 », « 12 »)
// en centimes (1250).
//
// On découpe la chaîne à la main au lieu d'utiliser strconv.ParseFloat :
// un float64 ne représente pas exactement 0,1 ou 0,2, et l'arrondi
// introduirait des erreurs d'un centime (ADR-004).
func ParseMontant(saisie string) (int64, error) {
	s := strings.TrimSpace(saisie)
	s = strings.TrimSpace(strings.TrimSuffix(s, "€"))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.Replace(s, ",", ".", 1)

	euros, centimes, aDesCentimes := strings.Cut(s, ".")

	if !queDesChiffres(euros) || len(euros) > 9 {
		return 0, ErrMontantInvalide
	}
	if aDesCentimes {
		if !queDesChiffres(centimes) || len(centimes) > 2 {
			return 0, ErrMontantInvalide
		}
		if len(centimes) == 1 {
			centimes += "0" // « 12,5 » signifie 12,50
		}
	} else {
		centimes = "00"
	}

	e, err := strconv.ParseInt(euros, 10, 64)
	if err != nil {
		return 0, ErrMontantInvalide
	}
	c, err := strconv.ParseInt(centimes, 10, 64)
	if err != nil {
		return 0, ErrMontantInvalide
	}

	total := e*100 + c
	if total <= 0 {
		return 0, ErrMontantInvalide
	}
	if total > MontantMaxCentimes {
		return 0, ErrMontantTropEleve
	}
	return total, nil
}

// FormatMontant affiche des centimes en euros : 1250 devient « 12,50 € ».
func FormatMontant(centimes int64) string {
	signe := ""
	if centimes < 0 {
		signe = "-"
		centimes = -centimes
	}
	return fmt.Sprintf("%s%d,%02d €", signe, centimes/100, centimes%100)
}

// queDesChiffres vérifie qu'une chaîne non vide ne contient que des chiffres 0-9.
func queDesChiffres(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
