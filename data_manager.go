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
	tableName    = "movimentacoes" // Nome da tabela
	csvDelimiter = ';'             // Delimitador do seu CSV
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

	flag.Parse()

	// 1. Conectar ao banco de dados usando a configuração do ambiente
	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	db := database.GetDB()
	defer database.CloseDB()

	// 2. Definir e executar as queries de criação de tabela de acordo com o DB_TYPE
	var createMovimentacoesTableSQL, createContasTableSQL string

	if database.DriverName == "postgres" {
		createMovimentacoesTableSQL = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id SERIAL PRIMARY KEY, data_ocorrencia DATE NOT NULL, descricao TEXT,
				valor NUMERIC(10, 2), categoria TEXT, conta TEXT, consolidado BOOLEAN DEFAULT FALSE
			);`, tableName)
		createContasTableSQL = `
			CREATE TABLE IF NOT EXISTS contas (
				nome TEXT PRIMARY KEY, saldo_inicial NUMERIC(10, 2) NOT NULL DEFAULT 0
			);`
	} else { // Padrão para sqlite3
		createMovimentacoesTableSQL = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id INTEGER PRIMARY KEY AUTOINCREMENT, data_ocorrencia TEXT NOT NULL, descricao TEXT,
				valor REAL, categoria TEXT, conta TEXT, consolidado BOOLEAN DEFAULT FALSE
			);`, tableName)
		createContasTableSQL = `
			CREATE TABLE IF NOT EXISTS contas (
				nome TEXT PRIMARY KEY, saldo_inicial REAL NOT NULL DEFAULT 0
			);`
	}

	if _, err = db.Exec(createMovimentacoesTableSQL); err != nil {
		fmt.Printf("Erro ao criar a tabela 'movimentacoes': %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Tabela '%s' verificada/criada com sucesso.\n", tableName)

	if _, err = db.Exec(createContasTableSQL); err != nil {
		fmt.Printf("Erro ao criar a tabela 'contas': %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Tabela 'contas' verificada/criada com sucesso.")

	// 3. Executar a lógica de importação ou exportação
	if *importFlag {
		fmt.Println("Modo: Importação de CSVs para o banco de dados.")
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
			if err := processCSVFile(db, filename); err != nil {
				fmt.Printf("ERRO ao processar %s: %v\n", filename, err)
			}
		}
		fmt.Println("\nProcesso de importação concluído.")

	} else if *exportFlag {
		fmt.Println("Modo: Exportação do banco de dados para CSV.")
		if err := exportToCSV(db, *outputPathParam); err != nil {
			fmt.Printf("ERRO ao exportar para CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Dados exportados com sucesso para '%s'.\n", *outputPathParam)

	} else {
		fmt.Println("Nenhuma flag de operação (-import ou -export) fornecida.")
		fmt.Println("Use: go run data_manager.go -import para importar CSVs.")
		fmt.Println("Use: go run data_manager.go -export -output-path <caminho/do/arquivo.csv> para exportar para CSV.")
		os.Exit(1)
	}
}

func processCSVFile(db *sql.DB, filename string) error {
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
	defer tx.Rollback() // Rollback é o padrão, commit será explícito

	// Prepara a query com '?'
	insertSQL := fmt.Sprintf(
		`INSERT INTO %s (data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?)`, tableName)
	
	// Adapta para a sintaxe do banco de dados atual
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

		// Parse dos dados do CSV
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
			consolidado = false // Assume false em caso de erro
		}

		// Executa a inserção
		if _, execErr := stmt.Exec(formattedDate, descricao, valor, categoria, conta, consolidado); execErr != nil {
			return fmt.Errorf("erro ao inserir dados da linha %v de '%s': %w", record, filename, execErr)
		}
		recordsProcessed++
	}

	fmt.Printf("  %d registros importados com sucesso de '%s'.\n", recordsProcessed, filename)
	
	// Se tudo deu certo, commita a transação
	return tx.Commit()
}

func exportToCSV(db *sql.DB, outputFilename string) error {
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

	// Esta query não tem placeholders, então não precisa de Rebind
	query := fmt.Sprintf("SELECT data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s ORDER BY data_ocorrencia ASC", tableName)
	rows, err := db.Query(query)
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
				formattedDateForCSV = s // fallback para o formato original
			}
		}

		valorFormatado := strings.Replace(strconv.FormatFloat(mov.Valor, 'f', 2, 64), ".", ",", -1)

		record := []string{
			formattedDateForCSV,
			mov.Descricao,
			valorFormatado,
			mov.Categoria,
			mov.Conta,
			strconv.FormatBool(mov.Consolidado),
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("erro ao escrever registro no CSV: %w", err)
		}
		recordsExported++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("erro durante a iteração das linhas do banco de dados: %w", err)
	}

	fmt.Printf("  %d registros exportados do banco de dados.\n", recordsExported)
	return nil
}