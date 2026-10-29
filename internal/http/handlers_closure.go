package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/miage-if-gin/bons-comptes-gin/internal/service"
)

// registerClosure déclare les routes du lot D donc les  soldes, clôture et
// remboursements. Toutes dans le groupe privé, derrière exigerCompte().
func registerClosure(prive *gin.RouterGroup, d Deps) {
	prive.GET("/wallets/:id/closure", func(c *gin.Context) {
		walletID, ok := lireID(c, "id")
		if !ok {
			return
		}
		afficherCloture(c, d, walletID, http.StatusOK, "")
	})

	prive.POST("/wallets/:id/closure", func(c *gin.Context) {
		walletID, ok := lireID(c, "id")
		if !ok {
			return
		}
		compte := CompteCourant(c)

		if err := d.Closure.Cloturer(c.Request.Context(), walletID, compte.ID); err != nil {
			if estRefusCloture(err) {
				// On réaffiche la page avec le message plutôt qu'une page d'erreur nue.
				status := http.StatusConflict
				if errors.Is(err, service.ErrClotureReservee) {
					status = http.StatusForbidden
				}
				afficherCloture(c, d, walletID, status, err.Error())
				return
			}
			repondreErreurCloture(c, err)
			return
		}
		// Post/Redirect/Get
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/wallets/%d/closure", walletID))
	})

	prive.POST("/wallets/:id/settlements/:settlementId/repay", func(c *gin.Context) {
		walletID, ok := lireID(c, "id")
		if !ok {
			return
		}
		settlementID, ok := lireID(c, "settlementId")
		if !ok {
			return
		}
		compte := CompteCourant(c)

		if err := d.Closure.Rembourser(c.Request.Context(), walletID, settlementID, compte.ID); err != nil {
			repondreErreurCloture(c, err)
			return
		}
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/wallets/%d/closure", walletID))
	})
}

// afficherCloture charge les soldes et rend la page
func afficherCloture(c *gin.Context, d Deps, walletID int64, status int, erreur string) {
	compte := CompteCourant(c)

	donnees, err := d.Closure.Consulter(c.Request.Context(), walletID, compte.ID)
	if err != nil {
		repondreErreurCloture(c, err)
		return
	}

	d.Renderer.Render(c, status, "closure.html", gin.H{
		"Title":  "Clôture — " + donnees.Wallet.Name,
		"Page":   donnees,
		"Erreur": erreur,
	})
}

// estRefusCloture : refus « normaux », affichés dans la page.
func estRefusCloture(err error) bool {
	return errors.Is(err, service.ErrClotureReservee) ||
		errors.Is(err, service.ErrDejaClos)
}

// repondreErreurCloture traduit une erreur métier en code HTTP.
func repondreErreurCloture(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPorteMonnaieIntrouvable),
		errors.Is(err, service.ErrVirementIntrouvable):
		c.String(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrAccesRefuse),
		errors.Is(err, service.ErrClotureReservee),
		errors.Is(err, service.ErrRemboursementInterdit):
		c.String(http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrDejaClos),
		errors.Is(err, service.ErrPasEncoreClos),
		errors.Is(err, service.ErrDejaRembourse):
		c.String(http.StatusConflict, err.Error())
	default:
		//  journalisée par Gin, détail caché au visiteur.
		_ = c.Error(err)
		c.String(http.StatusInternalServerError, "une erreur est survenue, réessayez")
	}
}
