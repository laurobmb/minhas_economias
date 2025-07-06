// handlers/movimentacoes.go
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort" // <-- IMPORT ADICIONADO
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"minhas_economias/database"
	"minhas_economias/models"
	"minhas_economias/pdfgenerator"
)

// --- NOVA FUNÇÃO ADICIONADA ---
// calculateAccountBalances calcula o saldo final de cada conta.
func calculateAccountBalances() ([]models.ContaSaldo, error) {
	db := database.GetDB()

	// 1. Estrutura para armazenar saldos: Nome da Conta -> Saldo
	saldos := make(map[string]float64)

	// 2. Buscar saldos iniciais da nova tabela 'contas'
	rowsContas, err := db.Query("SELECT nome, saldo_inicial FROM contas")
	if err == nil { // Se a tabela 'contas' existir e a query rodar
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

	// 3. Somar todas as movimentações (entradas e saídas) para cada conta
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

		// Adiciona o total de movimentações ao saldo (que pode ser o inicial ou 0)
		saldos[conta] += totalMovimentacoes
	}

	// 4. Converter o mapa para uma lista (slice) para ser usada no template
	var result []models.ContaSaldo
	for nome, saldo := range saldos {
		result = append(result, models.ContaSaldo{Nome: nome, SaldoAtual: saldo})
	}

	// 5. Ordenar por nome da conta para exibição consistente
	sort.Slice(result, func(i, j int) bool {
		return result[i].Nome < result[j].Nome
	})

	return result, nil
}

// --- FIM DA NOVA FUNÇÃO ---

// GetMovimentacoes busca os registros de movimentacoes do banco de dados, com filtros opcionais
func GetMovimentacoes(c *gin.Context) {
	db := database.GetDB()

	// --- ALTERAÇÃO: Adicionado search_descricao ---
	searchDescricao := c.Query("search_descricao")
	// --- FIM DA ALTERAÇÃO ---
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
		log.Printf("Nenhum filtro de data fornecido. Default para mês corrente: %s a %s\n", selectedStartDate, selectedEndDate)
	}

	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s", database.TableName)
	var args []interface{}
	var whereClauses []string

	// --- ALTERAÇÃO: Adicionada lógica de filtro para descrição ---
	if searchDescricao != "" {
		whereClauses = append(whereClauses, "descricao LIKE ?")
		args = append(args, "%"+searchDescricao+"%")
	}
	// --- FIM DA ALTERAÇÃO ---

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

	// --- ALTERAÇÃO: CHAMADA DA NOVA FUNÇÃO ---
	saldosContas, err := calculateAccountBalances()
	if err != nil {
		// Não quebra a página, apenas loga o erro. A tabela de saldos não será exibida.
		log.Printf("ERRO: Não foi possível calcular os saldos das contas: %v", err)
	}
	// --- FIM DA ALTERAÇÃO ---

	if c.Request.URL.Path == "/api/movimentacoes" {
		c.JSON(http.StatusOK, gin.H{
			"movimentacoes": movimentacoes,
			"totalValor":    totalValor,
			"totalEntradas": totalEntradas,
			"totalSaidas":   totalSaidas,
		})
	} else {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Movimentacoes":        movimentacoes,
			"Titulo":               "Extratos Financeiros",
			// --- ALTERAÇÃO: Passar valor do campo de busca para o template ---
			"SearchDescricao":      searchDescricao,
			// --- FIM DA ALTERAÇÃO ---
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
			// --- ALTERAÇÃO: PASSANDO OS DADOS PARA O TEMPLATE ---
			"SaldosContas":         saldosContas,
			// --- FIM DA ALTERAÇÃO ---
		})
	}
}

// ... (o resto do arquivo 'movimentacoes.go' continua igual)
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
	c.Redirect(http.StatusFound, "/")
}

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
	c.Redirect(http.StatusFound, "/")
}

func GetRelatorio(c *gin.Context) {
	// --- ALTERAÇÃO: Adicionado search_descricao ---
	searchDescricao := c.Query("search_descricao")
	// --- FIM DA ALTERAÇÃO ---
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

	// --- ALTERAÇÃO: Passar searchDescricao para a função de busca ---
	relatorioData, err := fetchReportData(selectedStartDate, selectedEndDate, selectedCategories, selectedAccounts, selectedConsolidado, searchDescricao)
	// --- FIM DA ALTERAÇÃO ---
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
		// --- ALTERAÇÃO: Passar valor do campo de busca para o template ---
		"SearchDescricao":      searchDescricao,
		// --- FIM DA ALTERAÇÃO ---
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

func GetTransactionsByCategory(c *gin.Context) {
	db := database.GetDB()
	category := c.Query("category")
	// --- ALTERAÇÃO: Adicionado search_descricao ---
	searchDescricao := c.Query("search_descricao")
	// --- FIM DA ALTERAÇÃO ---
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

	// Adiciona a categoria clicada como o primeiro e principal filtro.
	whereClauses = append(whereClauses, "categoria = ?")
	args = append(args, category)
	whereClauses = append(whereClauses, "valor < 0")

	// --- ALTERAÇÃO: Adicionar lógica de filtro de descrição ---
	if searchDescricao != "" {
		whereClauses = append(whereClauses, "descricao LIKE ?")
		args = append(args, "%"+searchDescricao+"%")
	}
	// --- FIM DA ALTERAÇÃO ---

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

func DownloadRelatorioPDF(c *gin.Context) {
	var payload models.PDFRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	// --- ALTERAÇÃO: Passar searchDescricao para as funções de busca ---
	reportData, err := fetchReportData(payload.StartDate, payload.EndDate, payload.Categories, payload.Accounts, payload.ConsolidatedFilter, payload.SearchDescricao)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar dados do relatório: " + err.Error()})
		return
	}
	transactions, err := fetchAllTransactions(payload.StartDate, payload.EndDate, payload.Categories, payload.Accounts, payload.ConsolidatedFilter, payload.SearchDescricao)
	// --- FIM DA ALTERAÇÃO ---
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

// --- FUNÇÕES AUXILIARES ---
// --- ALTERAÇÃO: Adicionada a variável searchDescricao ---
func fetchReportData(startDate, endDate string, categories, accounts []string, consolidated, searchDescricao string) ([]models.RelatorioCategoria, error) {
	db := database.GetDB()
	query := fmt.Sprintf("SELECT categoria, SUM(valor) FROM %s WHERE valor < 0", database.TableName)
	var args []interface{}
	var whereClauses []string

	if searchDescricao != "" {
		whereClauses = append(whereClauses, "descricao LIKE ?")
		args = append(args, "%"+searchDescricao+"%")
	}
	// --- FIM DA ALTERAÇÃO ---

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

// --- ALTERAÇÃO: Adicionada a variável searchDescricao ---
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
	// --- FIM DA ALTERAÇÃO ---

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