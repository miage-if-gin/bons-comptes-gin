// Package web gère le chargement et le rendu des templates HTML.
package web

import "embed"

// Templates et fichiers statiques sont embarqués dans le binaire, comme les
// migrations. Sans cela, l'application ne démarre que si le répertoire
// courant est la racine du projet : impossible à déployer dans un conteneur.
//
//go:embed templates/layouts/*.html templates/pages/*.html
var Templates embed.FS

//go:embedstatic
var Static embed.FS
