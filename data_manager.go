package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"minhas_economias/database"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	tableName    = "movimentacoes"
	csvDelimiter = ';'
)

// Movimentacao representa uma linha do extrato para o processo de import/export
type Movimentacao struct {
	DataOcorrencia string
	Descricao      string
	Valor          float64
	Categoria      string
	Conta          string
	Consolidado    bool
}

func main() {
	importFlag := flag.Bool("import", false, "Use esta flag para importar dados de CSVs para o banco de dados.")
	exportFlag := flag.Bool("export", false, "Use esta flag para exportar dados do banco de dados para um único CSV.")
	outputPathParam := flag.String("output-path", "extrato_exportado.csv", "Caminho e nome do arquivo CSV para exportação.")
	userIdParam := flag.Int64("user-id", 0, "ID do usuário para associar os dados durante a importação.")

	flag.Parse()

	if (*importFlag || *exportFlag) && *userIdParam == 1 {
		log.Fatal("ERRO: A importação/exportação de dados não é permitida para o usuário admin (ID 1).")
	}

	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	db := database.GetDB()
	defer database.CloseDB()

	// SQL de criação das tabelas atualizado
	var createUsersTableSQL, createMovimentacoesTableSQL, createContasTableSQL, createUserProfilesTableSQL string

	if database.DriverName == "postgres" {
		createUsersTableSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			is_admin BOOLEAN DEFAULT FALSE,
			dark_mode_enabled BOOLEAN DEFAULT FALSE
		);`
		createUsersTableSQL += `
			DO $$
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_sequences WHERE sequencename = 'users_id_seq') THEN
					CREATE SEQUENCE users_id_seq START 2;
				END IF;
			END$$;
		`
		createMovimentacoesTableSQL = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			data_ocorrencia DATE NOT NULL,
			descricao TEXT,
			valor NUMERIC(10, 2),
			categoria TEXT,
			conta TEXT,
			consolidado BOOLEAN DEFAULT FALSE,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`, tableName)
		createContasTableSQL = `
		CREATE TABLE IF NOT EXISTS contas (
			user_id BIGINT NOT NULL,
			nome TEXT NOT NULL,
			saldo_inicial NUMERIC(10, 2) NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, nome),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
		// NOVA TABELA PARA PERFIS DE USUÁRIO (POSTGRES)
		createUserProfilesTableSQL = `
		CREATE TABLE IF NOT EXISTS user_profiles (
			user_id BIGINT PRIMARY KEY,
			date_of_birth DATE,
			gender TEXT,
			marital_status TEXT,
			children_count INTEGER,
			country TEXT,
			state TEXT,
			city TEXT,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	} else { // Padrão para sqlite3
		createUsersTableSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			is_admin BOOLEAN DEFAULT FALSE,
			dark_mode_enabled BOOLEAN DEFAULT 0
		);`
		createMovimentacoesTableSQL = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			data_ocorrencia TEXT NOT NULL,
			descricao TEXT,
			valor REAL,
			categoria TEXT,
			conta TEXT,
			consolidado BOOLEAN DEFAULT FALSE,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`, tableName)
		createContasTableSQL = `
		CREATE TABLE IF NOT EXISTS contas (
			user_id INTEGER NOT NULL,
			nome TEXT NOT NULL,
			saldo_inicial REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, nome),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
		// NOVA TABELA PARA PERFIS DE USUÁRIO (SQLITE)
		createUserProfilesTableSQL = `
		CREATE TABLE IF NOT EXISTS user_profiles (
			user_id INTEGER PRIMARY KEY,
			date_of_birth TEXT,
			gender TEXT,
			marital_status TEXT,
			children_count INTEGER,
			country TEXT,
			state TEXT,
			city TEXT,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	}

	if _, err = db.Exec(createUsersTableSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'users': %v", err)
	}
	fmt.Println("Tabela 'users' verificada/criada com sucesso.")

	if _, err = db.Exec(createMovimentacoesTableSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'movimentacoes': %v", err)
	}
	fmt.Printf("Tabela '%s' verificada/criada com sucesso.\n", tableName)

	if _, err = db.Exec(createContasTableSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'contas': %v", err)
	}
	fmt.Println("Tabela 'contas' verificada/criada com sucesso.")

	// ADICIONADO: Execução da criação da nova tabela
	if _, err = db.Exec(createUserProfilesTableSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'user_profiles': %v", err)
	}
	fmt.Println("Tabela 'user_profiles' verificada/criada com sucesso.")


	if *importFlag {
		if *userIdParam == 0 {
			log.Fatal("ERRO: A flag -user-id é obrigatória para importação.")
		}
		fmt.Printf("Modo: Importação de CSVs para o banco de dados para o Usuário ID: %d.\n", *userIdParam)
		csvFiles := []string{
			"csv/Extrato_20130101_20131231.csv", "csv/Extrato_20140101_20141231.csv",
			"csv/Extrato_20150101_20151231.csv", "csv/Extrato_20160101_20161231.csv",
			"csv/Extrato_20170101_20171231.csv", "csv/Extrato_20180101_20181231.csv",
			"csv/Extrato_20190101_20191231.csv", "csv/Extrato_20200101_20201231.csv",
			"csv/Extrato_20210101_20211231.csv", "csv/Extrato_20220101_20221231.csv",
			"csv/Extrato_20230101_20231231.csv", "csv/Extrato_20240101_20241231.csv",
			"csv/Extrato_20250101_20251231.csv",
		}

		for _, filename := range csvFiles {
			fmt.Printf("\nProcessando arquivo: %s\n", filename)
			if err := processCSVFile(db, filename, *userIdParam); err != nil {
				fmt.Printf("ERRO ao processar %s: %v\n", filename, err)
			}
		}
		fmt.Println("\nProcesso de importação concluído.")

	} else if *exportFlag {
		if *userIdParam == 0 {
			log.Fatal("ERRO: A flag -user-id é obrigatória para exportação.")
		}
		fmt.Printf("Modo: Exportação do banco de dados para CSV para o Usuário ID: %d.\n", *userIdParam)
		if err := exportToCSV(db, *outputPathParam, *userIdParam); err != nil {
			fmt.Printf("ERRO ao exportar para CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Dados exportados com sucesso para '%s'.\n", *outputPathParam)

	} else {
		fmt.Println("Tabelas verificadas/criadas. Nenhuma operação de import/export solicitada.")
	}
}

func processCSVFile(db *sql.DB, filename string, userId int64) error {
	csvFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("erro ao abrir o arquivo CSV '%s': %w", filename, err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = rune(csvDelimiter)
	reader.LazyQuotes = true

	_, err = reader.Read() // Pular cabeçalho
	if err != nil && err != io.EOF {
		return fmt.Errorf("erro ao ler o cabeçalho do CSV '%s': %w", filename, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação para '%s': %w", filename, err)
	}
	defer tx.Rollback()

	insertSQL := fmt.Sprintf(
		`INSERT INTO %s (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?, ?)`, tableName)

	reboundSQL := database.Rebind(insertSQL)
	stmt, err := tx.Prepare(reboundSQL)
	if err != nil {
		return fmt.Errorf("erro ao preparar a instrução SQL para '%s': %w", filename, err)
	}
	defer stmt.Close()

	recordsProcessed := 0
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			fmt.Printf("AVISO: Erro ao ler linha do CSV '%s': %v. Pulando linha.\n", filename, readErr)
			continue
		}

		if len(record) != 6 {
			fmt.Printf("AVISO: Pulando linha com número inesperado de colunas (%d) em '%s': %v\n", len(record), filename, record)
			continue
		}

		dataOcorrencia, descricao, categoria, conta := record[0], record[1], record[3], record[4]
		valorStr := strings.Replace(record[2], ",", ".", -1)
		valor, convErr := strconv.ParseFloat(valorStr, 64)
		if convErr != nil {
			fmt.Printf("AVISO: Erro ao converter Valor '%s' em '%s': %v. Pulando linha.\n", record[2], filename, convErr)
			continue
		}
		parsedDate, timeErr := time.Parse("02/01/2006", dataOcorrencia)
		formattedDate := dataOcorrencia
		if timeErr == nil {
			formattedDate = parsedDate.Format("2006-01-02")
		}
		consolidado, convErr := strconv.ParseBool(strings.ToLower(record[5]))
		if convErr != nil {
			consolidado = false
		}

		if _, execErr := stmt.Exec(userId, formattedDate, descricao, valor, categoria, conta, consolidado); execErr != nil {
			return fmt.Errorf("erro ao inserir dados da linha %v de '%s': %w", record, filename, execErr)
		}
		recordsProcessed++
	}

	fmt.Printf(" 	%d registros importados com sucesso de '%s'.\n", recordsProcessed, filename)
	return tx.Commit()
}

func exportToCSV(db *sql.DB, outputFilename string, userId int64) error {
	if outputDir := filepath.Dir(outputFilename); outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório '%s': %w", outputDir, err)
		}
	}

	file, err := os.Create(outputFilename)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo CSV de saída '%s': %w", outputFilename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = rune(csvDelimiter)
	defer writer.Flush()

	header := []string{"Data Ocorrência", "Descrição", "Valor", "Categoria", "Conta", "Consolidado"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("erro ao escrever cabeçalho do CSV: %w", err)
	}

	query := fmt.Sprintf("SELECT data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ? ORDER BY data_ocorrencia ASC", tableName)
	reboundQuery := database.Rebind(query)
	rows, err := db.Query(reboundQuery, userId)
	if err != nil {
		return fmt.Errorf("erro ao consultar o banco de dados para exportação: %w", err)
	}
	defer rows.Close()

	recordsExported := 0
	for rows.Next() {
		var mov Movimentacao
		var rawDate interface{}

		if err := rows.Scan(&rawDate, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			log.Printf("AVISO: Erro ao escanear linha do banco de dados para exportação: %v. Pulando linha.\n", err)
			continue
		}

		var formattedDateForCSV string
		if t, ok := rawDate.(time.Time); ok {
			formattedDateForCSV = t.Format("02/01/2006")
		} else if s, ok := rawDate.(string); ok {
			parsedDate, err := time.Parse("2006-01-02", s)
			if err == nil {
				formattedDateForCSV = parsedDate.Format("02/01/2006")
			} else {
				formattedDateForCSV = s
			}
		}

		valorFormatado := strings.Replace(strconv.FormatFloat(mov.Valor, 'f', 2, 64), ".", ",", -1)
		record := []string{
			formattedDateForCSV, mov.Descricao, valorFormatado, mov.Categoria, mov.Conta, strconv.FormatBool(mov.Consolidado),
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("erro ao escrever registro no CSV: %w", err)
		}
		recordsExported++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("erro durante a iteração das linhas do banco de dados: %w", err)
	}

	fmt.Printf(" 	%d registros exportados do banco de dados.\n", recordsExported)
	return nil
}
