// database/sqlite.go
package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3" // Driver SQLite
)

// DBConnection é a variável global para a conexão com o banco de dados
var DBConnection *sql.DB

const TableName = "movimentacoes" // Definindo o nome da tabela

// InitDB inicializa a conexão com o banco de dados. Assume que o banco de dados e a tabela já existem.
func InitDB(dbName string) (*sql.DB, error) {
	var err error
	// Abrir o arquivo do banco de dados SQLite.
	DBConnection, err = sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir o banco de dados '%s': %w", dbName, err)
	}

	// Ping para verificar a conexão
	err = DBConnection.Ping()
	if err != nil {
		DBConnection.Close() // Fechar conexão em caso de erro de ping
		return nil, fmt.Errorf("erro ao conectar ao banco de dados '%s': %w", dbName, err)
	}
	log.Printf("Conectado ao banco de dados '%s' com sucesso.", dbName)

	return DBConnection, nil
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
