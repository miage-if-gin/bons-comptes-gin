package http

import (
	"errors"
	"net/http"
	"net/url"

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
		c.Redirect(http.StatusSeeOther, "/login")
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

// NomCookieSession est le nom du cookie qui porte le token de session.
const NomCookieSession = "session"

func registerLogin(r *gin.Engine, d Deps) {
	r.GET("/login", func(c *gin.Context) {
		d.Renderer.Render(c, http.StatusOK, "login.html", gin.H{
			"Title": "Connexion",
		})
	})

	r.POST("/login", func(c *gin.Context) {
		email := c.PostForm("email")
		password := c.PostForm("password")

		session, err := d.Auth.Login(c.Request.Context(), email, password)
		if err != nil {
			status := http.StatusUnauthorized
			message := service.ErrIdentifiantsInvalides.Error()
			if !errors.Is(err, service.ErrIdentifiantsInvalides) {
				status = http.StatusInternalServerError
				message = "une erreur est survenue, réessayez"
			}
			d.Renderer.Render(c, status, "login.html", gin.H{
				"Title":  "Connexion",
				"Erreur": message,
				"Email":  email,
			})
			return
		}

		poserCookieSession(c, d, session.Token)
		c.Redirect(http.StatusSeeOther, "/")
	})

	// La déconnexion est en POST, pas en GET : un lien <a href="/logout">
	// serait déclenché par n'importe quelle image ou préchargement de page,
	// et déconnecterait l'utilisateur à son insu.
	r.POST("/logout", func(c *gin.Context) {
		if token, err := c.Cookie(NomCookieSession); err == nil {
			_ = d.Auth.Logout(c.Request.Context(), token)
		}
		supprimerCookieSession(c, d)
		c.Redirect(http.StatusSeeOther, "/")
	})
}

func poserCookieSession(c *gin.Context, d Deps, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		NomCookieSession,
		url.QueryEscape(token),
		int(service.DureeSession.Seconds()),
		"/",
		"",
		d.Cfg.IsProduction(), // Secure : HTTPS uniquement, impossible en local
		true,                 // HttpOnly : inaccessible au JavaScript de la page
	)
}

func supprimerCookieSession(c *gin.Context, d Deps) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(NomCookieSession, "", -1, "/", "", d.Cfg.IsProduction(), true)
}
