package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// getEnv retorna o valor de uma variável de ambiente ou um valor padrão.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// GetSobrePage renderiza a página "Sobre".
func GetSobrePage(c *gin.Context) {
	c.HTML(http.StatusOK, "sobre.html", gin.H{
		"Titulo":      "Sobre o Projeto",
		"AuthorName":  getEnv("AUTHOR_NAME", "Lauro Gomes"),
		"AuthorEmail": getEnv("AUTHOR_EMAIL", "laurobmb@gmail.com"),
		"LinkedInURL": getEnv("LINKEDIN_URL", "https://www.linkedin.com/in/laurodepaula/"),
		"GitHubURL":   getEnv("GITHUB_URL", "https://github.com/laurobmb"),
		"ProjectURL":  "https://github.com/laurobmb/minhas_economias",
	})
}