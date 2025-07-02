package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shakinm/xlsReader/xls"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const csvDelimiter = ';'

// Esta função converte uma string da codificação Windows-1252 para UTF-8.
func toUTF8(input string) string {
	// Cria um "tradutor" que sabe como decodificar Windows-1252
	decoder := charmap.Windows1252.NewDecoder()
	// Cria um leitor que aplica essa tradução
	reader := transform.NewReader(strings.NewReader(input), decoder)
	// Lê o resultado traduzido
	output, err := io.ReadAll(reader)
	if err != nil {
		// Em caso de erro, retorna a string original para não quebrar o programa
		return input
	}
	return string(output)
}

func main() {
	// A lista de arquivos a processar permanece a mesma
	filesToProcess := []string{
		// "xls/example.xls",
		"xls/Extrato_20130101_20131231.xls",
		"xls/Extrato_20140101_20141231.xls",
		"xls/Extrato_20150101_20151231.xls",
		"xls/Extrato_20160101_20161231.xls",
		"xls/Extrato_20170101_20171231.xls",
		"xls/Extrato_20180101_20181231.xls",
		"xls/Extrato_20190101_20191231.xls",
		"xls/Extrato_20200101_20201231.xls",
		"xls/Extrato_20210101_20211231.xls",
		"xls/Extrato_20220101_20221231.xls",
		"xls/Extrato_20230101_20231231.xls",
		"xls/Extrato_20240101_20241231.xls",
		"xls/Extrato_20250101_20251231.xls",
	}

	outputDir := "csv"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Erro ao criar o diretório de saída '%s': %v\n", outputDir, err)
		return
	}

	for _, filePath := range filesToProcess {
		fmt.Printf("Processando arquivo: %s\n", filePath)
		err := convertXlsToCSV(filePath, outputDir)
		if err != nil {
			fmt.Printf("Erro ao converter XLS '%s': %v\n", filePath, err)
		}
	}
}

func convertXlsToCSV(xlsFilePath, outputDir string) error {
	baseName := strings.TrimSuffix(filepath.Base(xlsFilePath), filepath.Ext(xlsFilePath))
	csvFileName := baseName + ".csv"
	csvFilePath := filepath.Join(outputDir, csvFileName)

	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		return fmt.Errorf("falha ao criar o arquivo CSV de saída: %w", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	writer.Comma = rune(csvDelimiter)
	defer writer.Flush()

	workbook, err := xls.OpenFile(xlsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("arquivo não encontrado: %s", xlsFilePath)
		}
		return fmt.Errorf("falha ao abrir o arquivo XLS: %w", err)
	}

	sheet, err := workbook.GetSheet(0)
	if err != nil {
		return fmt.Errorf("falha ao obter planilha: %w", err)
	}

	headerRow, err := sheet.GetRow(0)
	if err != nil {
		return fmt.Errorf("falha ao ler a linha do cabeçalho: %w", err)
	}
	numCols := len(headerRow.GetCols())

	for i := 0; i <= sheet.GetNumberRows()-1; i++ {
		row, err := sheet.GetRow(i)
		if err != nil || row == nil {
			continue
		}

		var rowValues []string
		cols := row.GetCols()
		
		for j := 0; j < numCols; j++ {
			if j < len(cols) {
				cell := cols[j]
				originalString := cell.GetString()
				// --- APLICA A CONVERSÃO PARA CADA CÉLULA ---
				utf8String := toUTF8(originalString)
				rowValues = append(rowValues, utf8String)
			} else {
				rowValues = append(rowValues, "")
			}
		}

		var finalRow []string
		if i == 0 {
			finalRow = append(rowValues, "Consolidado")
		} else {
			dateValue := rowValues[0]
			if floatVal, err := strconv.ParseFloat(dateValue, 64); err == nil {
				if t, err := excelize.ExcelDateToTime(floatVal, false); err == nil {
					rowValues[0] = t.Format("02/01/2006")
				}
			}
			finalRow = append(rowValues, "true")
		}

		if err := writer.Write(finalRow); err != nil {
			return fmt.Errorf("falha ao escrever linha no CSV: %w", err)
		}
	}

	fmt.Printf("✔ Convertido com sucesso '%s' para '%s'.\n", xlsFilePath, csvFilePath)
	return nil
}