package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // Driver SQLite para Go
)

const (
	dbName       = "extratos.db" // Nome do arquivo do banco de dados SQLite
	tableName    = "movimentacoes"
	csvDelimiter = ';' // Delimitador do seu CSV (ponto e vírgula)
)

// Movimentacao representa uma linha do extrato com base no NOVO cabeçalho CSV, incluindo 'Consolidado'
type Movimentacao struct {
	DataOcorrencia string  // Mapeia para "Data Ocorrência"
	Descricao      string  // Mapeia para "Descrição"
	Valor          float64 // Mapeia para "Valor"
	Categoria      string  // Mapeia para "Categoria"
	Conta          string  // Mapeia para "Conta"
	Consolidado    bool    // Nova coluna: Mapeia para "Consolidado"
}

func main() {
	// 1. Abrir o banco de dados SQLite
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		fmt.Printf("Erro ao abrir o banco de dados: %v\n", err)
		return
	}
	defer db.Close()

	// 2. Criar a tabela se não existir com o NOVO ESQUEMA (incluindo 'consolidado')
	// VOCÊ DEVE EXCLUIR O ARQUIVO 'extratos.db' SE ELE JÁ EXISTIR PARA APLICAR ESTE NOVO ESQUEMA.
	// CASO CONTRÁRIO, A COLUNA NÃO SERÁ ADICIONADA.
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data_ocorrencia TEXT NOT NULL,
			descricao TEXT,
			valor REAL,
			categoria TEXT,
			conta TEXT,
			consolidado BOOLEAN DEFAULT FALSE -- Nova coluna
		);`, tableName)

	_, err = db.Exec(createTableSQL)
	if err != nil {
		fmt.Printf("Erro ao criar a tabela: %v\n", err)
		return
	}
	fmt.Printf("Tabela '%s' verificada/criada com sucesso com o novo esquema. (Lembre-se de excluir 'extratos.db' se já existia)\n", tableName)

	// Lista de arquivos CSV a serem processados.
	// Adicione o caminho correto para os seus arquivos.
	// Ex: "Extrato_20130101_20131231.csv" se estiver no mesmo nível
	// Ex: "csv/Extrato_20230101_20231231.csv" se estiver em um subdiretório 'csv'
	csvFiles := []string{
		// "csv/example.csv" 
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
		err := processCSVFile(db, filename)
		if err != nil {
			fmt.Printf("ERRO ao processar %s: %v\n", filename, err)
		}
	}

	fmt.Println("\nProcesso de importação concluído.")
}

// processCSVFile lida com a abertura, leitura e inserção de dados de um único arquivo CSV.
// Agora, espera um formato de 6 colunas: Data Ocorrência;Descrição;Valor;Categoria;Conta;Consolidado
func processCSVFile(db *sql.DB, filename string) error {
	csvFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("erro ao abrir o arquivo CSV '%s': %w", filename, err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = rune(csvDelimiter)
	reader.LazyQuotes = true

	// Pular o cabeçalho (que agora é o novo cabeçalho de 6 colunas)
	_, err = reader.Read()
	if err != nil && err != io.EOF {
		return fmt.Errorf("erro ao ler o cabeçalho do CSV '%s': %w", filename, err)
	}

	tx, err := db.Begin() // Inicia uma transação para inserções mais rápidas
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação para '%s': %w", filename, err)
	}
	// Em caso de erro na transação, garantimos o rollback
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback()
		}
	}()

	// A instrução INSERT agora corresponde ao novo esquema de 6 colunas
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
			continue // Pular para a próxima linha em caso de erro na leitura
		}

		// Validar se a linha tem o número correto de colunas (agora 6)
		if len(record) != 6 {
			fmt.Printf("AVISO: Pulando linha com número inesperado de colunas (%d em vez de 6) em '%s': %v\n", len(record), filename, record)
			continue
		}

		// Mapeamento direto das 6 colunas do CSV para a struct Movimentacao
		dataOcorrencia := record[0]
		descricao := record[1]
		categoria := record[3]
		conta := record[4]

		// Conversão do Valor (record[2])
		valorStr := strings.Replace(record[2], ",", ".", -1) // Substitui vírgula por ponto
		valor, err := strconv.ParseFloat(valorStr, 64)
		if err != nil {
			fmt.Printf("AVISO: Erro ao converter Valor '%s' (coluna 3) em '%s': %v. Pulando linha: %v\n", record[2], filename, err, record)
			continue
		}

		// Convertendo a data para o formato AAAA-MM-DD
		parsedDate, err := time.Parse("02/01/2006", dataOcorrencia) // Formato de entrada DD/MM/YYYY
		formattedDate := dataOcorrencia                             // Fallback para a data original se houver erro
		if err != nil {
			fmt.Printf("AVISO: Erro ao parsear 'Data Ocorrência' '%s' em '%s': %v. Usando formato original.\n", dataOcorrencia, filename, err)
		} else {
			formattedDate = parsedDate.Format("2006-01-02") // Formato de saída AAAA-MM-DD
		}

		// Conversão do Consolidado (record[5]) para bool
		consolidadoStr := strings.ToLower(record[5])
		consolidado, err := strconv.ParseBool(consolidadoStr)
		if err != nil {
			fmt.Printf("AVISO: Erro ao converter Consolidado '%s' (coluna 6) em '%s': %v. Usando 'false'.\n", record[5], filename, err)
			consolidado = false // Define como false em caso de erro na conversão
		}

		// Cria a struct com os dados mapeados
		mov := Movimentacao{
			DataOcorrencia: formattedDate,
			Descricao:      descricao,
			Valor:          valor,
			Categoria:      categoria,
			Conta:          conta,
			Consolidado:    consolidado, // Adicionando o novo campo
		}

		// Executa a inserção dos dados no banco de dados
		_, err = stmt.Exec(mov.DataOcorrencia, mov.Descricao, mov.Valor, mov.Categoria, mov.Conta, mov.Consolidado)
		if err != nil {
			fmt.Printf("ERRO: Erro ao inserir dados da linha %v de '%s': %v\\n", record, filename, err)
			return fmt.Errorf("erro ao inserir dados para '%s': %w", filename, err) // Propaga o erro
		}
		recordsProcessed++
	}

	err = tx.Commit() // Confirma a transação
	if err != nil {
		return fmt.Errorf("erro ao commitar transação para '%s': %w", filename, err)
	}

	fmt.Printf("  %d registros importados com sucesso de '%s'.\n", recordsProcessed, filename)
	return nil
}
