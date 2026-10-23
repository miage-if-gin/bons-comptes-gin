package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

// cleCompteCourant : clé sous laquelle le compte est rangé dans le contexte Gin.
const cleCompteCourant = "compteCourant"

// chargerCompte lit le cookie et place le compte dans le contexte s'il existe.
// Ne bloque jamais : les pages publiques doivent pouvoir afficher la
// navigation adaptée sans exiger une connexion.
func chargerCompte(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(NomCookieSession)
		if err == nil && token != "" {
			account, err := d.Auth.AccountFromToken(c.Request.Context(), token)
			if err == nil && account != nil {
				c.Set(cleCompteCourant, account)
			}
		}
		c.Next()
	}
}

// exigerCompte bloque l'accès aux comptes non connectés.
func exigerCompte() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get(cleCompteCourant); !ok {
			// c.Redirect pose le Location avant l'écriture des en-têtes.
			// Avec AbortWithStatus, ils partaient immédiatement et le
			// Location ajouté ensuite était perdu : le navigateur recevait
			// un 303 sans destination.
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// CompteCourant renvoie le compte connecté, ou nil.
func CompteCourant(c *gin.Context) *domain.Account {
	v, ok := c.Get(cleCompteCourant)
	if !ok {
		return nil
	}
	a, _ := v.(*domain.Account)
	return a
}
