package handlers

import (
	"minhas_economias/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LivenessProbe: O OpenShift chama isso para saber se o container está rodando.
// Se falhar (ex: deadlock), o OpenShift mata e reinicia o pod.
// Rota: /healthz
func LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"system": "minhas-economias",
	})
}

// ReadinessProbe: O OpenShift chama isso para saber se o pod pode receber tráfego.
// Verifica se o Banco de Dados está acessível. Se falhar, o pod sai do Load Balancer.
// Rota: /readyz
func ReadinessProbe(c *gin.Context) {
	db := database.GetDB()
	
	// Verifica se a conexão existe
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "conexao com o banco é nula",
		})
		return
	}

	// Tenta pingar o banco real para garantir que está respondendo
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "banco de dados inacessivel: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}