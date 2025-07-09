package handlers

import (
	"minhas_economias/auth"
	"minhas_economias/database"
	"minhas_economias/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetConfiguracoesPage renderiza a página de configurações do usuário.
func GetConfiguracoesPage(c *gin.Context) {
	// O objeto de usuário já está no contexto graças ao middleware
	user := c.MustGet("user").(*models.User)

	c.HTML(http.StatusOK, "configuracoes.html", gin.H{
		"Titulo": "Configurações",
		"User":   user, // Passa o objeto de usuário inteiro para o template
	})
}

// UpdateUserSettingsPayload é o struct para o corpo da requisição de atualização.
type UpdateUserSettingsPayload struct {
	DarkMode bool `json:"dark_mode"`
}

// UpdateUserSettings atualiza as configurações do usuário.
func UpdateUserSettings(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var payload UpdateUserSettingsPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
		return
	}

	query := "UPDATE users SET dark_mode_enabled = ? WHERE id = ?"
	reboundQuery := database.Rebind(query)
	db := database.GetDB()

	_, err := db.Exec(reboundQuery, payload.DarkMode, userID)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao atualizar as configurações.", err)
		return
	}

	// Atualiza o usuário na sessão para refletir a mudança imediatamente
	user, err := auth.GetUserByEmail(c.MustGet("user").(*models.User).Email)
	if err == nil {
		c.Set("user", user)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configurações atualizadas com sucesso"})
}