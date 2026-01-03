// middleware/audit.go
package middleware

import (
	"log"
	"minhas_economias/database"
	"minhas_economias/models"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditLog intercepta operações de escrita e salva no banco de dados
func AuditLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Só logamos métodos que alteram estado
		if c.Request.Method == "POST" || c.Request.Method == "DELETE" || c.Request.Method == "PUT" {
			start := time.Now()
			
			// Processa a requisição primeiro
			c.Next()

			// Recupera usuário do contexto (definido no AuthRequired)
			user, exists := c.Get("user")
			var userEmail string
			if exists {
				userEmail = user.(*models.User).Email
			} else {
				userEmail = "anonymous"
			}

			// Captura informações da ação
			db := database.GetDB()
			query := database.Rebind(`
				INSERT INTO audit_logs (user_email, action, path, status, latency_ms, created_at)
				VALUES (?, ?, ?, ?, ?, ?)
			`)

			duration := time.Since(start).Milliseconds()
			status := c.Writer.Status()
			
			_, err := db.Exec(query, 
				userEmail, 
				c.Request.Method, 
				c.FullPath(), 
				status, 
				duration, 
				time.Now(),
			)

			if err != nil {
				log.Printf("[AUDIT ERROR] Falha ao salvar log: %v", err)
			}
		} else {
			c.Next()
		}
	}
}