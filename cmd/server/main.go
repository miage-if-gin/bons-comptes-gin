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
	apphttp "github.com/miage-if-gin/bons-comptes-gin/internal/http"
	"github.com/miage-if-gin/bons-comptes-gin/internal/repository"
	"github.com/miage-if-gin/bons-comptes-gin/internal/service"
	"github.com/miage-if-gin/bons-comptes-gin/internal/web"
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

	if err := db.Migrate(gdb); err != nil {
		slog.Error("migrations en échec", "error", err)
		os.Exit(1)
	}

	renderer, err := web.NewRenderer("web/templates")
	if err != nil {
		slog.Error("chargement des templates", "error", err)
		os.Exit(1)
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	accountRepo := repository.NewAccountRepository(gdb)
	accountSvc := service.NewAccountService(accountRepo)

	r := apphttp.NewRouter(apphttp.Deps{
		Cfg:      cfg,
		DB:       gdb,
		Renderer: renderer,
		Accounts: accountSvc,
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
