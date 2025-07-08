package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthRequired é um middleware que verifica se o usuário está logado.
// Se não estiver, redireciona para a página de login.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := store.Get(c.Request, "session_token")
		userID, ok := session.Values["user_id"].(int64)

		if !ok || userID == 0 {
			// Redireciona para a página de login se não houver user_id na sessão
			c.Redirect(http.StatusFound, "/login")
			c.Abort() // Impede que os próximos handlers sejam chamados
			return
		}

		// Coloca o ID do usuário no contexto do Gin para uso posterior nos handlers
		c.Set("userID", userID)
		c.Next() // Continua para o próximo handler na cadeia
	}
}