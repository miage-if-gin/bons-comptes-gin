// Package config charge la configuration depuis l'environnement,
// conformément au principe III des Twelve-Factor Apps.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config regroupe tous les réglages de l'application.
// Aucune valeur n'est codée en dur ailleurs dans le code.
type Config struct {
	AppEnv        string `env:"APP_ENV" envDefault:"development"`
	Port          string `env:"PORT" envDefault:"8080"`
	DatabaseURL   string `env:"DATABASE_URL,required"`
	SessionSecret string `env:"SESSION_SECRET,required"`
}

// IsProduction indique si l'application tourne en production.
func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load lit l'environnement et renvoie une configuration validée.
// Une variable requise manquante est une erreur fatale au démarrage :
// mieux vaut un crash immédiat qu'une application à moitié fonctionnelle.
func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, fmt.Errorf("chargement de la configuration : %w", err)
	}
	return c, nil
}
