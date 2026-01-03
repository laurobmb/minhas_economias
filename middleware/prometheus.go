package middleware

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// --- Métricas de Negócio ---

	TransactionsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "minhas_economias_transactions_created_total",
		Help: "Total de movimentações financeiras cadastradas",
	})

	InvestmentsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "minhas_economias_investments_created_total",
		Help: "Total de ativos de investimento cadastrados",
	}, []string{"scope"})

	ReportsGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "minhas_economias_reports_generated_total",
		Help: "Total de relatórios gerados/baixados",
	}, []string{"type"})

	AiConsultationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "minhas_economias_ai_consultations_total",
		Help: "Total de perguntas enviadas para a análise de IA",
	})

	// --- Métricas de Resiliência e Segurança (Novas) ---

	AuthFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "minhas_economias_auth_failures_total",
		Help: "Total de falhas de login (possível brute force)",
	})

	ScrapingErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "minhas_economias_scraping_errors_total",
		Help: "Total de erros ao buscar cotações externas",
	}, []string{"provider"}) // "yahoo", "fundamentus"

	AiResponseDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "minhas_economias_ai_duration_seconds",
		Help:    "Tempo de resposta da API do Gemini",
		Buckets: []float64{0.5, 1, 2, 5, 10, 20},
	})

	DbOpenConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "minhas_economias_db_connections_active",
		Help: "Número de conexões abertas no pool do Postgres",
	})

	// --- Métricas HTTP Internas ---

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duração das requisições HTTP em segundos",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "method", "status"})

	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total de requisições HTTP recebidas",
	}, []string{"path", "method", "status"})
)

// RecordDbStats deve ser chamado via goroutine no main.go para atualizar o Gauge do DB
func RecordDbStats(db *sql.DB) {
	go func() {
		for {
			stats := db.Stats()
			DbOpenConnections.Set(float64(stats.InUse))
			time.Sleep(10 * time.Second)
		}
	}()
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "not_found"
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(path, method, status).Inc()
		httpDuration.WithLabelValues(path, method, status).Observe(duration)
	}
}