package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/miage-if-gin/bons-comptes-gin/internal/service"
)

func registerAuth(r *gin.Engine, d Deps) {
	r.GET("/register", func(c *gin.Context) {
		d.Renderer.Render(c, http.StatusOK, "register.html", gin.H{
			"Title": "Créer un compte",
		})
	})

	r.POST("/register", func(c *gin.Context) {
		email := c.PostForm("email")
		password := c.PostForm("password")
		displayName := c.PostForm("display_name")

		_, err := d.Accounts.Register(c.Request.Context(), email, password, displayName)
		if err != nil {
			// On réaffiche le formulaire avec les champs saisis, sauf le mot
			// de passe : il ne repart jamais vers le navigateur.
			status := http.StatusUnprocessableEntity
			if !estErreurDeSaisie(err) {
				status = http.StatusInternalServerError
			}
			d.Renderer.Render(c, status, "register.html", gin.H{
				"Title":       "Créer un compte",
				"Erreur":      messageErreur(err),
				"Email":       email,
				"DisplayName": displayName,
			})
			return
		}
		c.Redirect(http.StatusSeeOther, "/")
	})
}

func estErreurDeSaisie(err error) bool {
	return errors.Is(err, service.ErrEmailInvalide) ||
		errors.Is(err, service.ErrMotDePasseFaible) ||
		errors.Is(err, service.ErrNomVide) ||
		errors.Is(err, service.ErrEmailPris)
}

func messageErreur(err error) string {
	if estErreurDeSaisie(err) {
		return err.Error()
	}
	return "une erreur est survenue, réessayez"
}
