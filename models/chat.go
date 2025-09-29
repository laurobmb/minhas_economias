package models

import "time"

// ChatMessage representa uma única mensagem no histórico do chat.
type ChatMessage struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Role      string    `json:"role"` // "user" ou "ai"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}