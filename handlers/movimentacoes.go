package handlers

import (
	"bytes" // NOVO
	"database/sql"
	"encoding/csv" // NOVO
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

// AddTransferencia handles the creation of a transfer between two accounts.
func AddTransferencia(c *gin.Context) {
	userID := c.MustGet("userID").(int64)

	// Bind form data
	dataOcorrencia := c.PostForm("data_ocorrencia")
	descricao := c.PostForm("descricao")
	valorStr := strings.Replace(c.PostForm("valor"), ",", ".", -1)
	contaOrigem := c.PostForm("conta_origem")
	contaDestino := c.PostForm("conta_destino")

	// Validation
	if contaOrigem == "" || contaDestino == "" || valorStr == "" || dataOcorrencia == "" {
		renderErrorPage(c, http.StatusBadRequest, "Todos os campos da transferência são obrigatórios.", nil)
		return
	}
	if contaOrigem == contaDestino {
		renderErrorPage(c, http.StatusBadRequest, "A conta de origem e destino não podem ser a mesma.", nil)
		return
	}
	valor, err := strconv.ParseFloat(valorStr, 64)
	if err != nil || valor <= 0 {
		renderErrorPage(c, http.StatusBadRequest, "O valor da transferência deve ser um número positivo.", err)
		return
	}

	db := database.GetDB()
	tx, err := db.Begin()
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao iniciar a transação no banco de dados.", err)
		return
	}

	// Prepare statement inside transaction
	query := fmt.Sprintf(`INSERT INTO %s (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?, ?)`, database.TableName)
	stmt, err := tx.Prepare(database.Rebind(query))
	if err != nil {
		tx.Rollback()
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao preparar a inserção no banco de dados.", err)
		return
	}
	defer stmt.Close()

	// 1. Débito da conta de origem (valor negativo)
	valorNegativo := -math.Abs(valor)
	descricaoOrigem := fmt.Sprintf("Transferência para %s: %s", contaDestino, descricao)
	if _, err := stmt.Exec(userID, dataOcorrencia, descricaoOrigem, valorNegativo, "Transferência", contaOrigem, true); err != nil {
		tx.Rollback()
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao registrar a saída da conta de origem.", err)
		return
	}

	// 2. Crédito na conta de destino (valor positivo)
	valorPositivo := math.Abs(valor)
	descricaoDestino := fmt.Sprintf("Transferência de %s: %s", contaOrigem, descricao)
	if _, err := stmt.Exec(userID, dataOcorrencia, descricaoDestino, valorPositivo, "Transferência", contaDestino, true); err != nil {
		tx.Rollback()
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao registrar a entrada na conta de destino.", err)
		return
	}

	// Commit da transação se tudo ocorreu bem
	if err := tx.Commit(); err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao finalizar a transação.", err)
		return
	}

	c.Redirect(http.StatusFound, "/transacoes")
}

func GetSaldosAPI(c *gin.Context) {
	userID := c.MustGet("userID").(int64)

	saldosContas, err := calculateAccountBalances(userID)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Não foi possível carregar os saldos das contas.", err)
		return
	}

	var saldoGeral float64
	for _, saldo := range saldosContas {
		saldoGeral += saldo.SaldoAtual
	}

	c.JSON(http.StatusOK, gin.H{
		"saldoGeral":   saldoGeral,
		"saldosContas": saldosContas,
	})
}

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
	if rawData == nil {
		return ""
	}
	if t, ok := rawData.(time.Time); ok {
		return t.Format("2006-01-02")
	}
	if s, ok := rawData.(string); ok {
		return s
	}
	if b, ok := rawData.([]byte); ok {
		return string(b)
	}
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
	if len(mov.Descricao) > 60 {
		return mov, fmt.Errorf("A descrição não pode ter mais de 60 caracteres.")
	}
	if strings.TrimSpace(mov.Categoria) == "" {
		mov.Categoria = "Sem Categoria"
	}
	if strings.TrimSpace(mov.Conta) == "" {
		return mov, fmt.Errorf("O campo 'Conta' é obrigatório.")
	}
	if strings.TrimSpace(valorStr) == "" {
		mov.Valor = 0.0
	} else {
		valorParseable := strings.Replace(valorStr, ",", ".", -1)
		if isValid, _ := regexp.MatchString(`^-?\d+(\.\d{1,2})?$`, valorParseable); !isValid {
			return mov, fmt.Errorf("Valor inválido. Use um formato como 1234.56 ou -123.45.")
		}
		if mov.Valor, err = strconv.ParseFloat(valorParseable, 64); err != nil {
			return mov, fmt.Errorf("Valor inválido: formato numérico incorreto.")
		}
		if math.Abs(mov.Valor) >= 100000000 {
			return mov, fmt.Errorf("O valor excede o limite máximo permitido (100 milhões).")
		}
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

	var saldoGeral float64
	for _, saldo := range saldosContas {
		saldoGeral += saldo.SaldoAtual
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Titulo":       "Minhas Economias - Saldos",
		"SaldosContas": saldosContas,
		"User":         user,
		"SaldoGeral":   saldoGeral,
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

	// --- INÍCIO DA CORREÇÃO ---
	// Verifica se a requisição é para a API ou para a página web
	isApiRequest := strings.Contains(c.GetHeader("Accept"), "application/json") || c.Request.URL.Path == "/api/movimentacoes"

	// Aplica o filtro de data padrão SOMENTE se não for uma chamada de API
	// e se nenhuma data tiver sido fornecida pelo usuário.
	if !isApiRequest && selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")
	}
	// --- FIM DA CORREÇÃO ---

	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ?", database.TableName)
	var args []interface{}
	var whereClauses []string
	if searchDescricao != "" {
		clause := "descricao LIKE ?"
		if database.DriverName == "postgres" {
			clause = "descricao ILIKE ?"
		}
		whereClauses = append(whereClauses, clause)
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
		if b, err := strconv.ParseBool(selectedConsolidado); err == nil {
			whereClauses = append(whereClauses, "consolidado = ?")
			args = append(args, b)
		}
	}
	if selectedValueFilter == "income" {
		whereClauses = append(whereClauses, "valor >= 0")
	}
	if selectedValueFilter == "expense" {
		whereClauses = append(whereClauses, "valor < 0")
	}
	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY data_ocorrencia DESC, id DESC"

	rows, err := bindAndQuery(userID, query, args...)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao buscar movimentações.", err)
		return
	}
	defer rows.Close()

	var movimentacoes []models.Movimentacao
	var totalValor, totalEntradas, totalSaidas float64
	for rows.Next() {
		var mov models.Movimentacao
		var rawData interface{}
		if err := rows.Scan(&mov.ID, &rawData, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			log.Printf("Erro ao escanear linha da movimentação: %v", err)
			continue
		}
		mov.DataOcorrencia = scanDate(rawData)
		movimentacoes = append(movimentacoes, mov)
		totalValor += mov.Valor
		if mov.Valor >= 0 {
			totalEntradas += mov.Valor
		} else {
			totalSaidas += mov.Valor
		}
	}
	if err = rows.Err(); err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro durante a leitura das movimentações.", err)
		return
	}

	if strings.Contains(c.GetHeader("Accept"), "application/json") || c.Request.URL.Path == "/api/movimentacoes" {
		c.JSON(http.StatusOK, gin.H{"movimentacoes": movimentacoes, "totalValor": totalValor, "totalEntradas": totalEntradas, "totalSaidas": totalSaidas})
		return
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
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")
	}

	relatorioData, err := fetchReportData(userID, selectedStartDate, selectedEndDate, selectedCategories, selectedAccounts, selectedConsolidado, searchDescricao)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao buscar dados para o relatório.", err)
		return
	}

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

// handlers/movimentacoes.go

func AddMovimentacao(c *gin.Context) {
	log.Println("--- EXECUTANDO AddMovimentacao ---")
	userID := c.MustGet("userID").(int64)
	mov, err := validateMovimentacao(c)
	if err != nil {
		renderErrorPage(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	db := database.GetDB()

	// Lógica de inserção que funciona para PostgreSQL e SQLite
	if database.DriverName == "postgres" {
		query := fmt.Sprintf(`INSERT INTO %s (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, database.TableName)
		err = db.QueryRow(query, userID, mov.DataOcorrencia, mov.Descricao, mov.Valor, mov.Categoria, mov.Conta, mov.Consolidado).Scan(&mov.ID)
	} else { // Padrão para SQLite
		query := fmt.Sprintf(`INSERT INTO %s (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?, ?)`, database.TableName)
		result, execErr := db.Exec(database.Rebind(query), userID, mov.DataOcorrencia, mov.Descricao, mov.Valor, mov.Categoria, mov.Conta, mov.Consolidado)
		if execErr == nil {
			lastID, _ := result.LastInsertId()
			mov.ID = int(lastID)
		}
		err = execErr
	}

	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao inserir os dados no banco de dados.", err)
		return
	}

	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusCreated, mov)
	} else {
		c.Redirect(http.StatusFound, "/transacoes")
	}
}

func UpdateMovimentacao(c *gin.Context) {
    log.Println("--- EXECUTANDO UpdateMovimentacao ---") // <-- ADICIONE ESTA LINHA
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
	userID := c.MustGet("userID").(int64)
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
	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ?", database.TableName)
	var args []interface{}
	var whereClauses []string
	whereClauses = append(whereClauses, "categoria = ?")
	args = append(args, category)
	whereClauses = append(whereClauses, "valor < 0")
	if searchDescricao != "" {
		clause := "descricao LIKE ?"
		if database.DriverName == "postgres" {
			clause = "descricao ILIKE ?"
		}
		whereClauses = append(whereClauses, clause)
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
		if b, err := strconv.ParseBool(selectedConsolidado); err == nil {
			whereClauses = append(whereClauses, "consolidado = ?")
			args = append(args, b)
		}
	}
	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY data_ocorrencia DESC"
	rows, err := bindAndQuery(userID, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar transações: " + err.Error()})
		return
	}
	defer rows.Close()
	var transactions []models.Movimentacao
	for rows.Next() {
		var mov models.Movimentacao
		var rawData interface{}
		if err := rows.Scan(&mov.ID, &rawData, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			log.Printf("Erro ao escanear transação: %v", err)
			continue
		}
		mov.DataOcorrencia = scanDate(rawData)
		transactions = append(transactions, mov)
	}
	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro na iteração: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

func DownloadRelatorioPDF(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var payload models.PDFRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}
	reportData, err := fetchReportData(userID, payload.StartDate, payload.EndDate, payload.Categories, payload.Accounts, payload.ConsolidatedFilter, payload.SearchDescricao)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar dados do relatório: " + err.Error()})
		return
	}
	transactions, err := fetchAllTransactions(userID, payload.StartDate, payload.EndDate, payload.Categories, payload.Accounts, payload.ConsolidatedFilter, payload.SearchDescricao)
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

// Internal Data Fetching Functions
func calculateAccountBalances(userID int64) ([]models.ContaSaldo, error) {
	saldos := make(map[string]float64)
	db := database.GetDB()
	queryContas := database.Rebind("SELECT nome, saldo_inicial FROM contas WHERE user_id = ?")
	rowsContas, err := db.Query(queryContas, userID)
	if err == nil {
		for rowsContas.Next() {
			var nome string
			var saldoInicial float64
			if err := rowsContas.Scan(&nome, &saldoInicial); err == nil {
				saldos[nome] = saldoInicial
			}
		}
		rowsContas.Close()
	} else {
		log.Printf("Aviso: Não foi possível ler a tabela 'contas' para o usuário %d: %v.", userID, err)
	}
	queryMov := database.Rebind(fmt.Sprintf("SELECT conta, SUM(valor) FROM %s WHERE user_id = ? GROUP BY conta", database.TableName))
	rowsMov, err := db.Query(queryMov, userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao calcular totais por conta: %w", err)
	}
	defer rowsMov.Close()
	for rowsMov.Next() {
		var conta string
		var totalMovimentacoes float64
		if err := rowsMov.Scan(&conta, &totalMovimentacoes); err != nil {
			continue
		}
		saldos[conta] += totalMovimentacoes
	}
	var result []models.ContaSaldo
	for nome, saldo := range saldos {
		result = append(result, models.ContaSaldo{Nome: nome, SaldoAtual: saldo, URLEncodedNome: url.QueryEscape(nome)})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Nome < result[j].Nome })
	return result, nil
}

func fetchReportData(userID int64, startDate, endDate string, categories, accounts []string, consolidated, searchDescricao string) ([]models.RelatorioCategoria, error) {
	query := fmt.Sprintf("SELECT categoria, SUM(valor) FROM %s WHERE user_id = ? AND valor < 0", database.TableName)
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
		if b, err := strconv.ParseBool(consolidated); err == nil {
			whereClauses = append(whereClauses, "consolidado = ?")
			args = append(args, b)
		}
	}
	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}
	query += " GROUP BY categoria ORDER BY SUM(valor) ASC"
	rows, err := bindAndQuery(userID, query, args...)
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

func fetchAllTransactions(userID int64, startDate, endDate string, categories, accounts []string, consolidated, searchDescricao string) ([]models.Movimentacao, error) {
	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ?", database.TableName)
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
		if b, err := strconv.ParseBool(consolidated); err == nil {
			whereClauses = append(whereClauses, "consolidado = ?")
			args = append(args, b)
		}
	}
	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY data_ocorrencia DESC"
	rows, err := bindAndQuery(userID, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transactions []models.Movimentacao
	for rows.Next() {
		var mov models.Movimentacao
		var rawData interface{}
		if err := rows.Scan(&mov.ID, &rawData, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			log.Printf("Erro ao escanear transação: %v", err)
			continue
		}
		mov.DataOcorrencia = scanDate(rawData)
		transactions = append(transactions, mov)
	}
	return transactions, rows.Err()
}

// =============================================================================
// NOVO HANDLER PARA EXPORTAÇÃO DE CSV
// =============================================================================

// ExportTransactionsCSV gera um arquivo CSV com as transações do usuário, aplicando os filtros da requisição.
func ExportTransactionsCSV(c *gin.Context) {
	log.Println("--- EXECUTANDO HANDLER: ExportTransactionsCSV ---")
	userID := c.MustGet("userID").(int64)

	// 1. Reutiliza a lógica de filtragem da GetTransacoesPage
	searchDescricao := c.Query("search_descricao")
	selectedCategories := c.QueryArray("category")
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidado := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account")
	selectedValueFilter := c.Query("value_filter")

	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE user_id = ?", database.TableName)
	var args []interface{}
	var whereClauses []string

	if searchDescricao != "" {
		clause := "descricao LIKE ?"
		if database.DriverName == "postgres" {
			clause = "descricao ILIKE ?"
		}
		whereClauses = append(whereClauses, clause)
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
		if b, err := strconv.ParseBool(selectedConsolidado); err == nil {
			whereClauses = append(whereClauses, "consolidado = ?")
			args = append(args, b)
		}
	}
	if selectedValueFilter == "income" {
		whereClauses = append(whereClauses, "valor >= 0")
	}
	if selectedValueFilter == "expense" {
		whereClauses = append(whereClauses, "valor < 0")
	}
	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY data_ocorrencia ASC, id ASC"

	rows, err := bindAndQuery(userID, query, args...)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao buscar movimentações para exportação.", err)
		return
	}
	defer rows.Close()

	// 2. Cria o CSV em um buffer de memória
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	writer.Comma = ';' // Usando o mesmo delimitador do seu data_manager

	// Escreve o cabeçalho
	header := []string{"Data Ocorrência", "Descrição", "Valor", "Categoria", "Conta", "Consolidado"}
	if err := writer.Write(header); err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao escrever o cabeçalho do CSV.", err)
		return
	}

	// Escreve as linhas de dados
	for rows.Next() {
		var mov models.Movimentacao
		var rawData interface{}
		if err := rows.Scan(&mov.ID, &rawData, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado); err != nil {
			log.Printf("Erro ao escanear linha para CSV: %v", err)
			continue
		}

		// Formata a data
		var formattedDateForCSV string
		if t, ok := rawData.(time.Time); ok {
			formattedDateForCSV = t.Format("02/01/2006")
		} else if s, ok := rawData.(string); ok {
			parsedDate, err := time.Parse("2006-01-02", s)
			if err == nil {
				formattedDateForCSV = parsedDate.Format("02/01/2006")
			} else {
				formattedDateForCSV = s
			}
		}

		// Formata o valor com vírgula como separador decimal
		valorFormatado := strings.Replace(strconv.FormatFloat(mov.Valor, 'f', 2, 64), ".", ",", -1)

		record := []string{
			formattedDateForCSV,
			mov.Descricao,
			valorFormatado,
			mov.Categoria,
			mov.Conta,
			strconv.FormatBool(mov.Consolidado),
		}

		if err := writer.Write(record); err != nil {
			log.Printf("Erro ao escrever registro no CSV: %v", err)
			continue
		}
	}
	if err = rows.Err(); err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro durante a leitura das movimentações para CSV.", err)
		return
	}

	writer.Flush()

	// 3. Envia a resposta como um download de arquivo
	filename := fmt.Sprintf("backup_minhas_economias_%s.csv", time.Now().Format("2006-01-02"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/csv")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}