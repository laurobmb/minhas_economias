package pdfgenerator

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png" // Importa para decodificar imagens PNG
	"io"
	"log" // <-- ADICIONADO AQUI
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"minhas_economias/models"
)

// GenerateReportPDF cria um PDF contendo o gráfico e as tabelas do relatório.
// Recebe os dados do relatório, as transações detalhadas e a imagem do gráfico em Base64.
func GenerateReportPDF(reportData []models.RelatorioCategoria, transactions []models.Movimentacao, chartImageBase64 string) (*gofpdf.Fpdf, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// --- Título ---
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Relatorio Financeiro - Minhas Economias")
	pdf.Ln(12)

	// --- Imagem do Gráfico ---
	if chartImageBase64 != "" {
		// Decodifica a string Base64 para dados de imagem
		// O prefixo "data:image/png;base64," é removido
		b64data := chartImageBase64[strings.IndexByte(chartImageBase64, ',')+1:]
		imageData, err := base64.StdEncoding.DecodeString(b64data)
		if err == nil {
			imageReader := bytes.NewReader(imageData)
			// Descobre o formato da imagem (png, jpg, etc.)
			_, format, err := image.DecodeConfig(imageReader)
			if err == nil {
				imageReader.Seek(0, io.SeekStart) // Reseta o leitor para o início
				options := gofpdf.ImageOptions{ImageType: format, ReadDpi: true}
				// Registra a imagem para uso
				pdf.RegisterImageOptionsReader("chart", options, imageReader)
				// Centraliza a imagem na página (largura A4 = 210mm)
				pdf.ImageOptions("chart", (210-150)/2, pdf.GetY(), 150, 0, false, options, 0, "")
				pdf.Ln(85) // Pula o espaço vertical da imagem
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
		pdf.Cell(0, 10, "Resumo de Despesas por Categoria")
		pdf.Ln(8)

		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(240, 240, 240) // Cor de fundo do cabeçalho
		pdf.CellFormat(130, 7, "Categoria", "1", 0, "L", true, 0, "")
		pdf.CellFormat(60, 7, "Total (R$)", "1", 0, "R", true, 0, "")
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 10)
		var granTotal float64
		for _, item := range reportData {
			pdf.CellFormat(130, 7, item.Categoria, "1", 0, "L", false, 0, "")
			pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", item.Total), "1", 0, "R", false, 0, "")
			pdf.Ln(-1)
			granTotal += item.Total
		}
		// Linha de Total
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(130, 7, "TOTAL", "1", 0, "L", true, 0, "")
		pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", granTotal), "1", 0, "R", true, 0, "")
		pdf.Ln(-1)
	}
	pdf.Ln(10)

	// --- Tabela de Transações Detalhadas ---
	if len(transactions) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 10, "Transacoes Detalhadas")
		pdf.Ln(8)

		pdf.SetFont("Arial", "B", 8) // Fonte menor para caber mais colunas
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(20, 7, "Data", "1", 0, "C", true, 0, "")
		pdf.CellFormat(75, 7, "Descricao", "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 7, "Categoria", "1", 0, "L", true, 0, "")
		pdf.CellFormat(35, 7, "Conta", "1", 0, "L", true, 0, "")
		pdf.CellFormat(20, 7, "Valor (R$)", "1", 0, "R", true, 0, "")
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 8)
		for _, tx := range transactions {
			// Trunca a descrição se for muito longa para não quebrar a linha
			desc := tx.Descricao
			if len(desc) > 45 {
				desc = desc[:42] + "..."
			}

			pdf.CellFormat(20, 7, tx.DataOcorrencia, "1", 0, "C", false, 0, "")
			pdf.CellFormat(75, 7, desc, "1", 0, "L", false, 0, "")
			pdf.CellFormat(40, 7, tx.Categoria, "1", 0, "L", false, 0, "")
			pdf.CellFormat(35, 7, tx.Conta, "1", 0, "L", false, 0, "")
			pdf.CellFormat(20, 7, fmt.Sprintf("%.2f", tx.Valor), "1", 0, "R", false, 0, "")
			pdf.Ln(-1)
		}
	}

	// --- Rodapé ---
	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 10, fmt.Sprintf("Gerado em %s", time.Now().Format("02/01/2006 15:04")), "", 0, "C", false, 0, "")

	return pdf, nil
}