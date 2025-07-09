package auth

import (
	"log"
	"minhas_economias/models"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// A chave para assinar a sessão. DEVE ser secreta e, em produção, vir de uma variável de ambiente.
var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

const maxAgeSeconds = 3600 * 24 * 7 // 7 dias

// GetLoginPage renderiza a página de login.
func GetLoginPage(c *gin.Context) {
	// Pega mensagens "flash" (de sucesso ou erro) da sessão, se houver
	session, _ := store.Get(c.Request, "session_token")
	flash := session.Values["flash"]
	session.Values["flash"] = "" // Limpa a mensagem após lê-la
	session.Save(c.Request, c.Writer)

	c.HTML(http.StatusOK, "login.html", gin.H{
		"Titulo": "Login",
		"Flash":  flash,
	})
}

// PostLogin processa o formulário de login.
func PostLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	user, err := GetUserByEmail(email)
	if err != nil || !CheckPasswordHash(password, user.PasswordHash) {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Titulo": "Login",
			"Error":  "E-mail ou senha inválidos.",
		})
		return
	}

	session, _ := store.Get(c.Request, "session_token")
	session.Values["user_id"] = user.ID
	session.Values["user_email"] = user.Email // <-- ALTERADO: Adicionado para o middleware buscar o usuário completo
	session.Options.MaxAge = maxAgeSeconds    // Define o tempo de expiração do cookie
	session.Options.HttpOnly = true           // Medida de segurança
	session.Options.SameSite = http.SameSiteLaxMode

	err = session.Save(c.Request, c.Writer)
	if err != nil {
		log.Printf("Erro ao salvar a sessão: %v", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"Titulo": "Login",
			"Error":  "Não foi possível iniciar a sessão.",
		})
		return
	}

	c.Redirect(http.StatusFound, "/")
}

// GetRegisterPage renderiza a página de registro.
func GetRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"Titulo": "Criar Conta",
	})
}

// PostRegister processa o formulário de registro.
func PostRegister(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	if email == "" || password == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Titulo": "Criar Conta",
			"Error":  "E-mail e senha são obrigatórios.",
		})
		return
	}

	if len(password) < 6 {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Titulo": "Criar Conta",
			"Error":  "A senha deve ter pelo menos 6 caracteres.",
		})
		return
	}

	err := CreateUser(email, password)
	if err != nil {
		c.HTML(http.StatusConflict, "register.html", gin.H{
			"Titulo": "Criar Conta",
			"Error":  err.Error(),
		})
		return
	}

	// Salva uma mensagem de sucesso na sessão para exibir na página de login
	session, _ := store.Get(c.Request, "session_token")
	session.Values["flash"] = "Conta criada com sucesso! Por favor, faça o login."
	session.Save(c.Request, c.Writer)

	c.Redirect(http.StatusFound, "/login")
}

// PostLogout encerra a sessão do usuário.
func PostLogout(c *gin.Context) {
	session, _ := store.Get(c.Request, "session_token")
	// Apaga os dados da sessão
	session.Values["user_id"] = nil
	session.Values["user_email"] = nil // <-- Limpa também o email
	session.Options.MaxAge = -1        // Expira o cookie imediatamente
	session.Save(c.Request, c.Writer)
	c.Redirect(http.StatusFound, "/login")
}

// GetUserFromContext recupera os dados do usuário a partir do ID no contexto.
func GetUserFromContext(c *gin.Context) *models.User {
	// Embora tenhamos o objeto "user" completo no contexto agora,
	// esta função pode ser mantida para compatibilidade ou outros usos.
	userID, exists := c.Get("userID")
	if !exists {
		return nil
	}

	// Para uma implementação mais robusta, você buscaria no banco de dados aqui.
	// Por simplicidade, retornamos o que está no contexto.
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}

	// Fallback para o caso de apenas o ID estar disponível.
	return &models.User{
		ID: userID.(int64),
	}
}