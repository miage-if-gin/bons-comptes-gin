package http

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Compteur : combien de requêtes, par route, méthode et code de retour.
	// Un compteur ne fait que croître ; c'est Prometheus qui en dérive un
	// taux par seconde à l'affichage.
	requetes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Nombre total de requêtes HTTP traitées.",
		},
		[]string{"methode", "route", "code"},
	)

	// Histogramme : la distribution des temps de réponse. Une moyenne
	// masquerait les requêtes lentes ; un histogramme permet de lire les
	// centiles (95 % des requêtes sous X ms).
	duree = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Durée des requêtes HTTP, en secondes.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"methode", "route"},
	)
)

// mesurer enregistre chaque requête traitée.
//
// On utilise c.FullPath() et non c.Request.URL.Path : le premier donne le
// motif de route ("/wallets/:id"), le second l'URL réelle ("/wallets/42").
// Avec l'URL réelle, chaque porte-monnaie créerait sa propre série de
// métriques — l'explosion de cardinalité, le piège classique de Prometheus.
func mesurer() gin.HandlerFunc {
	return func(c *gin.Context) {
		debut := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "inconnue" // 404 : aucune route ne correspond
		}

		requetes.WithLabelValues(
			c.Request.Method,
			route,
			strconv.Itoa(c.Writer.Status()),
		).Inc()

		duree.WithLabelValues(c.Request.Method, route).
			Observe(time.Since(debut).Seconds())
	}
}

// registerMetrics expose /metrics, que Prometheus interroge périodiquement.
func registerMetrics(r *gin.Engine) {
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
