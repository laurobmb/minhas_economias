package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shakinm/xlsReader/xls"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const (
	csvDelimiter = ';'
	// Constante para corrigir o bug de integer overflow do Excel para números negativos
	magicOffset = 10737418.24
)

// toUTF8 converte Windows-1252 para UTF-8
func toUTF8(input string) string {
	decoder := charmap.Windows1252.NewDecoder()
	reader := transform.NewReader(strings.NewReader(input), decoder)
	output, err := io.ReadAll(reader)
	if err != nil {
		return input
	}
	return string(output)
}

// corrigirValorBugado aplica a correção matemática se o número for resultado de overflow
func corrigirValorBugado(valorStr string) string {
	// Normaliza para ponto flutuante (troca vírgula por ponto se houver)
	valClean := strings.Replace(valorStr, ",", ".", -1)
	val, err := strconv.ParseFloat(valClean, 64)
	if err != nil {
		return valorStr // Não é número, retorna original
	}

	// Se o valor estiver na faixa do bug (aprox 10.7 milhões)
	if val > 10000000 && val < 11000000 {
		// Aplica a correção: Valor Lido - Magic Offset = Valor Real Negativo
		valCorrigido := val - magicOffset
		// Retorna formatado com 2 casas decimais
		return fmt.Sprintf("%.2f", valCorrigido)
	}

	return valorStr
}

func main() {
	// Configuração via flags para flexibilidade
	inputDir := flag.String("input", "xls", "Diretório contendo os arquivos .xls")
	outputDir := flag.String("output", "csv", "Diretório de saída para os .csv")
	flag.Parse()

	// Cria diretório de saída
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Erro ao criar diretório de saída '%s': %v", *outputDir, err)
	}

	// Busca automática de arquivos (Substitui a lista hardcoded)
	pattern := filepath.Join(*inputDir, "*.xls")
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatalf("Erro ao buscar arquivos: %v", err)
	}

	if len(files) == 0 {
		log.Printf("Nenhum arquivo .xls encontrado em '%s'.", *inputDir)
		return
	}

	log.Printf("Iniciando processamento de %d arquivos...", len(files))

	for _, filePath := range files {
		fmt.Printf("-> Processando: %s... ", filepath.Base(filePath))
		err := convertXlsToCSV(filePath, *outputDir)
		if err != nil {
			fmt.Printf("ERRO: %v\n", err)
		} else {
			fmt.Printf("OK\n")
		}
	}
}

func convertXlsToCSV(xlsFilePath, outputDir string) error {
	baseName := strings.TrimSuffix(filepath.Base(xlsFilePath), filepath.Ext(xlsFilePath))
	csvFileName := baseName + ".csv"
	csvFilePath := filepath.Join(outputDir, csvFileName)

	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		return fmt.Errorf("falha ao criar CSV: %w", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	writer.Comma = rune(csvDelimiter)
	defer writer.Flush()

	workbook, err := xls.OpenFile(xlsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("arquivo não encontrado")
		}
		return fmt.Errorf("falha ao abrir XLS: %w", err)
	}

	sheet, err := workbook.GetSheet(0)
	if err != nil {
		return fmt.Errorf("falha ao obter planilha: %w", err)
	}

	headerRow, err := sheet.GetRow(0)
	if err != nil {
		return fmt.Errorf("falha ler cabeçalho: %w", err)
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
				utf8String := toUTF8(originalString)

				// --- CORREÇÃO DO BUG DO VALOR ---
				// Assumindo que a coluna de Valor é a 3ª coluna (índice 2)
				// Layout: Data(0); Descricao(1); Valor(2); Categoria(3)...
				if j == 2 {
					utf8String = corrigirValorBugado(utf8String)
				}
				// --------------------------------

				rowValues = append(rowValues, utf8String)
			} else {
				rowValues = append(rowValues, "")
			}
		}

		var finalRow []string
		if i == 0 {
			// Cabeçalho
			finalRow = append(rowValues, "Consolidado")
		} else {
			// Dados: Tenta formatar a data na primeira coluna
			dateValue := rowValues[0]
			if floatVal, err := strconv.ParseFloat(dateValue, 64); err == nil {
				if t, err := excelize.ExcelDateToTime(floatVal, false); err == nil {
					rowValues[0] = t.Format("02/01/2006")
				}
			}
			finalRow = append(rowValues, "true")
		}

		if err := writer.Write(finalRow); err != nil {
			return fmt.Errorf("falha escrita CSV: %w", err)
		}
	}

	return nil
}