package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// --- Métricas de Negócio (Exportadas para usar nos Handlers) ---

// Contador de Movimentações (Receitas/Despesas) criadas
var TransactionsCreated = promauto.NewCounter(prometheus.CounterOpts{
	Name: "minhas_economias_transactions_created_total",
	Help: "Total de movimentações financeiras cadastradas",
})

// Contador de Investimentos criados (com label de tipo: nacional/internacional)
var InvestmentsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "minhas_economias_investments_created_total",
	Help: "Total de ativos de investimento cadastrados",
}, []string{"scope"}) // scope = "nacional" ou "internacional"

// Contador de Relatórios Gerados
var ReportsGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "minhas_economias_reports_generated_total",
	Help: "Total de relatórios gerados/baixados",
}, []string{"type"}) // type = "pdf" ou "csv"

// Contador de Consultas à IA (Gemini)
var AiConsultationsTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "minhas_economias_ai_consultations_total",
	Help: "Total de perguntas enviadas para a análise de IA",
})

// --- Métricas HTTP Padrão (Internas do Middleware) ---

var httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "http_request_duration_seconds",
	Help:    "Duração das requisições HTTP em segundos",
	Buckets: prometheus.DefBuckets,
}, []string{"path", "method", "status"})

var httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "Total de requisições HTTP recebidas",
}, []string{"path", "method", "status"})

// PrometheusMiddleware intercepta todas as requisições para gerar métricas de latência e contagem
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Executa a requisição
		c.Next()

		// Coleta dados após a execução
		path := c.FullPath()
		if path == "" {
			path = "not_found" // Evita alta cardinalidade em 404s aleatórios
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		duration := time.Since(start).Seconds()

		// Registra nos coletores do Prometheus
		httpRequestsTotal.WithLabelValues(path, method, status).Inc()
		httpDuration.WithLabelValues(path, method, status).Observe(duration)
	}
}