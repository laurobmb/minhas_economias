package main

import (
	"database/sql"
	"encoding/csv"
	"flag" // Importe o pacote flag
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath" // Importe o pacote filepath
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // Driver SQLite para Go
)

const (
	dbName = "extratos.db"   // Nome do arquivo do banco de dados SQLite
	tableName = "movimentacoes" // Nome da tabela
	csvDelimiter = ';'         // Delimitador do seu CSV (ponto e vírgula)
)

// Movimentacao representa uma linha do extrato com base no cabeçalho CSV
type Movimentacao struct {
	DataOcorrencia string  // Mapeia para "Data Ocorrência" (AAAA-MM-DD)
	Descricao      string  // Mapeia para "Descrição"
	Valor          float64 // Mapeia para "Valor"
	Categoria      string  // Mapeia para "Categoria"
	Conta          string  // Mapeia para "Conta"
	Consolidado    bool    // Mapeia para "Consolidado"
}

func main() {
	// Definindo as flags de linha de comando
	importFlag := flag.Bool("import", false, "Use esta flag para importar dados de CSVs para o banco de dados.")
	exportFlag := flag.Bool("export", false, "Use esta flag para exportar dados do banco de dados para um único CSV.")
	// Nova flag para o caminho de saída da exportação
	outputPathParam := flag.String("output-path", "extrato_exportado.csv", "Caminho e nome do arquivo CSV para exportação (ex: 'data/extrato.csv').")
	
	flag.Parse() // Analisa as flags da linha de comando

	// 1. Abrir o banco de dados SQLite
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		fmt.Printf("Erro ao abrir o banco de dados: %v\n", err)
		os.Exit(1) // Sair com erro
	}
	defer db.Close()

	// 2. Criar a tabela se não existir com o ESQUEMA
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data_ocorrencia TEXT NOT NULL,
			descricao TEXT,
			valor REAL,
			categoria TEXT,
			conta TEXT,
			consolidado BOOLEAN DEFAULT FALSE
		);`, tableName)

	_, err = db.Exec(createTableSQL)
	if err != nil {
		fmt.Printf("Erro ao criar a tabela: %v\n", err)
		os.Exit(1) // Sair com erro
	}
	fmt.Printf("Tabela '%s' verificada/criada com sucesso.\n", tableName)

	// Lógica baseada nas flags
	if *importFlag {
		fmt.Println("Modo: Importação de CSVs para o banco de dados.")
		csvFiles := []string{
			// Adicione aqui os caminhos corretos para os seus arquivos CSV de importação
			"csv/Extrato_20130101_20131231.csv",
			"csv/Extrato_20140101_20141231.csv",
			"csv/Extrato_20150101_20151231.csv",
			"csv/Extrato_20160101_20161231.csv",
			"csv/Extrato_20170101_20171231.csv",
			"csv/Extrato_20180101_20181231.csv",
			"csv/Extrato_20190101_20191231.csv",
			"csv/Extrato_20200101_20201231.csv",
			"csv/Extrato_20210101_20211231.csv",
			"csv/Extrato_20220101_20221231.csv",
			"csv/Extrato_20230101_20231231.csv",
			"csv/Extrato_20240101_20241231.csv",
			"csv/Extrato_20250101_20250630.csv",
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
		// Passa o valor da flag output-path para a função de exportação
		if err := exportToCSV(db, *outputPathParam); err != nil { 
			fmt.Printf("ERRO ao exportar para CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Dados exportados com sucesso para '%s'.\n", *outputPathParam) // Confirma o caminho usado

	} else {
		fmt.Println("Nenhuma flag de operação (-import ou -export) fornecida.")
		fmt.Println("Use: go run data_manager.go -import para importar CSVs.")
		fmt.Println("Use: go run data_manager.go -export para exportar para CSV.")
		fmt.Println("Opcional para exportar: -output-path <caminho/do/arquivo.csv>")
		os.Exit(1) // Sair com erro
	}
}

// processCSVFile lida com a abertura, leitura e inserção de dados de um único arquivo CSV.
// Espera um formato de 6 colunas: Data Ocorrência;Descrição;Valor;Categoria;Conta;Consolidado
func processCSVFile(db *sql.DB, filename string) error {
	csvFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("erro ao abrir o arquivo CSV '%s': %w", filename, err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = rune(csvDelimiter)
	reader.LazyQuotes = true

	// Pular o cabeçalho
	_, err = reader.Read()
	if err != nil && err != io.EOF {
		return fmt.Errorf("erro ao ler o cabeçalho do CSV '%s': %w", filename, err)
	}

	tx, err := db.Begin() // Inicia uma transação para inserções mais rápidas
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação para '%s': %w", filename, err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-throw panic after Rollback
		} else if err != nil { // verifica se 'err' é nil para evitar rollback desnecessário
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(fmt.Sprintf(
		`INSERT INTO %s (data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?)`, tableName))
	if err != nil {
		return fmt.Errorf("erro ao preparar a instrução SQL para '%s': %w", filename, err)
	}
	defer stmt.Close()

	recordsProcessed := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // Fim do arquivo
		}
		if err != nil {
			fmt.Printf("AVISO: Erro ao ler linha do CSV '%s': %v. Pulando esta linha.\n", filename, err)
			continue
		}

		if len(record) != 6 {
			fmt.Printf("AVISO: Pulando linha com número inesperado de colunas (%d em vez de 6) em '%s': %v\n", len(record), filename, record)
			continue
		}

		dataOcorrencia := record[0]
		descricao := record[1]
		categoria := record[3]
		conta := record[4]

		valorStr := strings.Replace(record[2], ",", ".", -1)
		valor, err := strconv.ParseFloat(valorStr, 64)
		if err != nil {
			fmt.Printf("AVISO: Erro ao converter Valor '%s' (coluna 3) em '%s': %v. Pulando linha: %v\n", record[2], filename, err, record)
			continue
		}

		parsedDate, err := time.Parse("02/01/2006", dataOcorrencia) // Formato de entrada DD/MM/YYYY
		formattedDate := dataOcorrencia
		if err != nil {
			fmt.Printf("AVISO: Erro ao parsear 'Data Ocorrência' '%s' em '%s': %v. Usando formato original.\\n", dataOcorrencia, filename, err)
		} else {
			formattedDate = parsedDate.Format("2006-01-02") // Formato de saída AAAA-MM-DD
		}

		consolidadoStr := strings.ToLower(record[5])
		consolidado, err := strconv.ParseBool(consolidadoStr)
		if err != nil {
			fmt.Printf("AVISO: Erro ao converter Consolidado '%s' (coluna 6) em '%s': %v. Usando 'false'.\n", record[5], filename, err)
			consolidado = false
		}

		mov := Movimentacao{
			DataOcorrencia: formattedDate,
			Descricao:      descricao,
			Valor:          valor,
			Categoria:      categoria,
			Conta:          conta,
			Consolidado:    consolidado,
		}

		_, err = stmt.Exec(mov.DataOcorrencia, mov.Descricao, mov.Valor, mov.Categoria, mov.Conta, mov.Consolidado)
		if err != nil {
			fmt.Printf("ERRO: Erro ao inserir dados da linha %v de '%s': %v\n", record, filename, err)
			return fmt.Errorf("erro ao inserir dados para '%s': %w", filename, err)
		}
		recordsProcessed++
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("erro ao commitar transação para '%s': %w", filename, err)
	}

	fmt.Printf("  %d registros importados com sucesso de '%s'.\n", recordsProcessed, filename)
	return nil
}

// exportToCSV lê todos os dados do banco de dados e os grava em um arquivo CSV.
func exportToCSV(db *sql.DB, outputFilename string) error {
	// Garante que o diretório de saída exista
	outputDir := filepath.Dir(outputFilename)
	if outputDir != "" && outputDir != "." { // Verifica se há um diretório especificado
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
	defer writer.Flush() // Garante que todos os dados sejam gravados no arquivo

	// Escrever o cabeçalho do CSV
	header := []string{"Data Ocorrência", "Descrição", "Valor", "Categoria", "Conta", "Consolidado"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("erro ao escrever cabeçalho do CSV: %w", err)
	}

	// Consultar todos os registros da tabela movimentacoes
	rows, err := db.Query(fmt.Sprintf("SELECT data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s ORDER BY data_ocorrencia ASC", tableName))
	if err != nil {
		return fmt.Errorf("erro ao consultar o banco de dados para exportação: %w", err)
	}
	defer rows.Close()

	recordsExported := 0
	for rows.Next() {
		var mov Movimentacao
		err := rows.Scan(&mov.DataOcorrencia, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado) 
		if err != nil {
			log.Printf("AVISO: Erro ao escanear linha do banco de dados para exportação: %v. Pulando esta linha.\n", err)
			continue
		}

		// Formatar o valor para o padrão brasileiro com vírgula
		valorFormatado := strconv.FormatFloat(mov.Valor, 'f', 2, 64)
		valorFormatado = strings.Replace(valorFormatado, ".", ",", -1)

		// Formatar a data para DD/MM/YYYY se for necessário reverter do AAAA-MM-DD
		parsedDate, err := time.Parse("2006-01-02", mov.DataOcorrencia) // CORREÇÃO: "DataOcorrencia" sem cedilha
		formattedDateForCSV := mov.DataOcorrencia // Fallback
		if err != nil {
			log.Printf("AVISO: Erro ao parsear data '%s' para exportação: %v. Usando formato original.\n", mov.DataOcorrencia, err) // CORREÇÃO: "DataOcorrencia" sem cedilha
		} else {
			formattedDateForCSV = parsedDate.Format("02/01/2006")
		}

		record := []string{
			formattedDateForCSV,
			mov.Descricao,
			valorFormatado,
			mov.Categoria,
			mov.Conta,
			strconv.FormatBool(mov.Consolidado), // Converte bool para "true" ou "false"
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("erro ao escrever registro no CSV: %w", err)
		}
		recordsExported++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("erro durante a iteração das linhas do banco de dados para exportação: %w", err)
	}

	fmt.Printf("  %d registros exportados do banco de dados.\n", recordsExported)
	return nil
}
