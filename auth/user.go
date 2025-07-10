package auth

import (
	"database/sql"
	"fmt"
	"minhas_economias/database"
	"minhas_economias/models"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// CreateUser insere um novo usuário comum no banco de dados.
func CreateUser(email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}

	// A query agora é a mesma para ambos os bancos de dados, pois não especificamos o ID.
	// O banco de dados irá gerar o ID automaticamente, graças ao BIGSERIAL (Postgres) ou AUTOINCREMENT (SQLite).
	query := "INSERT INTO users (email, password_hash, is_admin) VALUES (?, ?, ?)"
	reboundQuery := database.Rebind(query)

	// Executa a query com os parâmetros corretos.
	_, err = database.GetDB().Exec(reboundQuery, email, string(hash), false)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("o e-mail '%s' já está em uso", email)
		}
		return fmt.Errorf("erro ao inserir usuário no banco de dados: %w", err)
	}

	return nil
}

// GetUserByEmail busca um usuário pelo seu e-mail, incluindo o status de admin.
func GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	// ATUALIZADO para selecionar o campo dark_mode_enabled
	query := "SELECT id, email, password_hash, is_admin, dark_mode_enabled FROM users WHERE email = ?"
	row := database.GetDB().QueryRow(database.Rebind(query), email)

	// ATUALIZADO para escanear o campo dark_mode_enabled
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.IsAdmin, &user.DarkModeEnabled)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("usuário não encontrado")
		}
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	return &user, nil
}


// CheckPasswordHash (sem alterações)
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
