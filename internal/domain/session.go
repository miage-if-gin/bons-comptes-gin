package domain

import "time"

// Session représente une connexion active. Le cookie du navigateur ne porte
// que le token : toutes les autres données restent côté serveur, et une
// session peut donc être révoquée en la supprimant de la base.
type Session struct {
	Token     string `gorm:"primaryKey"`
	AccountID int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (Session) TableName() string { return "sessions" }
