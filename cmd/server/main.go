package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/miage-if-gin/bons-comptes-gin/internal/config"
	"github.com/miage-if-gin/bons-comptes-gin/internal/db"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration invalide", "error", err)
		os.Exit(1)
	}

	gdb, err := db.Open(cfg.DatabaseURL, !cfg.IsProduction())
	if err != nil {
		slog.Error("connexion à la base impossible", "error", err)
		os.Exit(1)
	}
	defer db.Close(gdb)

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Sonde de vivacité : le processus répond-il ?
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Sonde de disponibilité : peut-on servir du trafic ?
	// Un orchestrateur retire l'instance du load balancer si elle échoue,
	// sans pour autant la redémarrer.
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx, gdb); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "degraded",
				"db":     "unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "ok"})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("démarrage du serveur", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("le serveur s'est arrêté", "error", err)
			os.Exit(1)
		}
	}()

	// Arrêt propre : on laisse dix secondes aux requêtes en cours.
	// Sans ça, chaque déploiement renvoie des 502 aux utilisateurs connectés
	// à cet instant.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("arrêt demandé, fermeture en cours")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("arrêt forcé", "error", err)
	}
	slog.Info("serveur arrêté")
}