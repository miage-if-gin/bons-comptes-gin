package db

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/migrations"
)

// Migrate applique les migrations en attente.
//
// golang-migrate tient une table schema_migrations dans la base et pose un
// verrou consultatif PostgreSQL le temps de l'opération. Deux instances qui
// démarrent en même temps ne peuvent donc pas jouer la même migration deux
// fois : la seconde attend, constate qu'il n'y a plus rien à faire, et
// poursuit. C'est ce qui rend l'appel au démarrage sûr, y compris quand
// plusieurs d'entre vous lancent le serveur sur la base partagée.
func Migrate(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("accès au pool : %w", err)
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("lecture des migrations embarquées : %w", err)
	}

	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("initialisation du driver de migration : %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("initialisation du migrateur : %w", err)
	}

	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("schéma déjà à jour")
		return nil
	}
	if err != nil {
		return fmt.Errorf("application des migrations : %w", err)
	}

	version, dirty, _ := m.Version()
	slog.Info("migrations appliquées", "version", version, "dirty", dirty)
	return nil
}