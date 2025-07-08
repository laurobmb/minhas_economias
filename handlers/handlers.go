package handlers

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// renderErrorPage é uma função auxiliar para exibir uma página de erro padronizada.
func renderErrorPage(c *gin.Context, statusCode int, errorMessage string, originalError error) {
	if originalError != nil {
		log.Printf("ERRO (Status %d) para %s: %v", statusCode, c.Request.RequestURI, originalError)
	} else {
		log.Printf("ERRO (Status %d) para %s: %s", statusCode, c.Request.RequestURI, errorMessage)
	}

	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(statusCode, gin.H{"error": errorMessage})
	} else {
		c.HTML(statusCode, "error.html", gin.H{
			"Titulo":       "Ocorreu um Erro",
			"StatusCode":   statusCode,
			"ErrorMessage": errorMessage,
		})
	}
	c.Abort()
}

// getEnv retorna o valor de uma variável de ambiente ou um valor padrão.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}