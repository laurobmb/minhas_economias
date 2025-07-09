package models

// UserProfile armazena informações pessoais adicionais do usuário.
type UserProfile struct {
	UserID        int64  `json:"user_id"`
	DateOfBirth   string `json:"date_of_birth"` // Formato YYYY-MM-DD
	Gender        string `json:"gender"`
	MaritalStatus string `json:"marital_status"`
	ChildrenCount int    `json:"children_count"`
	Country       string `json:"country"`
	State         string `json:"state"`
	City          string `json:"city"`
}
