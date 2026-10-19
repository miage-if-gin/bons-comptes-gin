// Package db gère la connexion à PostgreSQL.
package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open ouvre le pool de connexions et vérifie que la base répond.
//
// Le dimensionnement du pool compte pour le « Think HUGE » de l'énoncé :
// sans limite, chaque requête HTTP simultanée ouvrirait sa propre connexion
// et saturerait PostgreSQL (100 connexions par défaut, bien moins sur les
// offres managées gratuites).
func Open(dsn string, verbose bool) (*gorm.DB, error) {
	level := logger.Warn
	if verbose {
		level = logger.Info // journalise chaque requête SQL, utile en TP
	}

	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(level),
	})
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base : %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("accès au pool : %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("la base ne répond pas : %w", err)
	}

	return gdb, nil
}

// Close ferme proprement le pool de connexions.
func Close(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping vérifie que la base répond. Utilisé par la sonde de disponibilité.
func Ping(ctx context.Context, gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}