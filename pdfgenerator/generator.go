package pdfgenerator

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"minhas_economias/models"
	// A importação de "golang.org/x/text/" foi removida, pois a função toCP1252 não é mais necessária.
)

const (
	headerHeight     = 7.0
	rowHeight        = 7.0
	zebraColorR      = 245
	zebraColorG      = 245
	zebraColorB      = 245
	margin           = 10.0
	logoWidth        = 20.0
	logoHeight       = 15.0
	mainTitleFontSize    = 16.0
	subtitleFontSize = 11.0
	fontName         = "RedHatText"
)

// A função toCP1252 foi REMOVIDA para evitar conflito de codificação de caracteres.

// headerFunc define a função que será usada como cabeçalho em todas as páginas.
func headerFunc(pdf *gofpdf.Fpdf) {
	logoFile := "static/minhaseconomias.png"
	textLeftMargin := margin + logoWidth + 4

	pdf.Image(logoFile, margin, 10, logoWidth, 0, false, "", 0, "")

	pdf.SetXY(textLeftMargin, 11)
	pdf.SetFont(fontName, "B", mainTitleFontSize)
	pdf.SetTextColor(40, 40, 40)
	pdf.Cell(0, 8, "Minhas Economias") // Texto passado diretamente em UTF-8

	pdf.SetXY(textLeftMargin, 18)
	pdf.SetFont(fontName, "", subtitleFontSize)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 8, "Relatório Financeiro") // Texto passado diretamente em UTF-8

	lineY := 10 + logoHeight + 4
	pdf.SetDrawColor(220, 220, 220)
	pdf.Line(margin, lineY, 210-margin, lineY)

	pdf.SetY(lineY + 5)
}

// footerFunc define a função que será usada como rodapé em todas as páginas.
func footerFunc(pdf *gofpdf.Fpdf) {
	pdf.SetY(-15)
	pdf.SetFont(fontName, "I", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(0, 10, fmt.Sprintf("Página %d de {nb}", pdf.PageNo()), "", 0, "R", false, 0, "") // Texto passado diretamente
}

// drawChart insere a imagem do gráfico no PDF.
func drawChart(pdf *gofpdf.Fpdf, chartImageBase64 string) {
	if chartImageBase64 == "" {
		return
	}
	b64data := chartImageBase64[strings.IndexByte(chartImageBase64, ',')+1:]
	imageData, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		log.Printf("Erro ao decodificar string base64 da imagem: %v", err)
		return
	}

	imageReader := bytes.NewReader(imageData)
	_, format, err := image.DecodeConfig(imageReader)
	if err != nil {
		log.Printf("Erro ao decodificar config da imagem: %v", err)
		return
	}
	imageReader.Seek(0, io.SeekStart)
	pageWidth, _ := pdf.GetPageSize()
	imageWidth := 160.0
	x := (pageWidth - imageWidth) / 2
	options := gofpdf.ImageOptions{ImageType: format, ReadDpi: true}
	pdf.RegisterImageOptionsReader("chart", options, imageReader)
	pdf.ImageOptions("chart", x, pdf.GetY(), imageWidth, 0, false, options, 0, "")
	pdf.Ln(105)
}

// drawTable desenha a tabela com um estilo moderno.
func drawTable(pdf *gofpdf.Fpdf, headers []string, data [][]string, colWidths []float64) {
	_, pageHeight := pdf.GetPageSize()
	var tableWidth float64
	for _, w := range colWidths {
		tableWidth += w
	}

	drawHeader := func() {
		pdf.SetFillColor(40, 50, 60)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont(fontName, "B", 9)
		pdf.SetX(margin)
		for i, header := range headers {
			pdf.CellFormat(colWidths[i], headerHeight, header, "1", 0, "C", true, 0, "") // Texto passado diretamente
		}
		pdf.Ln(headerHeight)
	}

	drawHeader()

	pdf.SetFont(fontName, "", 8)
	pdf.SetTextColor(50, 50, 50)

	for i, row := range data {
		if pdf.GetY()+rowHeight > pageHeight-20 {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont(fontName, "", 8)
			pdf.SetTextColor(50, 50, 50)
		}
		
		fill := i%2 != 0
		if fill {
			pdf.SetFillColor(zebraColorR, zebraColorG, zebraColorB)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetX(margin)
		
		for j, cell := range row {
			align := "L"
			if j == len(row)-1 {
				align = "R"
			}
			pdf.CellFormat(colWidths[j], rowHeight, cell, "B", 0, align, true, 0, "") // Texto passado diretamente
		}
		pdf.Ln(rowHeight)
	}
}

// GenerateReportPDF é a função principal que monta o PDF.
func GenerateReportPDF(reportData []models.RelatorioCategoria, transactions []models.Movimentacao, chartImageBase64 string) (*gofpdf.Fpdf, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")

	// --- REGISTRO DA FONTE UTF-8 ---
	// Certifique-se de que os nomes dos arquivos .ttf na pasta 'fonts/' estão corretos.
	pdf.AddUTF8Font(fontName, "", "fonts/RedHatText-Regular.ttf")
	pdf.AddUTF8Font(fontName, "I", "fonts/RedHatText-Italic.ttf")
	// CORREÇÃO: Usando o arquivo da fonte em negrito para o estilo "B".
	pdf.AddUTF8Font(fontName, "B", "fonts/RedHatText-Bold.ttf")
	
	pdf.SetHeaderFunc(func() { headerFunc(pdf) })
	pdf.SetFooterFunc(func() { footerFunc(pdf) })

	pdf.SetTitle("Relatório Financeiro", true)
	pdf.SetAuthor("Minhas Economias", true)
	pdf.AliasNbPages("{nb}")
	pdf.SetMargins(margin, 15, margin)
	pdf.AddPage()

	if chartImageBase64 != "" {
		drawChart(pdf, chartImageBase64)
	} else {
		pdf.Ln(10)
	}

	if len(transactions) > 0 {
		pdf.SetFont(fontName, "B", 12)
		pdf.SetTextColor(50, 50, 50)
		pdf.Cell(0, 8, "Transações Detalhadas")
		pdf.Ln(10)

		headers := []string{"Data", "Descrição", "Categoria", "Conta", "Valor (R$)"}
		colWidths := []float64{25.0, 80.0, 30.0, 30.0, 25.0}
		var data [][]string
		for _, tx := range transactions {
			data = append(data, []string{
				tx.DataOcorrencia, tx.Descricao, tx.Categoria, tx.Conta, fmt.Sprintf("%.2f", tx.Valor),
			})
		}
		drawTable(pdf, headers, data, colWidths)
	}

	if len(reportData) > 0 {
		pdf.Ln(10)
		pdf.SetFont(fontName, "B", 12)
		pdf.SetTextColor(50, 50, 50)
		pdf.Cell(0, 8, "Resumo de Despesas por Categoria")
		pdf.Ln(10)

		summaryHeaders := []string{"Categoria", "Total (R$)"}
		summaryColWidths := []float64{160.0, 30.0}
		var summaryData [][]string
		var granTotal float64
		for _, item := range reportData {
			summaryData = append(summaryData, []string{item.Categoria, fmt.Sprintf("%.2f", item.Total)})
			granTotal += item.Total
		}
		summaryData = append(summaryData, []string{"TOTAL GERAL", fmt.Sprintf("%.2f", granTotal)})

		drawTable(pdf, summaryHeaders, summaryData, summaryColWidths)
	}

	return pdf, pdf.Error()
}