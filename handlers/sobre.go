package handlers

import (
	"minhas_economias/models" // <-- ADICIONAR IMPORT
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetSobrePage(c *gin.Context) {
	// ALTERADO: Buscar o usuário do contexto para passar a preferência de tema
	user := c.MustGet("user").(*models.User)

	c.HTML(http.StatusOK, "sobre.html", gin.H{
		"Titulo":      "Sobre o Projeto",
		"AuthorName":  getEnv("AUTHOR_NAME", "Lauro Gomes"),
		"AuthorEmail": getEnv("AUTHOR_EMAIL", "laurobmb@gmail.com"),
		"LinkedInURL": getEnv("LINKEDIN_URL", "https://www.linkedin.com/in/laurodepaula/"),
		"GitHubURL":   getEnv("GITHUB_URL", "https://github.com/laurobmb"),
		"ProjectURL":  "https://github.com/laurobmb/minhas_economias",
		"User":        user, // <-- ADICIONADO: Passa o usuário para o template
	})
}