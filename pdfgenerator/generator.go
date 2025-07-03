package pdfgenerator

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png" // Importa para decodificar imagens PNG
	"io"
	"log"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"minhas_economias/models"
	// --- ALTERAÇÃO: Imports adicionados para a conversão manual de caracteres ---
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// --- ALTERAÇÃO: Nova função auxiliar para converter texto para a codificação correta ---
func toCP1252(str string) string {
	// Converte uma string de UTF-8 para a codificação Windows-1252 (CP1252)
	transformer := charmap.Windows1252.NewEncoder().Transformer
	out, _, err := transform.String(transformer, str)
	if err != nil {
		// Em caso de erro, retorna a string original para não quebrar a aplicação
		return str
	}
	return out
}

// GenerateReportPDF cria um PDF contendo o gráfico e as tabelas do relatório.
func GenerateReportPDF(reportData []models.RelatorioCategoria, transactions []models.Movimentacao, chartImageBase64 string) (*gofpdf.Fpdf, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// A linha 'pdf.SetFontTranslator' foi removida.

	// --- Título ---
	pdf.SetFont("Arial", "B", 16)
	// --- ALTERAÇÃO: Usando a função de conversão ---
	pdf.Cell(0, 10, toCP1252("Relatório Financeiro - Minhas Economias"))
	pdf.Ln(12)

	// --- Imagem do Gráfico ---
	if chartImageBase64 != "" {
		b64data := chartImageBase64[strings.IndexByte(chartImageBase64, ',')+1:]
		imageData, err := base64.StdEncoding.DecodeString(b64data)
		if err == nil {
			imageReader := bytes.NewReader(imageData)
			_, format, err := image.DecodeConfig(imageReader)
			if err == nil {
				imageReader.Seek(0, io.SeekStart)
				options := gofpdf.ImageOptions{ImageType: format, ReadDpi: true}
				pdf.RegisterImageOptionsReader("chart", options, imageReader)
				pdf.ImageOptions("chart", (210-150)/2, pdf.GetY(), 150, 0, false, options, 0, "")
				pdf.Ln(85)
			} else {
				log.Printf("Erro ao decodificar config da imagem: %v", err)
			}
		} else {
			log.Printf("Erro ao decodificar string base64 da imagem: %v", err)
		}
	}

	// --- Tabela Resumo (Dados do Gráfico) ---
	if len(reportData) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 10, toCP1252("Resumo de Despesas por Categoria"))
		pdf.Ln(8)

		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(130, 7, toCP1252("Categoria"), "1", 0, "L", true, 0, "")
		pdf.CellFormat(60, 7, toCP1252("Total (R$)"), "1", 0, "R", true, 0, "")
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 10)
		var granTotal float64
		for _, item := range reportData {
			pdf.CellFormat(130, 7, toCP1252(item.Categoria), "1", 0, "L", false, 0, "")
			pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", item.Total), "1", 0, "R", false, 0, "")
			pdf.Ln(-1)
			granTotal += item.Total
		}
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(130, 7, "TOTAL", "1", 0, "L", true, 0, "")
		pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", granTotal), "1", 0, "R", true, 0, "")
		pdf.Ln(-1)
	}
	pdf.Ln(10)

	// --- Tabela de Transações Detalhadas ---
	if len(transactions) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 10, toCP1252("Transações Detalhadas"))
		pdf.Ln(8)

		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(20, 7, "Data", "1", 0, "C", true, 0, "")
		pdf.CellFormat(75, 7, toCP1252("Descrição"), "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 7, toCP1252("Categoria"), "1", 0, "L", true, 0, "")
		pdf.CellFormat(35, 7, "Conta", "1", 0, "L", true, 0, "")
		pdf.CellFormat(20, 7, toCP1252("Valor (R$)"), "1", 0, "R", true, 0, "")
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 8)
		for _, tx := range transactions {
			desc := tx.Descricao
			if len(desc) > 45 {
				desc = desc[:42] + "..."
			}
			pdf.CellFormat(20, 7, toCP1252(tx.DataOcorrencia), "1", 0, "C", false, 0, "")
			pdf.CellFormat(75, 7, toCP1252(desc), "1", 0, "L", false, 0, "")
			pdf.CellFormat(40, 7, toCP1252(tx.Categoria), "1", 0, "L", false, 0, "")
			pdf.CellFormat(35, 7, toCP1252(tx.Conta), "1", 0, "L", false, 0, "")
			pdf.CellFormat(20, 7, fmt.Sprintf("%.2f", tx.Valor), "1", 0, "R", false, 0, "")
			pdf.Ln(-1)
		}
	}

	// --- Rodapé ---
	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 10, toCP1252(fmt.Sprintf("Gerado em %s", time.Now().Format("02/01/2006 15:04"))), "", 0, "C", false, 0, "")

	return pdf, nil
}