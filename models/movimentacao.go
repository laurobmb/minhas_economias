// models/movimentacao.go
package models

// Movimentacao representa uma linha na tabela 'movimentacoes'
type Movimentacao struct {
	ID             int     `json:"id"`
	DataOcorrencia string  `json:"data_ocorrencia"`
	Descricao      string  `json:"descricao"`
	Valor          float64 `json:"valor"`
	Categoria      string  `json:"categoria"`
	Conta          string  `json:"conta"`
	Consolidado    bool    `json:"consolidado"`
}

// --- STRUCTS MOVIDOS PARA CÁ PARA QUEBRAR O CICLO DE IMPORTAÇÃO ---

// RelatorioCategoria representa o total de despesas por categoria.
type RelatorioCategoria struct {
	Categoria string  `json:"categoria"`
	Total     float64 `json:"total"`
}

// PDFRequestPayload é o struct para receber os dados do frontend para gerar o PDF.
type PDFRequestPayload struct {
	StartDate        string   `json:"start_date"`
	EndDate          string   `json:"end_date"`
	Categories       []string `json:"categories"`
	Accounts         []string `json:"accounts"`
	ConsolidatedFilter string `json:"consolidated_filter"`
	ChartImageBase64 string   `json:"chartImageBase64"`
}