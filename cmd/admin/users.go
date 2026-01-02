package main

import (
	"database/sql"
	"log"
	"minhas_economias/database"
	"strings"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func createNewUser(db *sql.DB, email, password string, isAdmin bool, specificID int64) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Erro ao gerar hash: %v", err)
	}

	if specificID > 0 {
		upsertUserWithID(db, specificID, email, string(hash), isAdmin)
	} else {
		insertUserAutoID(db, email, string(hash), isAdmin)
	}
}

func upsertUserWithID(db *sql.DB, id int64, email, hash string, isAdmin bool) {
	var query string
	if database.DriverName == "postgres" {
		query = `INSERT INTO users (id, email, password_hash, is_admin) VALUES ($1, $2, $3, $4) 
		ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, password_hash = EXCLUDED.password_hash, is_admin = EXCLUDED.is_admin;`
	} else {
		query = `INSERT OR REPLACE INTO users (id, email, password_hash, is_admin) VALUES (?, ?, ?, ?);`
	}

	_, err := db.Exec(query, id, email, hash, isAdmin)
	if err != nil {
		log.Fatalf("Erro ao criar/atualizar usuário ID %d: %v", id, err)
	}
	log.Printf("Usuário ID %d (%s) configurado com sucesso.", id, email)
}

func insertUserAutoID(db *sql.DB, email, hash string, isAdmin bool) {
	var err error
	var newID int64

	if database.DriverName == "postgres" {
		query := "INSERT INTO users (email, password_hash, is_admin) VALUES ($1, $2, $3) RETURNING id"
		err = db.QueryRow(query, email, hash, isAdmin).Scan(&newID)
	} else {
		query := "INSERT INTO users (email, password_hash, is_admin) VALUES (?, ?, ?)"
		res, execErr := db.Exec(query, email, hash, isAdmin)
		err = execErr
		if err == nil {
			newID, _ = res.LastInsertId()
		}
	}

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			log.Fatalf("ERRO: O e-mail '%s' já existe.", email)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Fatalf("ERRO: O e-mail '%s' já existe.", email)
		}
		log.Fatalf("Erro ao criar usuário: %v", err)
	}
	log.Printf("Novo usuário criado: %s (ID: %d, Admin: %t)", email, newID, isAdmin)
}