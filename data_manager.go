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
	// --- 1. Definição de todas as Flags ---
	importMovimentacoes := flag.Bool("import", false, "Importar dados de movimentações de CSVs para o banco de dados.")
	exportMovimentacoes := flag.Bool("export", false, "Exportar dados de movimentações do banco de dados para um único CSV.")
	importNacionais := flag.Bool("import-nacionais", false, "Importar CSV de investimentos nacionais.")
	importInternacionais := flag.Bool("import-internacionais", false, "Importar CSV de investimentos internacionais.")
	userIdParam := flag.Int64("user-id", 0, "ID do usuário para associar os dados durante a importação/exportação.")
	outputPathParam := flag.String("output-path", "extrato_exportado.csv", "Caminho e nome do arquivo CSV para exportação.")

	flag.Parse()

	// --- 2. Validação e Conexão com o Banco de Dados ---
	if (*importMovimentacoes || *exportMovimentacoes || *importNacionais || *importInternacionais) && *userIdParam == 0 {
		log.Fatal("ERRO: A flag -user-id é obrigatória para qualquer operação de importação ou exportação.")
	}
	if (*importMovimentacoes || *exportMovimentacoes || *importNacionais || *importInternacionais) && *userIdParam == 1 {
		log.Fatal("ERRO: A importação/exportação de dados não é permitida para o usuário admin (ID 1).")
	}

	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	db := database.GetDB()
	defer database.CloseDB()
	log.Println("Conectado ao banco de dados com sucesso.")

	// --- 3. Definição das Queries de Criação de Tabelas ---
	var createUsersTableSQL, createMovimentacoesTableSQL, createContasTableSQL, createUserProfilesTableSQL, createInvestNacionaisSQL, createInvestInternacionaisSQL string

	if database.DriverName == "postgres" {
		// CORREÇÃO: Trocado "id BIGINT PRIMARY KEY" por "id BIGSERIAL PRIMARY KEY"
		// Isso cria automaticamente a sequência de auto-incremento (users_id_seq).
		createUsersTableSQL = `CREATE TABLE IF NOT EXISTS users (id BIGSERIAL PRIMARY KEY, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, is_admin BOOLEAN DEFAULT FALSE, dark_mode_enabled BOOLEAN DEFAULT FALSE);`
		createMovimentacoesTableSQL = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY, user_id BIGINT NOT NULL, data_ocorrencia DATE NOT NULL, descricao TEXT, valor NUMERIC(10, 2), categoria TEXT, conta TEXT, consolidado BOOLEAN DEFAULT FALSE, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`, tableName)
		createContasTableSQL = `CREATE TABLE IF NOT EXISTS contas (user_id BIGINT NOT NULL, nome TEXT NOT NULL, saldo_inicial NUMERIC(10, 2) NOT NULL DEFAULT 0, PRIMARY KEY (user_id, nome), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createUserProfilesTableSQL = `CREATE TABLE IF NOT EXISTS user_profiles (user_id BIGINT PRIMARY KEY, date_of_birth DATE, gender TEXT, marital_status TEXT, children_count INTEGER, country TEXT, state TEXT, city TEXT, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createInvestNacionaisSQL = `CREATE TABLE IF NOT EXISTS investimentos_nacionais (user_id BIGINT NOT NULL, ticker TEXT NOT NULL, tipo TEXT, quantidade INTEGER NOT NULL, PRIMARY KEY (user_id, ticker), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createInvestInternacionaisSQL = `CREATE TABLE IF NOT EXISTS investimentos_internacionais (user_id BIGINT NOT NULL, ticker TEXT NOT NULL, descricao TEXT, quantidade REAL NOT NULL, moeda TEXT, PRIMARY KEY (user_id, ticker), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
	} else { // Padrão para sqlite3
		createUsersTableSQL = `CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, is_admin BOOLEAN DEFAULT FALSE, dark_mode_enabled BOOLEAN DEFAULT 0);`
		createMovimentacoesTableSQL = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, data_ocorrencia TEXT NOT NULL, descricao TEXT, valor REAL, categoria TEXT, conta TEXT, consolidado BOOLEAN DEFAULT FALSE, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`, tableName)
		createContasTableSQL = `CREATE TABLE IF NOT EXISTS contas (user_id INTEGER NOT NULL, nome TEXT NOT NULL, saldo_inicial REAL NOT NULL DEFAULT 0, PRIMARY KEY (user_id, nome), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createUserProfilesTableSQL = `CREATE TABLE IF NOT EXISTS user_profiles (user_id INTEGER PRIMARY KEY, date_of_birth TEXT, gender TEXT, marital_status TEXT, children_count INTEGER, country TEXT, state TEXT, city TEXT, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createInvestNacionaisSQL = `CREATE TABLE IF NOT EXISTS investimentos_nacionais (user_id INTEGER NOT NULL, ticker TEXT NOT NULL, tipo TEXT, quantidade INTEGER NOT NULL, PRIMARY KEY (user_id, ticker), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
		createInvestInternacionaisSQL = `CREATE TABLE IF NOT EXISTS investimentos_internacionais (user_id INTEGER NOT NULL, ticker TEXT NOT NULL, descricao TEXT, quantidade REAL NOT NULL, moeda TEXT, PRIMARY KEY (user_id, ticker), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);`
	}

	// --- 4. Execução da Criação de Tabelas na Ordem Correta ---
	if _, err = db.Exec(createUsersTableSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'users': %v", err)
	}
	if _, err = db.Exec(createMovimentacoesTableSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'movimentacoes': %v", err)
	}
	if _, err = db.Exec(createContasTableSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'contas': %v", err)
	}
	if _, err = db.Exec(createUserProfilesTableSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'user_profiles': %v", err)
	}
	if _, err = db.Exec(createInvestNacionaisSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'investimentos_nacionais': %v", err)
	}
	if _, err = db.Exec(createInvestInternacionaisSQL); err != nil {
		log.Fatalf("Erro ao criar a tabela 'investimentos_internacionais': %v", err)
	}
	log.Println("Verificação/criação de todas as tabelas concluída.")

	// --- 5. Lógica de Execução com Base nas Flags ---
	operacaoRealizada := false

	if *importNacionais {
		operacaoRealizada = true
		log.Printf("-> Iniciando importação de investimentos nacionais para o Usuário ID: %d...\n", *userIdParam)
		if err := processNacionaisCSV(db, "csv/investimentos_nacionais.csv", *userIdParam); err != nil {
			log.Fatalf("ERRO ao processar CSV de investimentos nacionais: %v", err)
		}
		log.Println("-> Importação de investimentos nacionais concluída com sucesso!")
	}

	if *importInternacionais {
		operacaoRealizada = true
		log.Printf("-> Iniciando importação de investimentos internacionais para o Usuário ID: %d...\n", *userIdParam)
		if err := processInternacionaisCSV(db, "csv/investimentos_internacionais.csv", *userIdParam); err != nil {
			log.Fatalf("ERRO ao processar CSV de investimentos internacionais: %v", err)
		}
		log.Println("-> Importação de investimentos internacionais concluída com sucesso!")
	}

	if *importMovimentacoes {
		operacaoRealizada = true
		log.Printf("Modo: Importação de movimentações para o Usuário ID: %d.\n", *userIdParam)
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
			log.Printf("\nProcessando arquivo: %s\n", filename)
			if err := processCSVFile(db, filename, *userIdParam); err != nil {
				log.Printf("ERRO ao processar %s: %v\n", filename, err)
			}
		}
		log.Println("\nProcesso de importação de movimentações concluído.")
	}

	if *exportMovimentacoes {
		operacaoRealizada = true
		log.Printf("Modo: Exportação de movimentações para CSV para o Usuário ID: %d.\n", *userIdParam)
		if err := exportToCSV(db, *outputPathParam, *userIdParam); err != nil {
			log.Printf("ERRO ao exportar para CSV: %v\n", err)
			os.Exit(1)
		}
		log.Printf("Dados exportados com sucesso para '%s'.\n", *outputPathParam)
	}

	if !operacaoRealizada {
		log.Println("Nenhuma operação de import/export solicitada. Use flags como -import, -export, -import-nacionais, etc.")
	}
}

// --- Funções Auxiliares ---

func processCSVFile(db *sql.DB, filename string, userId int64) error {
	csvFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("erro ao abrir o arquivo CSV '%s': %w", filename, err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = rune(csvDelimiter)
	reader.LazyQuotes = true

	if _, err = reader.Read(); err != nil && err != io.EOF {
		return fmt.Errorf("erro ao ler o cabeçalho do CSV '%s': %w", filename, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação para '%s': %w", filename, err)
	}
	defer tx.Rollback()

	insertSQL := fmt.Sprintf(`INSERT INTO %s (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?, ?)`, tableName)
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
			log.Printf("AVISO: Erro ao ler linha do CSV '%s': %v. Pulando linha.\n", filename, readErr)
			continue
		}
		if len(record) != 6 {
			log.Printf("AVISO: Pulando linha com número inesperado de colunas (%d) em '%s': %v\n", len(record), filename, record)
			continue
		}

		dataOcorrencia, descricao, categoria, conta := record[0], record[1], record[3], record[4]
		valorStr := strings.Replace(record[2], ",", ".", -1)
		valor, convErr := strconv.ParseFloat(valorStr, 64)
		if convErr != nil {
			log.Printf("AVISO: Erro ao converter Valor '%s' em '%s': %v. Pulando linha.\n", record[2], filename, convErr)
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

	log.Printf("   %d registros importados com sucesso de '%s'.\n", recordsProcessed, filename)
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
		record := []string{formattedDateForCSV, mov.Descricao, valorFormatado, mov.Categoria, mov.Conta, strconv.FormatBool(mov.Consolidado)}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("erro ao escrever registro no CSV: %w", err)
		}
		recordsExported++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("erro durante a iteração das linhas do banco de dados: %w", err)
	}
	log.Printf("   %d registros exportados do banco de dados.\n", recordsExported)
	return nil
}

func processNacionaisCSV(db *sql.DB, filePath string, userID int64) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	if _, err := reader.Read(); err != nil {
		return fmt.Errorf("erro ao ler cabeçalho de %s: %w", filePath, err)
	}

	var query string
	if database.DriverName == "postgres" {
		query = `INSERT INTO investimentos_nacionais (user_id, tipo, ticker, quantidade) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, ticker) DO UPDATE SET tipo = EXCLUDED.tipo, quantidade = EXCLUDED.quantidade;`
	} else {
		query = `INSERT OR REPLACE INTO investimentos_nacionais (user_id, tipo, ticker, quantidade) VALUES (?, ?, ?, ?);`
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tx.Rollback()
			return err
		}

		quantidade, _ := strconv.Atoi(record[2])
		_, err = stmt.Exec(userID, record[0], record[1], quantidade)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("falha ao inserir registro %v: %w", record, err)
		}
	}
	return tx.Commit()
}

func processInternacionaisCSV(db *sql.DB, filePath string, userID int64) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	if _, err := reader.Read(); err != nil {
		return fmt.Errorf("erro ao ler cabeçalho de %s: %w", filePath, err)
	}

	var query string
	if database.DriverName == "postgres" {
		query = `INSERT INTO investimentos_internacionais (user_id, descricao, ticker, quantidade, moeda) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (user_id, ticker) DO UPDATE SET descricao = EXCLUDED.descricao, quantidade = EXCLUDED.quantidade, moeda = EXCLUDED.moeda;`
	} else {
		query = `INSERT OR REPLACE INTO investimentos_internacionais (user_id, descricao, ticker, quantidade, moeda) VALUES (?, ?, ?, ?, ?);`
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tx.Rollback()
			return err
		}

		quantidade, _ := strconv.ParseFloat(record[2], 64)
		_, err = stmt.Exec(userID, record[0], record[1], quantidade, record[3])
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("falha ao inserir registro %v: %w", record, err)
		}
	}
	return tx.Commit()
}
