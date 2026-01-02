package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"minhas_economias/database"
	"strings"

	"github.com/lib/pq" // Importação direta para checagem de erro
	"golang.org/x/crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// --- FLAGS ATUALIZADAS PARA MAIOR FLEXIBILIDADE ---
	email := flag.String("email", "", "E-mail do usuário. (Obrigatório)")
	password := flag.String("password", "", "Senha do usuário. (Obrigatório)")
	id := flag.Int64("id", 0, "ID específico para o usuário. Se 0, será gerado automaticamente. (Opcional)")
	isAdmin := flag.Bool("admin", false, "Define se o usuário é um administrador. (Opcional, padrão: false)")
	flag.Parse()

	if *email == "" || *password == "" {
		log.Fatal("ERRO: As flags -email e -password são obrigatórias.")
	}

	// Conecta ao banco de dados
	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	db := database.GetDB()
	defer database.CloseDB()

	// Gera o hash da senha
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Erro ao gerar hash da senha: %v", err)
	}

	// --- LÓGICA CONDICIONAL PARA CRIAR/ATUALIZAR USUÁRIOS ---
	if *id > 0 {
		// Se um ID foi fornecido, tentamos inserir ou atualizar esse usuário específico.
		upsertUserWithID(db, *id, *email, string(hash), *isAdmin)
	} else {
		// Se nenhum ID foi fornecido, criamos um novo usuário com ID automático.
		createNewUser(db, *email, string(hash), *isAdmin)
	}
}

// upsertUserWithID insere um usuário com um ID específico ou atualiza se já existir.
func upsertUserWithID(db *sql.DB, id int64, email, hash string, isAdmin bool) {
	var query string
	if database.DriverName == "postgres" {
		query = `
            INSERT INTO users (id, email, password_hash, is_admin)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (id) DO UPDATE
            SET email = EXCLUDED.email, password_hash = EXCLUDED.password_hash, is_admin = EXCLUDED.is_admin;
        `
	} else { // sqlite3
		query = `INSERT OR REPLACE INTO users (id, email, password_hash, is_admin) VALUES (?, ?, ?, ?);`
	}

	_, err := db.Exec(query, id, email, hash, isAdmin)
	if err != nil {
		log.Fatalf("Erro ao inserir/atualizar o usuário com ID %d: %v", id, err)
	}
	fmt.Printf("✔ Usuário com ID %d criado/atualizado com sucesso para o e-mail: %s (Admin: %t)\n", id, email, isAdmin)
}

// createNewUser cria um novo usuário com ID gerado automaticamente.
func createNewUser(db *sql.DB, email, hash string, isAdmin bool) {
	var err error
	var newID int64

	if database.DriverName == "postgres" {
		query := "INSERT INTO users (email, password_hash, is_admin) VALUES ($1, $2, $3) RETURNING id"
		err = db.QueryRow(query, email, hash, isAdmin).Scan(&newID)
	} else { // sqlite3
		query := "INSERT INTO users (email, password_hash, is_admin) VALUES (?, ?, ?)"
		result, execErr := db.Exec(query, email, hash, isAdmin)
		err = execErr
		if err == nil {
			newID, _ = result.LastInsertId()
		}
	}

	if err != nil {
		// **MELHORIA**: Checagem de erro mais específica para PostgreSQL
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" {
				// Verifica se a violação foi na chave primária (ID) ou no e-mail
				if pqErr.Constraint == "users_pkey" {
					log.Fatalf("ERRO: Conflito de ID no banco de dados. A sequência de IDs pode estar dessincronizada.\nSUGESTÃO: Rode o comando SQL -> SELECT setval(pg_get_serial_sequence('users', 'id'), (SELECT MAX(id) FROM users));")
				}
				// Se não for a chave primária, assume que é o e-mail
				log.Fatalf("ERRO: O e-mail '%s' já está em uso.", email)
			}
		}

		// Checagem genérica para SQLite
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
			log.Fatalf("ERRO: O e-mail '%s' já está em uso.", email)
		}

		// Se for outro tipo de erro, exibe a mensagem genérica
		log.Fatalf("Erro ao criar novo usuário: %v", err)
	}

	fmt.Printf("✔ Novo usuário criado com sucesso para o e-mail: %s (ID: %d, Admin: %t)\n", email, newID, isAdmin)
}
