// database/database.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"           // Driver PostgreSQL
	_ "github.com/mattn/go-sqlite3" // Driver SQLite
)

// DBConnection é a variável global para a conexão com o banco de dados
var DBConnection *sql.DB
// DriverName armazena o tipo de banco de dados em uso ("postgres" ou "sqlite3")
var DriverName string

// TableName é o nome da tabela principal da aplicação
const TableName = "movimentacoes"

// getEnv retorna o valor de uma variável de ambiente ou um valor padrão.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// InitDB inicializa a conexão com o banco de dados com base na variável de ambiente DB_TYPE.
func InitDB() (*sql.DB, error) {
	DriverName = getEnv("DB_TYPE", "sqlite3") // Padrão para sqlite3 se não definido
	var err error

	switch DriverName {
	case "postgres":
		user := getEnv("DB_USER", "postgres")
		pass := getEnv("DB_PASS", "postgres")
		host := getEnv("DB_HOST", "localhost")
		dbname := getEnv("DB_NAME", "minhas_economias")
		port := getEnv("DB_PORT", "5432")

		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, pass, dbname)

		DBConnection, err = sql.Open("postgres", connStr)
		if err != nil {
			return nil, fmt.Errorf("erro ao abrir o banco de dados postgres: %w", err)
		}
		log.Printf("Conectado ao banco de dados PostgreSQL '%s' em '%s'.", dbname, host)

	case "sqlite3":
		dbPath := getEnv("DB_NAME", "extratos.db")
		DBConnection, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, fmt.Errorf("erro ao abrir o banco de dados sqlite '%s': %w", dbPath, err)
		}
		log.Printf("Conectado ao banco de dados SQLite em '%s'.", dbPath)

	default:
		return nil, fmt.Errorf("DB_TYPE '%s' não suportado", DriverName)
	}

	err = DBConnection.Ping()
	if err != nil {
		if DBConnection != nil {
			DBConnection.Close()
		}
		return nil, fmt.Errorf("erro ao conectar ao banco de dados: %w", err)
	}
	
	return DBConnection, nil
}

// Rebind adapta uma query com placeholders '?' para a sintaxe do driver de banco de dados atual.
func Rebind(query string) string {
	if DriverName == "postgres" {
		parts := strings.Split(query, "?")
		var result strings.Builder
		for i, part := range parts {
			if i < len(parts)-1 {
				result.WriteString(part)
				result.WriteString(fmt.Sprintf("$%d", i+1))
			} else {
				result.WriteString(part)
			}
		}
		return result.String()
	}
	return query
}

// CloseDB fecha a conexão com o banco de dados
func CloseDB() {
	if DBConnection != nil {
		DBConnection.Close()
		log.Println("Conexão com o banco de dados fechada.")
	}
}

// GetDB retorna a instância da conexão com o banco de dados.
func GetDB() *sql.DB {
	return DBConnection
}