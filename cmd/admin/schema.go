package main

import (
	"database/sql"
	"fmt"
	"log"
	"minhas_economias/database"
)

func createTables(db *sql.DB) {
	log.Println("Verificando/Criando schema do banco de dados...")
	var createUsers, createMov, createContas, createProfile, createInvNac, createInvInt, createChat string

	if database.DriverName == "postgres" {
		createUsers = `CREATE TABLE IF NOT EXISTS users (id BIGSERIAL PRIMARY KEY, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, is_admin BOOLEAN DEFAULT FALSE, dark_mode_enabled BOOLEAN DEFAULT FALSE);`
		createMov = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY, user_id BIGINT NOT NULL, data_ocorrencia DATE NOT NULL, descricao TEXT, valor NUMERIC(10, 2), categoria TEXT, conta TEXT, consolidado BOOLEAN DEFAULT FALSE, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`, tableName)
		createContas = `CREATE TABLE IF NOT EXISTS contas (user_id BIGINT NOT NULL, nome TEXT NOT NULL, saldo_inicial NUMERIC(10, 2) NOT NULL DEFAULT 0, PRIMARY KEY (user_id, nome), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createProfile = `CREATE TABLE IF NOT EXISTS user_profiles (user_id BIGINT PRIMARY KEY, date_of_birth DATE, gender TEXT, marital_status TEXT, children_count INTEGER, country TEXT, state TEXT, city TEXT, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createInvNac = `CREATE TABLE IF NOT EXISTS investimentos_nacionais (user_id BIGINT NOT NULL, ticker TEXT NOT NULL, tipo TEXT, quantidade INTEGER NOT NULL, PRIMARY KEY (user_id, ticker), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createInvInt = `CREATE TABLE IF NOT EXISTS investimentos_internacionais (user_id BIGINT NOT NULL, ticker TEXT NOT NULL, descricao TEXT, quantidade REAL NOT NULL, moeda TEXT, PRIMARY KEY (user_id, ticker), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createChat = `CREATE TABLE IF NOT EXISTS chat_history (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL, role TEXT NOT NULL, content TEXT NOT NULL, created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
	} else {
		createUsers = `CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, is_admin BOOLEAN DEFAULT FALSE, dark_mode_enabled BOOLEAN DEFAULT 0);`
		createMov = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, data_ocorrencia TEXT NOT NULL, descricao TEXT, valor REAL, categoria TEXT, conta TEXT, consolidado BOOLEAN DEFAULT FALSE, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`, tableName)
		createContas = `CREATE TABLE IF NOT EXISTS contas (user_id INTEGER NOT NULL, nome TEXT NOT NULL, saldo_inicial REAL NOT NULL DEFAULT 0, PRIMARY KEY (user_id, nome), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createProfile = `CREATE TABLE IF NOT EXISTS user_profiles (user_id INTEGER PRIMARY KEY, date_of_birth TEXT, gender TEXT, marital_status TEXT, children_count INTEGER, country TEXT, state TEXT, city TEXT, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createInvNac = `CREATE TABLE IF NOT EXISTS investimentos_nacionais (user_id INTEGER NOT NULL, ticker TEXT NOT NULL, tipo TEXT, quantidade INTEGER NOT NULL, PRIMARY KEY (user_id, ticker), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createInvInt = `CREATE TABLE IF NOT EXISTS investimentos_internacionais (user_id INTEGER NOT NULL, ticker TEXT NOT NULL, descricao TEXT, quantidade REAL NOT NULL, moeda TEXT, PRIMARY KEY (user_id, ticker), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createChat = `CREATE TABLE IF NOT EXISTS chat_history (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, role TEXT NOT NULL, content TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
	}

	execQuery(db, createUsers, "users")
	execQuery(db, createMov, "movimentacoes")
	execQuery(db, createContas, "contas")
	execQuery(db, createProfile, "user_profiles")
	execQuery(db, createInvNac, "investimentos_nacionais")
	execQuery(db, createInvInt, "investimentos_internacionais")
	execQuery(db, createChat, "chat_history")
	
	log.Println("Schema verificado/criado com sucesso.")
}

func execQuery(db *sql.DB, query, tableName string) {
	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Erro crítico ao criar tabela '%s': %v", tableName, err)
	}
}