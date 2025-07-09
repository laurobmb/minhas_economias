package handlers

import (
	"database/sql"
	"minhas_economias/auth"
	"minhas_economias/database"
	"minhas_economias/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetUserProfileByUserID busca o perfil de um usuário no banco de dados.
func GetUserProfileByUserID(userID int64) (*models.UserProfile, error) {
	profile := &models.UserProfile{UserID: userID}
	db := database.GetDB()
	query := database.Rebind(`SELECT date_of_birth, gender, marital_status, children_count, country, state, city FROM user_profiles WHERE user_id = ?`)

	var dob sql.NullTime
	var gender, maritalStatus, country, state, city sql.NullString
	var childrenCount sql.NullInt64

	err := db.QueryRow(query, userID).Scan(&dob, &gender, &maritalStatus, &childrenCount, &country, &state, &city)

	if err != nil {
		if err == sql.ErrNoRows {
			return profile, nil
		}
		return nil, err
	}

	if dob.Valid { profile.DateOfBirth = dob.Time.Format("2006-01-02") }
	if gender.Valid { profile.Gender = gender.String }
	if maritalStatus.Valid { profile.MaritalStatus = maritalStatus.String }
	if childrenCount.Valid { profile.ChildrenCount = int(childrenCount.Int64) }
	if country.Valid { profile.Country = country.String }
	if state.Valid { profile.State = state.String }
	if city.Valid { profile.City = city.String }

	return profile, nil
}

// UpdateUserProfile atualiza ou cria o perfil de um usuário.
func UpdateUserProfile(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var profile models.UserProfile

	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	db := database.GetDB()
	var query string

	if database.DriverName == "postgres" {
		query = `
			INSERT INTO user_profiles (user_id, date_of_birth, gender, marital_status, children_count, country, state, city)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (user_id) DO UPDATE SET
				date_of_birth = EXCLUDED.date_of_birth,
				gender = EXCLUDED.gender,
				marital_status = EXCLUDED.marital_status,
				children_count = EXCLUDED.children_count,
				country = EXCLUDED.country,
				state = EXCLUDED.state,
				city = EXCLUDED.city;
		`
	} else {
		query = `
			INSERT OR REPLACE INTO user_profiles (user_id, date_of_birth, gender, marital_status, children_count, country, state, city)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?);
		`
	}

	var dob interface{}
	if profile.DateOfBirth != "" {
		parsedTime, err := time.Parse("2006-01-02", profile.DateOfBirth)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de data inválido. Use AAAA-MM-DD."})
			return
		}
		dob = parsedTime
	}

	_, err := db.Exec(query, userID, dob, profile.Gender, profile.MaritalStatus, profile.ChildrenCount, profile.Country, profile.State, profile.City)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao salvar o perfil do usuário.", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Perfil atualizado com sucesso!"})
}

// ChangePasswordPayload define a estrutura para a requisição de alteração de senha.
type ChangePasswordPayload struct {
	CurrentPassword    string `json:"current_password" binding:"required"`
	NewPassword        string `json:"new_password" binding:"required,min=6"`
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required"`
}

// ChangePassword processa a alteração de senha do usuário.
func ChangePassword(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	// O utilizador do contexto é usado para obter o email, mas não para a verificação de senha.
	userFromContext := c.MustGet("user").(*models.User)
	var payload ChangePasswordPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Todos os campos são obrigatórios e a nova senha deve ter no mínimo 6 caracteres."})
		return
	}

	if payload.NewPassword != payload.ConfirmNewPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A nova senha e a confirmação não correspondem."})
		return
	}

	// --- CORREÇÃO PRINCIPAL ---
	// Busca o utilizador mais recente do banco de dados para obter o hash da senha atual.
	userFromDB, err := auth.GetUserByEmail(userFromContext.Email)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao verificar o utilizador.", err)
		return
	}

	// Verifica se a senha atual está correta usando o hash do banco de dados.
	if !auth.CheckPasswordHash(payload.CurrentPassword, userFromDB.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "A senha atual está incorreta."})
		return
	}
	// --- FIM DA CORREÇÃO ---

	// Gera o hash da nova senha
	newHashedPassword, err := auth.HashPassword(payload.NewPassword)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao processar a nova senha.", err)
		return
	}

	// Atualiza a senha no banco de dados
	db := database.GetDB()
	query := database.Rebind("UPDATE users SET password_hash = ? WHERE id = ?")
	_, err = db.Exec(query, newHashedPassword, userID)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao atualizar a senha no banco de dados.", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Senha alterada com sucesso!"})
}
