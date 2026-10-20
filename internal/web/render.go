// Package web gère le chargement et le rendu des templates HTML.
package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Renderer détient une template compilée par page.
//
// html/template partage un espace de noms unique : si toutes les pages étaient
// chargées dans la même instance, le dernier {{ define "content" }} écraserait
// tous les précédents. On compile donc layout + une seule page à la fois.
type Renderer struct {
	pages map[string]*template.Template
}

// NewRenderer compile chaque page du dossier avec le layout de base.
func NewRenderer(dir string) (*Renderer, error) {
	layout := filepath.Join(dir, "layouts", "base.html")

	paths, err := filepath.Glob(filepath.Join(dir, "pages", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("lecture des pages : %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("aucune page trouvée dans %s/pages", dir)
	}

	pages := make(map[string]*template.Template, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		tmpl, err := template.ParseFiles(layout, path)
		if err != nil {
			return nil, fmt.Errorf("compilation de %s : %w", name, err)
		}
		pages[name] = tmpl
	}

	return &Renderer{pages: pages}, nil
}

// Render écrit la page demandée dans la réponse.
func (r *Renderer) Render(c *gin.Context, status int, page string, data gin.H) {
	tmpl, ok := r.pages[page]
	if !ok {
		c.String(http.StatusInternalServerError, "template inconnue : %s", page)
		return
	}

	if data == nil {
		data = gin.H{}
	}
	// Valeurs attendues par le layout, pour éviter les champs manquants.
	if _, ok := data["Title"]; !ok {
		data["Title"] = ""
	}

	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(c.Writer, "base", data); err != nil {
		c.String(http.StatusInternalServerError, "rendu de %s : %v", page, err)
	}
}
