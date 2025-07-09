package handlers

import (
	"minhas_economias/database"
	"minhas_economias/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetConfiguracoesPage renderiza a página de configurações do usuário.
func GetConfiguracoesPage(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	// Busca o perfil do usuário
	userProfile, err := GetUserProfileByUserID(user.ID)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Não foi possível carregar o perfil do usuário.", err)
		return
	}

	c.HTML(http.StatusOK, "configuracoes.html", gin.H{
		"Titulo":      "Configurações",
		"User":        user,
		"UserProfile": userProfile, // Passa o perfil para o template
	})
}

// UpdateUserSettingsPayload é o struct para o corpo da requisição de atualização do tema.
type UpdateUserSettingsPayload struct {
	DarkMode bool `json:"dark_mode"`
}

// UpdateUserSettings atualiza as configurações de tema do usuário.
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

	c.JSON(http.StatusOK, gin.H{"message": "Configurações atualizadas com sucesso"})
}
