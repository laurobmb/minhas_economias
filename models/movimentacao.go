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
	Consolidado    bool    `json:"consolidado"` // Nova coluna
}
