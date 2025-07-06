package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url" // <-- IMPORT ADICIONADO
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"minhas_economias/database"
	"minhas_economias/models"
	"minhas_economias/pdfgenerator"
)

// GetIndexPage renderiza a página inicial apenas com os saldos.
func GetIndexPage(c *gin.Context) {
	saldosContas, err := calculateAccountBalances()
	if err != nil {
		log.Printf("ERRO ao calcular saldos para a página inicial: %v", err)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Titulo":       "Minhas Economias - Saldos",
			"SaldosContas": nil,
			"Error":        "Não foi possível carregar os saldos.",
		})
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Titulo":       "Minhas Economias - Saldos",
		"SaldosContas": saldosContas,
	})
}

// calculateAccountBalances calcula o saldo final de cada conta.
func calculateAccountBalances() ([]models.ContaSaldo, error) {
	db := database.GetDB()
	saldos := make(map[string]float64)

	rowsContas, err := db.Query("SELECT nome, saldo_inicial FROM contas")
	if err == nil {
		defer rowsContas.Close()
		for rowsContas.Next() {
			var nome string
			var saldoInicial float64
			if err := rowsContas.Scan(&nome, &saldoInicial); err == nil {
				saldos[nome] = saldoInicial
			}
		}
	} else {
		log.Printf("Aviso: Não foi possível ler a tabela 'contas' para obter saldos iniciais: %v. Assumindo saldo inicial 0 para todas.", err)
	}

	rowsMov, err := db.Query(fmt.Sprintf("SELECT conta, SUM(valor) FROM %s GROUP BY conta", database.TableName))
	if err != nil {
		return nil, fmt.Errorf("erro ao calcular totais das movimentações por conta: %w", err)
	}
	defer rowsMov.Close()

	for rowsMov.Next() {
		var conta string
		var totalMovimentacoes float64
		if err := rowsMov.Scan(&conta, &totalMovimentacoes); err != nil {
			log.Printf("Erro ao escanear total da conta: %v", err)
			continue
		}
		saldos[conta] += totalMovimentacoes
	}

	var result []models.ContaSaldo
	for nome, saldo := range saldos {
		// ADICIONADA LÓGICA PARA PREENCHER O NOVO CAMPO
		result = append(result, models.ContaSaldo{
			Nome:           nome,
			SaldoAtual:     saldo,
			URLEncodedNome: url.QueryEscape(nome), // <-- ALTERAÇÃO AQUI
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Nome < result[j].Nome
	})

	return result, nil
}

// ... (O resto do arquivo permanece o mesmo)
// GetTransacoesPage e as outras funções não precisam de alteração.
// ... (cole o resto do seu arquivo handlers/movimentacoes.go aqui para manter as outras funções)
// GetTransacoesPage, AddMovimentacao, DeleteMovimentacao, etc.

// GetTransacoesPage busca os registros de movimentacoes e renderiza a página de transações.
// Também serve a rota da API.
func GetTransacoesPage(c *gin.Context) {
	db := database.GetDB()

	searchDescricao := c.Query("search_descricao")
	selectedCategories := c.QueryArray("category")
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidado := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account")
	selectedValueFilter := c.Query("value_filter")

	if selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")
	}

	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s", database.TableName)
	var args []interface{}
	var whereClauses []string

	if searchDescricao != "" {
		whereClauses = append(whereClauses, "descricao LIKE ?")
		args = append(args, "%"+searchDescricao+"%")
	}
	if len(selectedCategories) > 0 && selectedCategories[0] != "" {
		placeholders := strings.Repeat("?,", len(selectedCategories)-1) + "?"
		whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", placeholders))
		for _, v := range selectedCategories {
			args = append(args, v)
		}
	}
	if len(selectedAccounts) > 0 && selectedAccounts[0] != "" {
		placeholders := strings.Repeat("?,", len(selectedAccounts)-1) + "?"
		whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", placeholders))
		for _, v := range selectedAccounts {
			args = append(args, v)
		}
	}
	if selectedStartDate != "" {
		whereClauses = append(whereClauses, "data_ocorrencia >= ?")
		args = append(args, selectedStartDate)
	}
	if selectedEndDate != "" {
		whereClauses = append(whereClauses, "data_ocorrencia <= ?")
		args = append(args, selectedEndDate)
	}
	if selectedConsolidado != "" {
		if selectedConsolidado == "true" {
			whereClauses = append(whereClauses, "consolidado = 1")
		} else if selectedConsolidado == "false" {
			whereClauses = append(whereClauses, "consolidado = 0")
		}
	}
	if selectedValueFilter != "" {
		if selectedValueFilter == "income" {
			whereClauses = append(whereClauses, "valor >= 0")
		} else if selectedValueFilter == "expense" {
			whereClauses = append(whereClauses, "valor < 0")
		}
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY data_ocorrencia DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao buscar movimentações: %v", err)})
		return
	}
	defer rows.Close()

	var movimentacoes []models.Movimentacao
	var totalValor, totalEntradas, totalSaidas float64

	for rows.Next() {
		var mov models.Movimentacao
		if err := rows.Scan(&mov.ID, &mov.DataOcorrencia, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			log.Printf("Erro ao escanear linha da movimentação: %v", err)
			continue
		}
		movimentacoes = append(movimentacoes, mov)
		totalValor += mov.Valor
		if mov.Valor >= 0 {
			totalEntradas += mov.Valor
		} else {
			totalSaidas += mov.Valor
		}
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro na iteração das linhas de movimentação: %v", err)})
		return
	}

	// Lógica para endpoint de API
	if c.Request.URL.Path == "/api/movimentacoes" {
		c.JSON(http.StatusOK, gin.H{
			"movimentacoes": movimentacoes,
			"totalValor":    totalValor,
			"totalEntradas": totalEntradas,
			"totalSaidas":   totalSaidas,
		})
		return
	}
	
	// Busca dados para os filtros da página de transações
	categoryRows, _ := db.Query(fmt.Sprintf("SELECT DISTINCT categoria FROM %s ORDER BY categoria ASC", database.TableName))
	var categories []string
	if categoryRows != nil {
		defer categoryRows.Close()
		for categoryRows.Next() {
			var cat string
			if err := categoryRows.Scan(&cat); err == nil {
				categories = append(categories, cat)
			}
		}
	}

	accountRows, _ := db.Query(fmt.Sprintf("SELECT DISTINCT conta FROM %s ORDER BY conta ASC", database.TableName))
	var accounts []string
	if accountRows != nil {
		defer accountRows.Close()
		for accountRows.Next() {
			var acc string
			if err := accountRows.Scan(&acc); err == nil {
				accounts = append(accounts, acc)
			}
		}
	}

	consolidatedOptions := []struct{ Value, Label string }{{"", "Todos"}, {"true", "Sim"}, {"false", "Não"}}
	currentDate := time.Now().Format("2006-01-02")
	
	c.HTML(http.StatusOK, "transacoes.html", gin.H{
		"Movimentacoes":        movimentacoes,
		"Titulo":               "Transações Financeiras",
		"SearchDescricao":      searchDescricao,
		"SelectedCategories":   selectedCategories,
		"SelectedStartDate":    selectedStartDate,
		"SelectedEndDate":      selectedEndDate,
		"SelectedConsolidado":  selectedConsolidado,
		"SelectedAccounts":     selectedAccounts,
		"SelectedValueFilter":  selectedValueFilter,
		"Categories":           categories,
		"Accounts":             accounts,
		"ConsolidatedOptions":  consolidatedOptions,
		"TotalValor":           totalValor,
		"TotalEntradas":        totalEntradas,
		"TotalSaidas":          totalSaidas,
		"CurrentDate":          currentDate,
	})
}

// AddMovimentacao adiciona uma nova transação
func AddMovimentacao(c *gin.Context) {
	db := database.GetDB()
	dataOcorrencia := c.PostForm("data_ocorrencia")
	descricao := c.PostForm("descricao")
	valorStr := c.PostForm("valor")
	categoria := c.PostForm("categoria")
	conta := c.PostForm("conta")
	consolidadoStr := c.PostForm("consolidado")

	if strings.TrimSpace(categoria) == "" {
		categoria = "Sem Categoria"
	}
	if strings.TrimSpace(conta) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "O campo 'Conta' é obrigatório."})
		return
	}
	var valor float64
	if strings.TrimSpace(valorStr) == "" {
		valor = 0.0
	} else {
		valorParseable := strings.Replace(valorStr, ",", ".", -1)
		parsedValor, err := strconv.ParseFloat(valorParseable, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Valor inválido: formato numérico incorreto."})
			return
		}
		valor = parsedValor
	}
	consolidado := (consolidadoStr == "on")
	stmt, err := db.Prepare(fmt.Sprintf(
		`INSERT INTO %s (data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?)`, database.TableName))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno do servidor."})
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(dataOcorrencia, descricao, valor, categoria, conta, consolidado)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao inserir dados: " + err.Error()})
		return
	}
    // Redireciona para a página de transações após adicionar
	c.Redirect(http.StatusFound, "/transacoes") 
}

// DeleteMovimentacao deleta uma transação
func DeleteMovimentacao(c *gin.Context) {
	db := database.GetDB()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido."})
		return
	}
	_, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", database.TableName), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar a movimentação."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Movimentação deletada com sucesso!"})
}

// UpdateMovimentacao atualiza uma transação
func UpdateMovimentacao(c *gin.Context) {
	db := database.GetDB()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido."})
		return
	}
	dataOcorrencia := c.PostForm("data_ocorrencia")
	descricao := c.PostForm("descricao")
	valorStr := c.PostForm("valor")
	categoria := c.PostForm("categoria")
	conta := c.PostForm("conta")
	consolidadoStr := c.PostForm("consolidado")
	if strings.TrimSpace(categoria) == "" {
		categoria = "Sem Categoria"
	}
	if strings.TrimSpace(conta) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "O campo 'Conta' é obrigatório."})
		return
	}
	var valor float64
	if strings.TrimSpace(valorStr) == "" {
		valor = 0.0
	} else {
		valorParseable := strings.Replace(valorStr, ",", ".", -1)
		parsedValor, err := strconv.ParseFloat(valorParseable, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Valor inválido: formato numérico incorreto."})
			return
		}
		valor = parsedValor
	}
	consolidado := (consolidadoStr == "on")
	stmt, err := db.Prepare(fmt.Sprintf(
		`UPDATE %s SET data_ocorrencia = ?, descricao = ?, valor = ?, categoria = ?, conta = ?, consolidado = ? WHERE id = ?`, database.TableName))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno do servidor."})
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(dataOcorrencia, descricao, valor, categoria, conta, consolidado, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar dados: " + err.Error()})
		return
	}
    // Redireciona para a página de transações após atualizar
	c.Redirect(http.StatusFound, "/transacoes")
}

// GetRelatorio renderiza a página de relatório
func GetRelatorio(c *gin.Context) {
	searchDescricao := c.Query("search_descricao")
	selectedCategories := c.QueryArray("category")
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidado := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account")

	if selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")
	}

	relatorioData, err := fetchReportData(selectedStartDate, selectedEndDate, selectedCategories, selectedAccounts, selectedConsolidado, searchDescricao)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao buscar dados para relatório: %v", err)})
		return
	}

	db := database.GetDB()
	categoryRows, _ := db.Query(fmt.Sprintf("SELECT DISTINCT categoria FROM %s ORDER BY categoria ASC", database.TableName))
	var categories []string
	if categoryRows != nil {
		defer categoryRows.Close()
		for categoryRows.Next() {
			var cat string
			if err := categoryRows.Scan(&cat); err == nil {
				categories = append(categories, cat)
			}
		}
	}
	accountRows, _ := db.Query(fmt.Sprintf("SELECT DISTINCT conta FROM %s ORDER BY conta ASC", database.TableName))
	var accounts []string
	if accountRows != nil {
		defer accountRows.Close()
		for accountRows.Next() {
			var acc string
			if err := accountRows.Scan(&acc); err == nil {
				accounts = append(accounts, acc)
			}
		}
	}

	c.HTML(http.StatusOK, "relatorio.html", gin.H{
		"Titulo":               "Relatório de Despesas por Categoria",
		"ReportData":           relatorioData,
		"SearchDescricao":      searchDescricao,
		"SelectedCategories":   selectedCategories,
		"SelectedStartDate":    selectedStartDate,
		"SelectedEndDate":      selectedEndDate,
		"SelectedConsolidated": selectedConsolidado,
		"SelectedAccounts":     selectedAccounts,
		"Categories":           categories,
		"Accounts":             accounts,
		"ConsolidatedOptions":  []struct{ Value, Label string }{{"", "Todos"}, {"true", "Sim"}, {"false", "Não"}},
		"CurrentDate":          time.Now().Format("2006-01-02"),
	})
}

// GetTransactionsByCategory busca transações para o gráfico de relatório
func GetTransactionsByCategory(c *gin.Context) {
	db := database.GetDB()
	category := c.Query("category")
	searchDescricao := c.Query("search_descricao")
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidado := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account")

	if selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")
	}

	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s", database.TableName)
	var args []interface{}
	var whereClauses []string

	whereClauses = append(whereClauses, "categoria = ?")
	args = append(args, category)
	whereClauses = append(whereClauses, "valor < 0")

	if searchDescricao != "" {
		whereClauses = append(whereClauses, "descricao LIKE ?")
		args = append(args, "%"+searchDescricao+"%")
	}
	if len(selectedAccounts) > 0 && selectedAccounts[0] != "" {
		placeholders := strings.Repeat("?,", len(selectedAccounts)-1) + "?"
		whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", placeholders))
		for _, v := range selectedAccounts {
			args = append(args, v)
		}
	}
	if selectedStartDate != "" {
		whereClauses = append(whereClauses, "data_ocorrencia >= ?")
		args = append(args, selectedStartDate)
	}
	if selectedEndDate != "" {
		whereClauses = append(whereClauses, "data_ocorrencia <= ?")
		args = append(args, selectedEndDate)
	}
	if selectedConsolidado != "" {
		if selectedConsolidado == "true" {
			whereClauses = append(whereClauses, "consolidado = 1")
		} else if selectedConsolidado == "false" {
			whereClauses = append(whereClauses, "consolidado = 0")
		}
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY data_ocorrencia DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar transações: " + err.Error()})
		return
	}
	defer rows.Close()

	var transactions []models.Movimentacao
	for rows.Next() {
		var mov models.Movimentacao
		if err := rows.Scan(&mov.ID, &mov.DataOcorrencia, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			log.Printf("Erro ao escanear transação: %v", err)
			continue
		}
		transactions = append(transactions, mov)
	}
	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro na iteração das transações: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// DownloadRelatorioPDF gera o relatório em PDF
func DownloadRelatorioPDF(c *gin.Context) {
	var payload models.PDFRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	reportData, err := fetchReportData(payload.StartDate, payload.EndDate, payload.Categories, payload.Accounts, payload.ConsolidatedFilter, payload.SearchDescricao)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar dados do relatório: " + err.Error()})
		return
	}
	transactions, err := fetchAllTransactions(payload.StartDate, payload.EndDate, payload.Categories, payload.Accounts, payload.ConsolidatedFilter, payload.SearchDescricao)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar transações detalhadas: " + err.Error()})
		return
	}

	pdf, err := pdfgenerator.GenerateReportPDF(reportData, transactions, payload.ChartImageBase64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar PDF: " + err.Error()})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="relatorio_financeiro.pdf"`)

	if err := pdf.Output(c.Writer); err != nil {
		log.Printf("Erro ao enviar PDF para o cliente: %v", err)
	}
}

// fetchReportData busca dados para o relatório.
func fetchReportData(startDate, endDate string, categories, accounts []string, consolidated, searchDescricao string) ([]models.RelatorioCategoria, error) {
	db := database.GetDB()
	query := fmt.Sprintf("SELECT categoria, SUM(valor) FROM %s WHERE valor < 0", database.TableName)
	var args []interface{}
	var whereClauses []string

	if searchDescricao != "" {
		whereClauses = append(whereClauses, "descricao LIKE ?")
		args = append(args, "%"+searchDescricao+"%")
	}

	if len(categories) > 0 && categories[0] != "" {
		placeholders := strings.Repeat("?,", len(categories)-1) + "?"
		whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", placeholders))
		for _, v := range categories {
			args = append(args, v)
		}
	}
	if len(accounts) > 0 && accounts[0] != "" {
		placeholders := strings.Repeat("?,", len(accounts)-1) + "?"
		whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", placeholders))
		for _, v := range accounts {
			args = append(args, v)
		}
	}
	if startDate != "" {
		whereClauses = append(whereClauses, "data_ocorrencia >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		whereClauses = append(whereClauses, "data_ocorrencia <= ?")
		args = append(args, endDate)
	}
	if consolidated != "" {
		if consolidated == "true" {
			whereClauses = append(whereClauses, "consolidado = 1")
		} else if consolidated == "false" {
			whereClauses = append(whereClauses, "consolidado = 0")
		}
	}

	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}
	query += " GROUP BY categoria ORDER BY SUM(valor) ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relatorioData []models.RelatorioCategoria
	for rows.Next() {
		var rc models.RelatorioCategoria
		if err := rows.Scan(&rc.Categoria, &rc.Total); err != nil {
			log.Printf("Erro ao escanear linha do relatório: %v", err)
			continue
		}
		relatorioData = append(relatorioData, rc)
	}
	return relatorioData, rows.Err()
}

// fetchAllTransactions busca todas as transações para o relatório.
func fetchAllTransactions(startDate, endDate string, categories, accounts []string, consolidated, searchDescricao string) ([]models.Movimentacao, error) {
	db := database.GetDB()
	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s", database.TableName)
	var args []interface{}
	var whereClauses []string
	whereClauses = append(whereClauses, "valor < 0")

	if searchDescricao != "" {
		whereClauses = append(whereClauses, "descricao LIKE ?")
		args = append(args, "%"+searchDescricao+"%")
	}

	if len(categories) > 0 && categories[0] != "" {
		placeholders := strings.Repeat("?,", len(categories)-1) + "?"
		whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", placeholders))
		for _, v := range categories {
			args = append(args, v)
		}
	}
	if len(accounts) > 0 && accounts[0] != "" {
		placeholders := strings.Repeat("?,", len(accounts)-1) + "?"
		whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", placeholders))
		for _, v := range accounts {
			args = append(args, v)
		}
	}
	if startDate != "" {
		whereClauses = append(whereClauses, "data_ocorrencia >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		whereClauses = append(whereClauses, "data_ocorrencia <= ?")
		args = append(args, endDate)
	}
	if consolidated != "" {
		if consolidated == "true" {
			whereClauses = append(whereClauses, "consolidado = 1")
		} else if consolidated == "false" {
			whereClauses = append(whereClauses, "consolidado = 0")
		}
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY data_ocorrencia DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Movimentacao
	for rows.Next() {
		var mov models.Movimentacao
		if err := rows.Scan(&mov.ID, &mov.DataOcorrencia, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			log.Printf("Erro ao escanear transação: %v", err)
			continue
		}
		transactions = append(transactions, mov)
	}
	return transactions, rows.Err()
}