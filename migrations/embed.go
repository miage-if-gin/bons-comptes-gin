// Package migrations embarque les fichiers SQL dans le binaire.
//
// Conséquence : le binaire compilé contient son propre schéma. Aucun outil
// externe à installer sur les machines de l'équipe ni en production, et
// impossible de déployer un binaire avec les mauvaises migrations.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS