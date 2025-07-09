package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"minhas_economias/database"
	"minhas_economias/models"
	"minhas_economias/pdfgenerator"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// Helper Functions
// =============================================================================

func bindAndQuery(userID int64, query string, args ...interface{}) (*sql.Rows, error) {
	db := database.GetDB()
	finalArgs := append([]interface{}{userID}, args...)
	reboundQuery := database.Rebind(query)
	return db.Query(reboundQuery, finalArgs...)
}

func bindAndExec(userID int64, query string, args ...interface{}) (sql.Result, error) {
	db := database.GetDB()
	finalArgs := append([]interface{}{userID}, args...)
	reboundQuery := database.Rebind(query)
	return db.Exec(reboundQuery, finalArgs...)
}

func scanDate(rawData interface{}) string {
	if rawData == nil { return "" }
	if t, ok := rawData.(time.Time); ok { return t.Format("2006-01-02") }
	if s, ok := rawData.(string); ok { return s }
	if b, ok := rawData.([]byte); ok { return string(b) }
	return ""
}

func getDistinctColumnValues(userID int64, columnName string) []string {
	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE user_id = ? AND %s <> '' ORDER BY %s ASC", columnName, database.TableName, columnName, columnName)
	rows, err := bindAndQuery(userID, query)
	if err != nil {
		log.Printf("Erro ao buscar valores distintos para a coluna '%s': %v", columnName, err)
		return []string{}
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var val string
		if err := rows.Scan(&val); err == nil {
			values = append(values, val)
		}
	}
	return values
}

func validateMovimentacao(c *gin.Context) (models.Movimentacao, error) {
	var mov models.Movimentacao
	var err error
	mov.DataOcorrencia = c.PostForm("data_ocorrencia")
	mov.Descricao = c.PostForm("descricao")
	valorStr := c.PostForm("valor")
	mov.Categoria = c.PostForm("categoria")
	mov.Conta = c.PostForm("conta")
	mov.Consolidado = (c.PostForm("consolidado") == "on")
	if len(mov.Descricao) > 60 { return mov, fmt.Errorf("A descrição não pode ter mais de 60 caracteres.") }
	if strings.TrimSpace(mov.Categoria) == "" { mov.Categoria = "Sem Categoria" }
	if strings.TrimSpace(mov.Conta) == "" { return mov, fmt.Errorf("O campo 'Conta' é obrigatório.") }
	if strings.TrimSpace(valorStr) == "" {
		mov.Valor = 0.0
	} else {
		valorParseable := strings.Replace(valorStr, ",", ".", -1)
		if isValid, _ := regexp.MatchString(`^-?\d+(\.\d{1,2})?$`, valorParseable); !isValid { return mov, fmt.Errorf("Valor inválido. Use um formato como 1234.56 ou -123.45.") }
		if mov.Valor, err = strconv.ParseFloat(valorParseable, 64); err != nil { return mov, fmt.Errorf("Valor inválido: formato numérico incorreto.") }
		if math.Abs(mov.Valor) >= 100000000 { return mov, fmt.Errorf("O valor excede o limite máximo permitido (100 milhões).") }
	}
	return mov, nil
}

// =============================================================================
// Page Handlers
// =============================================================================

func GetIndexPage(c *gin.Context) {
	log.Println("--- EXECUTANDO HANDLER: GetIndexPage para a rota / ---") 
	userID := c.MustGet("userID").(int64)
	user := c.MustGet("user").(*models.User) // <-- NOVO: Pega o usuário do contexto

	saldosContas, err := calculateAccountBalances(userID)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Não foi possível carregar os saldos das contas.", err)
		return
	}
	c.HTML(http.StatusOK, "index.html", gin.H{
		"Titulo":       "Minhas Economias - Saldos",
		"SaldosContas": saldosContas,
		"User":         user, // <-- NOVO: Passa o usuário para o template
	})
}

func GetTransacoesPage(c *gin.Context) {
	log.Println("--- EXECUTANDO HANDLER: GetTransacoesPage para a rota /transacoes ---") // <-- LOG DE DIAGNÓSTICO
	userID := c.MustGet("userID").(int64)
	user := c.MustGet("user").(*models.User)

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

	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ?", database.TableName)
	var args []interface{}
	var whereClauses []string
	if searchDescricao != "" { clause := "descricao LIKE ?"; if database.DriverName == "postgres" { clause = "descricao ILIKE ?" }; whereClauses = append(whereClauses, clause); args = append(args, "%"+searchDescricao+"%") }
	if len(selectedCategories) > 0 && selectedCategories[0] != "" { placeholders := strings.Repeat("?,", len(selectedCategories)-1) + "?"; whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", placeholders)); for _, v := range selectedCategories { args = append(args, v) } }
	if len(selectedAccounts) > 0 && selectedAccounts[0] != "" { placeholders := strings.Repeat("?,", len(selectedAccounts)-1) + "?"; whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", placeholders)); for _, v := range selectedAccounts { args = append(args, v) } }
	if selectedStartDate != "" { whereClauses = append(whereClauses, "data_ocorrencia >= ?"); args = append(args, selectedStartDate) }
	if selectedEndDate != "" { whereClauses = append(whereClauses, "data_ocorrencia <= ?"); args = append(args, selectedEndDate) }
	if selectedConsolidado != "" { if b, err := strconv.ParseBool(selectedConsolidado); err == nil { whereClauses = append(whereClauses, "consolidado = ?"); args = append(args, b) } }
	if selectedValueFilter == "income" { whereClauses = append(whereClauses, "valor >= 0") }
	if selectedValueFilter == "expense" { whereClauses = append(whereClauses, "valor < 0") }
	if len(whereClauses) > 0 { query += " AND " + strings.Join(whereClauses, " AND ") }
	query += " ORDER BY data_ocorrencia DESC, id DESC"

	rows, err := bindAndQuery(userID, query, args...)
	if err != nil { renderErrorPage(c, http.StatusInternalServerError, "Erro ao buscar movimentações.", err); return }
	defer rows.Close()

	var movimentacoes []models.Movimentacao
	var totalValor, totalEntradas, totalSaidas float64
	for rows.Next() {
		var mov models.Movimentacao; var rawData interface{}
		if err := rows.Scan(&mov.ID, &rawData, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil { log.Printf("Erro ao escanear linha da movimentação: %v", err); continue }
		mov.DataOcorrencia = scanDate(rawData); movimentacoes = append(movimentacoes, mov); totalValor += mov.Valor
		if mov.Valor >= 0 { totalEntradas += mov.Valor } else { totalSaidas += mov.Valor }
	}
	if err = rows.Err(); err != nil { renderErrorPage(c, http.StatusInternalServerError, "Erro durante a leitura das movimentações.", err); return }

	if strings.Contains(c.GetHeader("Accept"), "application/json") || c.Request.URL.Path == "/api/movimentacoes" {
		c.JSON(http.StatusOK, gin.H{ "movimentacoes": movimentacoes, "totalValor": totalValor, "totalEntradas": totalEntradas, "totalSaidas": totalSaidas, }); return
	}
	
	c.HTML(http.StatusOK, "transacoes.html", gin.H{
		"Movimentacoes":       movimentacoes, "Titulo": "Transações Financeiras", "SearchDescricao": searchDescricao,
		"SelectedCategories":  selectedCategories, "SelectedStartDate": selectedStartDate, "SelectedEndDate": selectedEndDate,
		"SelectedConsolidado": selectedConsolidado, "SelectedAccounts": selectedAccounts, "SelectedValueFilter": selectedValueFilter,
		"Categories":          getDistinctColumnValues(userID, "categoria"), "Accounts": getDistinctColumnValues(userID, "conta"),
		"ConsolidatedOptions": []struct{ Value, Label string }{{"", "Todos"}, {"true", "Sim"}, {"false", "Não"}},
		"TotalValor":          totalValor, "TotalEntradas": totalEntradas, "TotalSaidas": totalSaidas,
		"CurrentDate":         time.Now().Format("2006-01-02"),
		"User":                user,
	})
}

func GetRelatorio(c *gin.Context) {
	log.Println("--- EXECUTANDO HANDLER: GetRelatorio para a rota /relatorio ---") // <-- LOG DE DIAGNÓSTICO
	userID := c.MustGet("userID").(int64)
	user := c.MustGet("user").(*models.User)

	searchDescricao := c.Query("search_descricao")
	selectedCategories := c.QueryArray("category")
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidado := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account")

	if selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now(); firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()); selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()); selectedEndDate = lastOfMonth.Format("2006-01-02")
	}

	relatorioData, err := fetchReportData(userID, selectedStartDate, selectedEndDate, selectedCategories, selectedAccounts, selectedConsolidado, searchDescricao)
	if err != nil { renderErrorPage(c, http.StatusInternalServerError, "Erro ao buscar dados para o relatório.", err); return }

	c.HTML(http.StatusOK, "relatorio.html", gin.H{
		"Titulo": "Relatório de Despesas por Categoria", "ReportData": relatorioData,
		"SearchDescricao":     searchDescricao, "SelectedCategories": selectedCategories, "SelectedStartDate": selectedStartDate,
		"SelectedEndDate":     selectedEndDate, "SelectedConsolidado": selectedConsolidado, "SelectedAccounts": selectedAccounts,
		"Categories":          getDistinctColumnValues(userID, "categoria"), "Accounts": getDistinctColumnValues(userID, "conta"),
		"ConsolidatedOptions": []struct{ Value, Label string }{{"", "Todos"}, {"true", "Sim"}, {"false", "Não"}},
		"CurrentDate":         time.Now().Format("2006-01-02"),
		"User":                user,
	})
}

// =============================================================================
// Form & API Handlers
// =============================================================================

func AddMovimentacao(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	mov, err := validateMovimentacao(c)
	if err != nil { renderErrorPage(c, http.StatusBadRequest, err.Error(), err); return }
	query := fmt.Sprintf(`INSERT INTO %s (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`, database.TableName)
	reboundQuery := database.Rebind(query); db := database.GetDB()
	err = db.QueryRow(reboundQuery, userID, mov.DataOcorrencia, mov.Descricao, mov.Valor, mov.Categoria, mov.Conta, mov.Consolidado).Scan(&mov.ID)
	if err != nil { renderErrorPage(c, http.StatusInternalServerError, "Erro ao inserir os dados no banco de dados.", err); return }
	if strings.Contains(c.GetHeader("Accept"), "application/json") { c.JSON(http.StatusCreated, mov) } else { c.Redirect(http.StatusFound, "/transacoes") }
}

func UpdateMovimentacao(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderErrorPage(c, http.StatusBadRequest, "O ID da transação é inválido.", err)
		return
	}

	mov, err := validateMovimentacao(c)
	if err != nil {
		renderErrorPage(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	// Consulta SQL com placeholders corretos para o driver
	query := fmt.Sprintf(`UPDATE %s SET data_ocorrencia = ?, descricao = ?, valor = ?, categoria = ?, conta = ?, consolidado = ? WHERE id = ? AND user_id = ?`, database.TableName)
	reboundQuery := database.Rebind(query)

	// Obtém a conexão com o banco de dados
	db := database.GetDB()

	// Executa a consulta diretamente com os argumentos NA ORDEM CORRETA
	// Nota: `id` e `userID` são os últimos, correspondendo a `id = ?` e `user_id = ?`
	_, err = db.Exec(reboundQuery, mov.DataOcorrencia, mov.Descricao, mov.Valor, mov.Categoria, mov.Conta, mov.Consolidado, id, userID)

	if err != nil {
		// O log do erro original é muito útil aqui
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao atualizar os dados.", err)
		return
	}

	c.Redirect(http.StatusFound, "/transacoes")
}

func DeleteMovimentacao(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido."})
		return
	}

	// Query com a ordem correta dos placeholders
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ? AND user_id = ?", database.TableName)
	reboundQuery := database.Rebind(query)

	db := database.GetDB()

	// Executa diretamente, passando os argumentos na ordem correta: primeiro o `id`, depois o `userID`
	_, err = db.Exec(reboundQuery, id, userID)

	if err != nil {
		log.Printf("Erro ao deletar movimentação ID %d para usuário %d: %v", id, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar a movimentação."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Movimentação deletada com sucesso!"})
}

func GetTransactionsByCategory(c *gin.Context) {
	userID := c.MustGet("userID").(int64); category := c.Query("category"); searchDescricao := c.Query("search_descricao"); selectedStartDate := c.Query("start_date"); selectedEndDate := c.Query("end_date"); selectedConsolidado := c.Query("consolidated_filter"); selectedAccounts := c.QueryArray("account")
	if selectedStartDate == "" && selectedEndDate == "" { now := time.Now(); firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()); selectedStartDate = firstOfMonth.Format("2006-01-02"); lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()); selectedEndDate = lastOfMonth.Format("2006-01-02") }
	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ?", database.TableName)
	var args []interface{}; var whereClauses []string; whereClauses = append(whereClauses, "categoria = ?"); args = append(args, category); whereClauses = append(whereClauses, "valor < 0")
	if searchDescricao != "" { clause := "descricao LIKE ?"; if database.DriverName == "postgres" { clause = "descricao ILIKE ?" }; whereClauses = append(whereClauses, clause); args = append(args, "%"+searchDescricao+"%") }
	if len(selectedAccounts) > 0 && selectedAccounts[0] != "" { placeholders := strings.Repeat("?,", len(selectedAccounts)-1) + "?"; whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", placeholders)); for _, v := range selectedAccounts { args = append(args, v) } }
	if selectedStartDate != "" { whereClauses = append(whereClauses, "data_ocorrencia >= ?"); args = append(args, selectedStartDate) }
	if selectedEndDate != "" { whereClauses = append(whereClauses, "data_ocorrencia <= ?"); args = append(args, selectedEndDate) }
	if selectedConsolidado != "" { if b, err := strconv.ParseBool(selectedConsolidado); err == nil { whereClauses = append(whereClauses, "consolidado = ?"); args = append(args, b) } }
	if len(whereClauses) > 0 { query += " AND " + strings.Join(whereClauses, " AND ") }; query += " ORDER BY data_ocorrencia DESC"
	rows, err := bindAndQuery(userID, query, args...); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar transações: " + err.Error()}); return }; defer rows.Close()
	var transactions []models.Movimentacao
	for rows.Next() { var mov models.Movimentacao; var rawData interface{}; if err := rows.Scan(&mov.ID, &rawData, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil { log.Printf("Erro ao escanear transação: %v", err); continue }; mov.DataOcorrencia = scanDate(rawData); transactions = append(transactions, mov) }
	if err = rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro na iteração: " + err.Error()}); return }
	c.JSON(http.StatusOK, transactions)
}

func DownloadRelatorioPDF(c *gin.Context) {
	userID := c.MustGet("userID").(int64); var payload models.PDFRequestPayload; if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()}); return }
	reportData, err := fetchReportData(userID, payload.StartDate, payload.EndDate, payload.Categories, payload.Accounts, payload.ConsolidatedFilter, payload.SearchDescricao); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar dados do relatório: " + err.Error()}); return }
	transactions, err := fetchAllTransactions(userID, payload.StartDate, payload.EndDate, payload.Categories, payload.Accounts, payload.ConsolidatedFilter, payload.SearchDescricao); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar transações detalhadas: " + err.Error()}); return }
	pdf, err := pdfgenerator.GenerateReportPDF(reportData, transactions, payload.ChartImageBase64); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar PDF: " + err.Error()}); return }
	c.Header("Content-Type", "application/pdf"); c.Header("Content-Disposition", `attachment; filename="relatorio_financeiro.pdf"`); if err := pdf.Output(c.Writer); err != nil { log.Printf("Erro ao enviar PDF para o cliente: %v", err) }
}

// Internal Data Fetching Functions
func calculateAccountBalances(userID int64) ([]models.ContaSaldo, error) {
	saldos := make(map[string]float64); db := database.GetDB()
	queryContas := database.Rebind("SELECT nome, saldo_inicial FROM contas WHERE user_id = ?"); rowsContas, err := db.Query(queryContas, userID)
	if err == nil { for rowsContas.Next() { var nome string; var saldoInicial float64; if err := rowsContas.Scan(&nome, &saldoInicial); err == nil { saldos[nome] = saldoInicial } }; rowsContas.Close() } else { log.Printf("Aviso: Não foi possível ler a tabela 'contas' para o usuário %d: %v.", userID, err) }
	queryMov := database.Rebind(fmt.Sprintf("SELECT conta, SUM(valor) FROM %s WHERE user_id = ? GROUP BY conta", database.TableName)); rowsMov, err := db.Query(queryMov, userID); if err != nil { return nil, fmt.Errorf("erro ao calcular totais por conta: %w", err) }; defer rowsMov.Close()
	for rowsMov.Next() { var conta string; var totalMovimentacoes float64; if err := rowsMov.Scan(&conta, &totalMovimentacoes); err != nil { continue }; saldos[conta] += totalMovimentacoes }
	var result []models.ContaSaldo; for nome, saldo := range saldos { result = append(result, models.ContaSaldo{Nome: nome, SaldoAtual: saldo, URLEncodedNome: url.QueryEscape(nome)}) }
	sort.Slice(result, func(i, j int) bool { return result[i].Nome < result[j].Nome }); return result, nil
}

func fetchReportData(userID int64, startDate, endDate string, categories, accounts []string, consolidated, searchDescricao string) ([]models.RelatorioCategoria, error) {
	query := fmt.Sprintf("SELECT categoria, SUM(valor) FROM %s WHERE user_id = ? AND valor < 0", database.TableName)
	var args []interface{}; var whereClauses []string
	if searchDescricao != "" { whereClauses = append(whereClauses, "descricao LIKE ?"); args = append(args, "%"+searchDescricao+"%") }
	if len(categories) > 0 && categories[0] != "" { placeholders := strings.Repeat("?,", len(categories)-1) + "?"; whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", placeholders)); for _, v := range categories { args = append(args, v) } }
	if len(accounts) > 0 && accounts[0] != "" { placeholders := strings.Repeat("?,", len(accounts)-1) + "?"; whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", placeholders)); for _, v := range accounts { args = append(args, v) } }
	if startDate != "" { whereClauses = append(whereClauses, "data_ocorrencia >= ?"); args = append(args, startDate) }
	if endDate != "" { whereClauses = append(whereClauses, "data_ocorrencia <= ?"); args = append(args, endDate) }
	if consolidated != "" { if b, err := strconv.ParseBool(consolidated); err == nil { whereClauses = append(whereClauses, "consolidado = ?"); args = append(args, b) } }
	if len(whereClauses) > 0 { query += " AND " + strings.Join(whereClauses, " AND ") }; query += " GROUP BY categoria ORDER BY SUM(valor) ASC"
	rows, err := bindAndQuery(userID, query, args...); if err != nil { return nil, err }; defer rows.Close()
	var relatorioData []models.RelatorioCategoria
	for rows.Next() { var rc models.RelatorioCategoria; if err := rows.Scan(&rc.Categoria, &rc.Total); err != nil { log.Printf("Erro ao escanear linha do relatório: %v", err); continue }; relatorioData = append(relatorioData, rc) }
	return relatorioData, rows.Err()
}

func fetchAllTransactions(userID int64, startDate, endDate string, categories, accounts []string, consolidated, searchDescricao string) ([]models.Movimentacao, error) {
	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ?", database.TableName)
	var args []interface{}; var whereClauses []string; whereClauses = append(whereClauses, "valor < 0")
	if searchDescricao != "" { whereClauses = append(whereClauses, "descricao LIKE ?"); args = append(args, "%"+searchDescricao+"%") }
	if len(categories) > 0 && categories[0] != "" { placeholders := strings.Repeat("?,", len(categories)-1) + "?"; whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", placeholders)); for _, v := range categories { args = append(args, v) } }
	if len(accounts) > 0 && accounts[0] != "" { placeholders := strings.Repeat("?,", len(accounts)-1) + "?"; whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", placeholders)); for _, v := range accounts { args = append(args, v) } }
	if startDate != "" { whereClauses = append(whereClauses, "data_ocorrencia >= ?"); args = append(args, startDate) }
	if endDate != "" { whereClauses = append(whereClauses, "data_ocorrencia <= ?"); args = append(args, endDate) }
	if consolidated != "" { if b, err := strconv.ParseBool(consolidated); err == nil { whereClauses = append(whereClauses, "consolidado = ?"); args = append(args, b) } }
	if len(whereClauses) > 0 { query += " AND " + strings.Join(whereClauses, " AND ") }; query += " ORDER BY data_ocorrencia DESC"
	rows, err := bindAndQuery(userID, query, args...); if err != nil { return nil, err }; defer rows.Close()
	var transactions []models.Movimentacao
	for rows.Next() { var mov models.Movimentacao; var rawData interface{}; if err := rows.Scan(&mov.ID, &rawData, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil { log.Printf("Erro ao escanear transação: %v", err); continue }; mov.DataOcorrencia = scanDate(rawData); transactions = append(transactions, mov) }
	return transactions, rows.Err()
}