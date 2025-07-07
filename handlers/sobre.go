package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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