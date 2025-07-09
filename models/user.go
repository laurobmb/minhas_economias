package models

// User representa um usuário no sistema.
type User struct {
	ID              int64
	Email           string
	PasswordHash    string
	IsAdmin         bool
	DarkModeEnabled bool // <-- NOVO CAMPO ADICIONADO
}