package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"minhas_economias/database"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func runExport(db *sql.DB, outputFilename string, userId int64) {
	if err := exportToCSV(db, outputFilename, userId); err != nil {
		log.Printf("Erro na exportação: %v", err)
		os.Exit(1)
	}
	log.Printf("Dados exportados com sucesso para '%s'.\n", outputFilename)
}

func exportToCSV(db *sql.DB, outputFilename string, userId int64) error {
	if outputDir := filepath.Dir(outputFilename); outputDir != "" && outputDir != "." {
		os.MkdirAll(outputDir, 0755)
	}

	file, err := os.Create(outputFilename)
	if err != nil { return err }
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = rune(csvDelimiter)
	defer writer.Flush()

	writer.Write([]string{"Data Ocorrência", "Descrição", "Valor", "Categoria", "Conta", "Consolidado"})

	query := fmt.Sprintf("SELECT data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ? ORDER BY data_ocorrencia ASC", tableName)
	rows, err := db.Query(database.Rebind(query), userId)
	if err != nil { return err }
	defer rows.Close()

	count := 0
	for rows.Next() {
		var mov MovimentacaoAux
		var rawDate interface{}

		if err := rows.Scan(&rawDate, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			continue
		}

		var dataFmt string
		if t, ok := rawDate.(time.Time); ok {
			dataFmt = t.Format("02/01/2006")
		} else if s, ok := rawDate.(string); ok {
			p, _ := time.Parse("2006-01-02", s)
			dataFmt = p.Format("02/01/2006")
		}

		valFmt := strings.Replace(strconv.FormatFloat(mov.Valor, 'f', 2, 64), ".", ",", -1)
		writer.Write([]string{dataFmt, mov.Descricao, valFmt, mov.Categoria, mov.Conta, strconv.FormatBool(mov.Consolidado)})
		count++
	}
	log.Printf("   %d registros exportados.", count)
	return nil
}