package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthRequired é um middleware que verifica se o usuário está logado.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := store.Get(c.Request, "session_token")
		userID, userID_ok := session.Values["user_id"].(int64)

		// ADICIONADO: Verificação segura para o email do usuário na sessão
		userEmail, userEmail_ok := session.Values["user_email"].(string)

		// ALTERADO: A condição agora verifica o ID e o e-mail.
		// Se qualquer um dos dois falhar, a sessão é considerada inválida.
		if !userID_ok || !userEmail_ok || userID == 0 || userEmail == "" {
			// Invalida a sessão antiga/incompleta e redireciona para o login
			session.Options.MaxAge = -1
			session.Save(c.Request, c.Writer)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// A partir daqui, temos certeza de que userEmail é uma string válida
		user, err := GetUserByEmail(userEmail)
		if err != nil {
			// Se o usuário não for encontrado no DB (pode ter sido deletado), desloga
			session.Options.MaxAge = -1
			session.Save(c.Request, c.Writer)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Armazena o objeto User e o ID no contexto para uso nos handlers
		c.Set("user", user)
		c.Set("userID", user.ID)

		c.Next()
	}
}