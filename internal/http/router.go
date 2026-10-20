package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/config"
	"github.com/miage-if-gin/bons-comptes-gin/internal/db"
	"github.com/miage-if-gin/bons-comptes-gin/internal/service"
	"github.com/miage-if-gin/bons-comptes-gin/internal/web"
)

type Deps struct {
	Cfg      config.Config
	DB       *gorm.DB
	Renderer *web.Renderer
	Accounts *service.AccountService
	Auth     *service.AuthService
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.Default()

	r.Use(chargerCompte(d))

	r.Static("/static", "web/static")

	registerHealth(r, d)
	registerPages(r, d)
	registerAuth(r, d)
	registerLogin(r, d)

    prive := r.Group("/", exigerCompte())

    registerWallets(prive, d)

    return r
}

func registerPages(r *gin.Engine, d Deps) {
	r.GET("/", func(c *gin.Context) {
		d.Renderer.Render(c, http.StatusOK, "home.html", gin.H{
			"Title": "Accueil",
		})
	})
}

func registerHealth(r *gin.Engine, d Deps) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx, d.DB); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "degraded",
				"db":     "unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "ok"})
	})
}
