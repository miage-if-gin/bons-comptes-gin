package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/miage-if-gin/bons-comptes-gin/internal/service"
)

// registerExpenses déclare les routes du lot C. Elles sont toutes dans le
// groupe privé : exigerCompte() a déjà vérifié que le visiteur est connecté.
func registerExpenses(prive *gin.RouterGroup, d Deps) {
	prive.GET("/wallets/:id/expenses", func(c *gin.Context) {
		walletID, ok := lireID(c, "id")
		if !ok {
			return
		}
		compte := CompteCourant(c)
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

		// Par défaut, le formulaire propose le visiteur comme payeur.
		saisie := service.SaisieDepense{PayeurID: strconv.FormatInt(compte.ID, 10)}
		afficherDepenses(c, d, walletID, page, http.StatusOK, "", saisie)
	})

	prive.POST("/wallets/:id/expenses", func(c *gin.Context) {
		walletID, ok := lireID(c, "id")
		if !ok {
			return
		}
		compte := CompteCourant(c)

		saisie := service.SaisieDepense{
			Libelle:  c.PostForm("label"),
			Montant:  c.PostForm("amount"),
			Date:     c.PostForm("spent_at"),
			PayeurID: c.PostForm("payer_id"),
		}

		_, err := d.Expenses.Declarer(c.Request.Context(), walletID, compte.ID, saisie)
		if err != nil {
			if estErreurDeSaisieDepense(err) {
				// On réaffiche la page avec le message et les valeurs tapées.
				afficherDepenses(c, d, walletID, 1, http.StatusUnprocessableEntity, err.Error(), saisie)
				return
			}
			repondreErreurDepense(c, err)
			return
		}

		// Post/Redirect/Get : après un POST réussi, on redirige. Recharger la
		// page ne renverra donc pas le formulaire une seconde fois.
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/wallets/%d/expenses", walletID))
	})

	// Un formulaire HTML ne sait envoyer que GET ou POST : la suppression
	// passe donc par un POST sur une URL dédiée.
	prive.POST("/wallets/:id/expenses/:expenseId/delete", func(c *gin.Context) {
		walletID, ok := lireID(c, "id")
		if !ok {
			return
		}
		expenseID, ok := lireID(c, "expenseId")
		if !ok {
			return
		}
		compte := CompteCourant(c)

		if err := d.Expenses.Supprimer(c.Request.Context(), walletID, expenseID, compte.ID); err != nil {
			repondreErreurDepense(c, err)
			return
		}
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/wallets/%d/expenses", walletID))
	})
}

// afficherDepenses charge la page des dépenses et la rend.
func afficherDepenses(c *gin.Context, d Deps, walletID int64, page, status int, erreur string, saisie service.SaisieDepense) {
	compte := CompteCourant(c)

	donnees, err := d.Expenses.Lister(c.Request.Context(), walletID, compte.ID, page)
	if err != nil {
		repondreErreurDepense(c, err)
		return
	}

	d.Renderer.Render(c, status, "expenses_index.html", gin.H{
		"Title":      "Dépenses — " + donnees.Wallet.Name,
		"Page":       donnees,
		"Erreur":     erreur,
		"Saisie":     saisie,
		"Aujourdhui": time.Now().Format("2006-01-02"),
	})
}

// lireID lit un paramètre numérique de l'URL. En cas d'échec, la réponse
// 400 est déjà envoyée et le handler doit s'arrêter.
func lireID(c *gin.Context, nom string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(nom), 10, 64)
	if err != nil || id <= 0 {
		c.String(http.StatusBadRequest, "identifiant invalide")
		return 0, false
	}
	return id, true
}

func estErreurDeSaisieDepense(err error) bool {
	return errors.Is(err, service.ErrLibelleVide) ||
		errors.Is(err, service.ErrLibelleTropLong) ||
		errors.Is(err, service.ErrMontantInvalide) ||
		errors.Is(err, service.ErrMontantTropEleve) ||
		errors.Is(err, service.ErrDateInvalide) ||
		errors.Is(err, service.ErrPayeurInvalide)
}

// repondreErreurDepense traduit une erreur métier en code HTTP.
func repondreErreurDepense(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPorteMonnaieIntrouvable),
		errors.Is(err, service.ErrDepenseIntrouvable):
		c.String(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrAccesRefuse),
		errors.Is(err, service.ErrSuppressionInterdite):
		c.String(http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrPorteMonnaieClos):
		c.String(http.StatusConflict, err.Error())
	default:
		// Erreur technique (base injoignable...) : on la journalise via Gin
		// mais on n'en montre pas le détail au visiteur.
		_ = c.Error(err)
		c.String(http.StatusInternalServerError, "une erreur est survenue, réessayez")
	}
}
