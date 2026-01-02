package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"minhas_economias/database"
	"os"
	"strconv"
	"strings"
	"time"
)



func runImportMovimentacoes(db *sql.DB, userID int64) {

	runPopulateSaldos(db, userID)

	csvFiles := []string{
		"csv/Extrato_20130101_20131231.csv", "csv/Extrato_20140101_20141231.csv",
		"csv/Extrato_20150101_20151231.csv", "csv/Extrato_20160101_20161231.csv",
		"csv/Extrato_20170101_20171231.csv", "csv/Extrato_20180101_20181231.csv",
		"csv/Extrato_20190101_20191231.csv", "csv/Extrato_20200101_20201231.csv",
		"csv/Extrato_20210101_20211231.csv", "csv/Extrato_20220101_20221231.csv",
		"csv/Extrato_20230101_20231231.csv", "csv/Extrato_20240101_20241231.csv",
		"csv/Extrato_20250101_20251231.csv", "csv/example.csv",
	}

	for _, filename := range csvFiles {
		log.Printf("Processando: %s", filename)
		if err := processCSVFile(db, filename, userID); err != nil {
			log.Printf("ERRO ao processar %s: %v", filename, err)
		}
	}
	log.Println("Importação de movimentações concluída.")
}

func processCSVFile(db *sql.DB, filename string, userId int64) error {
	csvFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("erro abrir arquivo: %w", err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = rune(csvDelimiter)
	reader.LazyQuotes = true

	if _, err = reader.Read(); err != nil && err != io.EOF {
		return fmt.Errorf("erro ler header: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertSQL := fmt.Sprintf(`INSERT INTO %s (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?, ?)`, tableName)
	stmt, err := tx.Prepare(database.Rebind(insertSQL))
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) != 6 {
			continue
		}

		valorStr := strings.Replace(record[2], ",", ".", -1)
		valor, _ := strconv.ParseFloat(valorStr, 64)

		parsedDate, _ := time.Parse("02/01/2006", record[0])
		formattedDate := parsedDate.Format("2006-01-02")
		if formattedDate == "0001-01-01" { formattedDate = record[0] } // Fallback

		consolidado := strings.ToLower(record[5]) == "true"

		if _, err := stmt.Exec(userId, formattedDate, record[1], valor, record[3], record[4], consolidado); err != nil {
			return err
		}
		count++
	}
	log.Printf("   %d linhas importadas.", count)
	return tx.Commit()
}

// --- Investimentos Nacionais ---

func runImportInvestimentosNacionais(db *sql.DB, userID int64) {
	log.Println("Iniciando importação de Investimentos Nacionais...")
	if err := processNacionaisCSV(db, "csv/investimentos_nacionais.csv", userID); err != nil {
		log.Printf("ERRO: %v", err)
	} else {
		log.Println("Sucesso!")
	}
}

func processNacionaisCSV(db *sql.DB, filePath string, userID int64) error {
	file, err := os.Open(filePath)
	if err != nil { return err }
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	if _, err := reader.Read(); err != nil { return err } // Pula header

	var query string
	if database.DriverName == "postgres" {
		query = `INSERT INTO investimentos_nacionais (user_id, tipo, ticker, quantidade) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, ticker) DO UPDATE SET tipo = EXCLUDED.tipo, quantidade = EXCLUDED.quantidade;`
	} else {
		query = `INSERT OR REPLACE INTO investimentos_nacionais (user_id, tipo, ticker, quantidade) VALUES (?, ?, ?, ?);`
	}

	tx, err := db.Begin()
	if err != nil { return err }
	stmt, err := tx.Prepare(query)
	if err != nil { tx.Rollback(); return err }
	defer stmt.Close()

	for {
		record, err := reader.Read()
		if err == io.EOF { break }
		if err != nil { tx.Rollback(); return err }

		qtd, _ := strconv.Atoi(record[2])
		// CSV: [0]Tipo, [1]Ticker, [2]Qtd
		_, err = stmt.Exec(userID, record[0], record[1], qtd)
		if err != nil { tx.Rollback(); return err }
	}
	return tx.Commit()
}

// --- Investimentos Internacionais (CORREÇÃO APLICADA) ---

func runImportInvestimentosInternacionais(db *sql.DB, userID int64) {
	log.Println("Iniciando importação de Investimentos Internacionais...")
	if err := processInternacionaisCSV(db, "csv/investimentos_internacionais.csv", userID); err != nil {
		log.Printf("ERRO: %v", err)
	} else {
		log.Println("Sucesso!")
	}
}

func processInternacionaisCSV(db *sql.DB, filePath string, userID int64) error {
	file, err := os.Open(filePath)
	if err != nil { return err }
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	if _, err := reader.Read(); err != nil { return err } // Pula header

	// Query espera: user_id ($1), descricao ($2), ticker ($3), quantidade ($4), moeda ($5)
	var query string
	if database.DriverName == "postgres" {
		query = `INSERT INTO investimentos_internacionais (user_id, descricao, ticker, quantidade, moeda) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (user_id, ticker) DO UPDATE SET descricao = EXCLUDED.descricao, quantidade = EXCLUDED.quantidade, moeda = EXCLUDED.moeda;`
	} else {
		query = `INSERT OR REPLACE INTO investimentos_internacionais (user_id, descricao, ticker, quantidade, moeda) VALUES (?, ?, ?, ?, ?);`
	}

	tx, err := db.Begin()
	if err != nil { return err }
	stmt, err := tx.Prepare(query)
	if err != nil { tx.Rollback(); return err }
	defer stmt.Close()

	for {
		record, err := reader.Read()
		if err == io.EOF { break }
		if err != nil { tx.Rollback(); return err }

		// CSV Esperado: [0]Ticker, [1]Descricao, [2]Quantidade, [3]Moeda
		qtd, _ := strconv.ParseFloat(record[2], 64)

		// CORREÇÃO: A ordem dos parâmetros estava invertida.
		// Antes: stmt.Exec(userID, record[0], record[1], ...) -> Ticker ia para Descricao
		// Agora: stmt.Exec(userID, record[1], record[0], ...) -> Descricao vai para Descricao ($2), Ticker vai para Ticker ($3)
		_, err = stmt.Exec(userID, record[1], record[0], qtd, record[3])
		
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("falha ao inserir %v: %w", record, err)
		}
	}
	return tx.Commit()
}

func runPopulateSaldos(db *sql.DB, userID int64) {
	log.Println("Populando saldos iniciais das contas...")

	// Seus valores pessoais hardcoded aqui
	saldos := map[string]float64{
		"bradescard":          0.00,
		"c6":                  0.00,
		"nuinvest":            0.00,
		"em_casa":             0.00,
		"inter":               0.00,
		"mercado_pago":        0.00,
		"nuconta":             0.00,
		"p_1421_2312_8":       0.00,
		"p_1421_2534_1":       0.00,
		"btg":                 8112.55,
		"cc_1421_20660_1":     1413.93,
		"cc_brasil":           162.83,
		"meu_bolso":           67.60,
		"p_banco_brasil_96":   0.01,
		"p_bancobrasil_51":    8013.82,
		"vale_refeicao":       308.37,
	}

	var query string
	if database.DriverName == "postgres" {
		query = `INSERT INTO contas (user_id, nome, saldo_inicial) VALUES ($1, $2, $3)
				 ON CONFLICT (user_id, nome) DO UPDATE SET saldo_inicial = EXCLUDED.saldo_inicial;`
	} else {
		query = `INSERT OR REPLACE INTO contas (user_id, nome, saldo_inicial) VALUES (?, ?, ?);`
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Erro ao iniciar transação de saldos: %v", err)
		return
	}

	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		log.Printf("Erro ao preparar query de saldos: %v", err)
		return
	}
	defer stmt.Close()

	for conta, valor := range saldos {
		_, err := stmt.Exec(userID, conta, valor)
		if err != nil {
			log.Printf("Erro ao inserir saldo para %s: %v", conta, err)
			tx.Rollback()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Erro ao commitar saldos: %v", err)
	} else {
		log.Println("Saldos iniciais atualizados com sucesso!")
	}
}
