// handlers/movimentacoes.go
package handlers

import (
	// "database/sql" // Removida esta importação desnecessária
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"strconv"

	"github.com/gin-gonic/gin"
	// O nome do módulo que você usou em 'go mod init'
	"minhas_economias/models"    // Importe o pacote models
	"minhas_economias/database" // Importe o pacote database para TableName
)

// GetMovimentacoes busca os registros de movimentacoes do banco de dados, com filtros opcionais
func GetMovimentacoes(c *gin.Context) {
	db := database.GetDB() // Obtém a conexão do banco de dados

	// Obter parâmetros de filtro da URL
	selectedCategories := c.QueryArray("category")
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidado := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account")
	selectedValueFilter := c.Query("value_filter") // Novo: "income", "expense", ou ""

	// Se não houver filtros de data na URL, define para o mês corrente
	if selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")
		log.Printf("Nenhum filtro de data fornecido. Default para mês corrente: %s a %s\n", selectedStartDate, selectedEndDate)
	}

	// Construir a consulta SQL base
	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s", database.TableName)
	var args []interface{}
	var whereClauses []string

	if len(selectedCategories) > 0 && selectedCategories[0] != "" {
		placeholders := make([]string, len(selectedCategories))
		for i := range selectedCategories {
			placeholders[i] = "?"
			args = append(args, selectedCategories[i])
		}
		whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(selectedAccounts) > 0 && selectedAccounts[0] != "" {
		placeholders := make([]string, len(selectedAccounts))
		for i := range selectedAccounts {
			placeholders[i] = "?"
			args = append(args, selectedAccounts[i])
		}
		whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", strings.Join(placeholders, ",")))
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
	// Novo: Filtro por tipo de valor (entradas/saídas)
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

	var movimentacoes []models.Movimentacao // Usar o struct Movimentacao do pacote models
	var totalValor float64
	var totalEntradas float64
	var totalSaidas float64

	for rows.Next() {
		var mov models.Movimentacao
		err := rows.Scan(&mov.ID, &mov.DataOcorrencia, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado)
		if err != nil {
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

	err = rows.Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro na iteração das linhas de movimentação: %v", err)})
		return
	}

	// Buscar categorias distintas para o filtro
	categoryRows, err := db.Query(fmt.Sprintf("SELECT DISTINCT categoria FROM %s ORDER BY categoria ASC", database.TableName))
	if err != nil {
		log.Printf("AVISO: Erro ao buscar categorias distintas: %v", err)
	}
	defer func() {
		if categoryRows != nil {
			categoryRows.Close()
		}
	}()

	var categories []string
	if categoryRows != nil {
		for categoryRows.Next() {
			var cat string
			if err := categoryRows.Scan(&cat); err != nil {
				log.Printf("AVISO: Erro ao escanear categoria distinta: %v", err)
				continue
			}
			categories = append(categories, cat)
		}
		if err = categoryRows.Err(); err != nil {
			log.Printf("AVISO: Erro na iteração das categorias distintas: %v", err)
		}
	}

	// Buscar contas distintas para o filtro
	accountRows, err := db.Query(fmt.Sprintf("SELECT DISTINCT conta FROM %s ORDER BY conta ASC", database.TableName))
	if err != nil {
		log.Printf("AVISO: Erro ao buscar contas distintas: %v", err)
	}
	defer func() {
		if accountRows != nil {
			accountRows.Close()
		}
	}()

	var accounts []string
	if accountRows != nil {
		for accountRows.Next() {
			var acc string
			if err := accountRows.Scan(&acc); err != nil {
				log.Printf("AVISO: Erro ao escanear conta distinta: %v", err)
				continue
			}
			accounts = append(accounts, acc)
		}
		if err = accountRows.Err(); err != nil {
			log.Printf("AVISO: Erro na iteração das contas distintas: %v", err)
		}
	}

	consolidatedOptions := []struct {
		Value string
		Label string
	}{
		{"", "Todos"},
		{"true", "Sim"},
		{"false", "Não"},
	}

	currentDate := time.Now().Format("2006-01-02")

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
			"SelectedCategories":   selectedCategories,
			"SelectedStartDate":    selectedStartDate,
			"SelectedEndDate":      selectedEndDate,
			"SelectedConsolidated": selectedConsolidado,
			"SelectedAccounts":     selectedAccounts,
			"SelectedValueFilter":  selectedValueFilter, // Novo: passa o filtro de valor selecionado
			"Categories":           categories,
			"Accounts":             accounts,
			"ConsolidatedOptions":  consolidatedOptions,
			"TotalValor":           totalValor,
			"TotalEntradas":        totalEntradas,
			"TotalSaidas":          totalSaidas,
			"CurrentDate":          currentDate,
		})
	}
}

// AddMovimentacao lida com a inserção de uma nova movimentação via formulário POST
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
		log.Println("ERRO: Campo 'Conta' é obrigatório.")
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
			log.Printf("Erro ao converter Valor '%s': %v", valorStr, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Valor inválido: formato numérico incorreto."})
			return
		}
		valor = parsedValor
	}

	consolidado := (consolidadoStr == "on")

	stmt, err := db.Prepare(fmt.Sprintf(
		`INSERT INTO %s (data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?)`, database.TableName))
	if err != nil {
		log.Printf("Erro ao preparar instrução SQL para adição: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno do servidor."})
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(dataOcorrencia, descricao, valor, categoria, conta, consolidado)
	if err != nil {
		log.Printf("Erro ao inserir nova movimentação: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao inserir dados: %v", err.Error())})
		return
	}

	c.Redirect(http.StatusFound, "/")
}

// DeleteMovimentacao lida com a exclusão de uma movimentação
func DeleteMovimentacao(c *gin.Context) {
    db := database.GetDB()

    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido."})
        return
    }

    _, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", database.TableName), id)
    if err != nil {
        log.Printf("Erro ao deletar movimentação com ID %d: %v", id, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar a movimentação."})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Movimentação deletada com sucesso!"})
}

// UpdateMovimentacao lida com a atualização de uma movimentação
func UpdateMovimentacao(c *gin.Context) {
    db := database.GetDB()

    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
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
        log.Println("ERRO: Campo 'Conta' é obrigatório na atualização.")
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
            log.Printf("Erro ao converter Valor '%s' na atualização: %v", valorStr, err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "Valor inválido: formato numérico incorreto."})
            return
        }
        valor = parsedValor
    }

    consolidado := (consolidadoStr == "on")

    stmt, err := db.Prepare(fmt.Sprintf(
        `UPDATE %s SET data_ocorrencia = ?, descricao = ?, valor = ?, categoria = ?, conta = ?, consolidado = ? WHERE id = ?`, database.TableName))
    if err != nil {
        log.Printf("Erro ao preparar instrução SQL para atualização: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno do servidor."})
        return
    }
    defer stmt.Close()

    _, err = stmt.Exec(dataOcorrencia, descricao, valor, categoria, conta, consolidado, id)
    if err != nil {
        log.Printf("Erro ao atualizar movimentação com ID %d: %v", id, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao atualizar dados: %v", err.Error())})
        return
    }

    c.Redirect(http.StatusFound, "/")
}

// RelatorioCategoria representa o total de despesas por categoria para o relatório.
type RelatorioCategoria struct {
	Categoria string  `json:"categoria"`
	Total     float64 `json:"total"`
}

// GetRelatorio busca e agrega despesas por categoria para o relatório.
func GetRelatorio(c *gin.Context) {
	db := database.GetDB() // Obtém a conexão do banco de dados

	// Obter parâmetros de filtro da URL (idênticos aos da página principal)
	selectedCategories := c.QueryArray("category")
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidado := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account")

	// Se não houver filtros de data na URL, define para o mês corrente
	if selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")
	}

	// Construir a consulta SQL para despesas por categoria
	query := fmt.Sprintf("SELECT categoria, SUM(valor) FROM %s WHERE valor < 0", database.TableName)
	var args []interface{}
	var whereClauses []string // Cláusulas WHERE adicionais para filtros

	if len(selectedCategories) > 0 && selectedCategories[0] != "" {
		placeholders := make([]string, len(selectedCategories))
		for i := range selectedCategories {
			placeholders[i] = "?"
			args = append(args, selectedCategories[i])
		}
		whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(selectedAccounts) > 0 && selectedAccounts[0] != "" {
		placeholders := make([]string, len(selectedAccounts))
		for i := range selectedAccounts {
			placeholders[i] = "?"
			args = append(args, selectedAccounts[i])
		}
		whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", strings.Join(placeholders, ",")))
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

	// Adicionar cláusulas WHERE à query principal
	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ") // Usa AND porque já temos "WHERE valor < 0"
	}

	query += " GROUP BY categoria ORDER BY SUM(valor) ASC" // Ordena do menor (mais negativo) para o maior

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao buscar dados para relatório: %v", err)})
		return
	}
	defer rows.Close()

	var relatorioData []RelatorioCategoria
	for rows.Next() {
		var rc RelatorioCategoria
		err := rows.Scan(&rc.Categoria, &rc.Total)
		if err != nil {
			log.Printf("Erro ao escanear linha do relatório: %v", err)
			continue
		}
		relatorioData = append(relatorioData, rc)
	}

	err = rows.Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro na iteração das linhas do relatório: %v", err)})
		return
	}

	// Buscar categorias distintas (para o filtro da página de relatório)
	categoryRows, err := db.Query(fmt.Sprintf("SELECT DISTINCT categoria FROM %s ORDER BY categoria ASC", database.TableName))
	var categories []string
	if err == nil { // Apenas processa se não houver erro na query
		defer categoryRows.Close()
		for categoryRows.Next() {
			var cat string
			if err := categoryRows.Scan(&cat); err != nil {
				log.Printf("AVISO: Erro ao escanear categoria distinta para relatório: %v", err)
				continue
			}
			categories = append(categories, cat)
		}
	} else {
		log.Printf("AVISO: Erro ao buscar categorias distintas para relatório: %v", err)
	}

	// Buscar contas distintas (para o filtro da página de relatório)
	accountRows, err := db.Query(fmt.Sprintf("SELECT DISTINCT conta FROM %s ORDER BY conta ASC", database.TableName))
	var accounts []string
	if err == nil { // Apenas processa se não houver erro na query
		defer accountRows.Close()
		for accountRows.Next() {
			var acc string
			if err := accountRows.Scan(&acc); err != nil {
				log.Printf("AVISO: Erro ao escanear conta distinta para relatório: %v", err)
				continue
			}
			accounts = append(accounts, acc)
		}
	} else {
		log.Printf("AVISO: Erro ao buscar contas distintas para relatório: %v", err)
	}

	consolidatedOptions := []struct {
		Value string
		Label string
	}{
		{"", "Todos"},
		{"true", "Sim"},
		{"false", "Não"},
	}

	currentDate := time.Now().Format("2006-01-02")

	// Renderiza a página de relatório
	c.HTML(http.StatusOK, "relatorio.html", gin.H{
		"Titulo":               "Relatório de Despesas por Categoria",
		"ReportData":           relatorioData, // Dados agregados para o gráfico
		"SelectedCategories":   selectedCategories,
		"SelectedStartDate":    selectedStartDate,
		"SelectedEndDate":      selectedEndDate,
		"SelectedConsolidated": selectedConsolidado,
		"SelectedAccounts":     selectedAccounts,
		"Categories":           categories,
		"Accounts":             accounts,
		"ConsolidatedOptions":  consolidatedOptions,
		"CurrentDate":          currentDate,
	})
}

// GetTransactionsByCategory busca transações individuais para uma categoria e filtros dados.
func GetTransactionsByCategory(c *gin.Context) {
	db := database.GetDB()

	category := c.Query("category") // Categoria específica clicada
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidado := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account")

	// Se não houver filtros de data na URL, define para o mês corrente (padrão da página de relatório)
	if selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")
	}

	// Construir a consulta SQL para transações individuais da categoria e que são despesas
	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s WHERE categoria = ? AND valor < 0", database.TableName)
	var args []interface{}
	args = append(args, category) // A categoria é sempre o primeiro argumento

	var whereClauses []string // Cláusulas WHERE adicionais para outros filtros

	if len(selectedAccounts) > 0 && selectedAccounts[0] != "" {
		placeholders := make([]string, len(selectedAccounts))
		for i := range selectedAccounts {
			placeholders[i] = "?"
			args = append(args, selectedAccounts[i])
		}
		whereClauses = append(whereClauses, fmt.Sprintf("conta IN (%s)", strings.Join(placeholders, ",")))
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

	// Adicionar cláusulas WHERE à query principal
	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY data_ocorrencia DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao buscar transações por categoria: %v", err)})
		return
	}
	defer rows.Close()

	var transactions []models.Movimentacao
	for rows.Next() {
		var mov models.Movimentacao
		err := rows.Scan(&mov.ID, &mov.DataOcorrencia, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado)
		if err != nil {
			log.Printf("Erro ao escanear linha da transação por categoria: %v", err)
			continue
		}
		transactions = append(transactions, mov)
	}

	err = rows.Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro na iteração das transações por categoria: %v", err)})
		return
	}

	c.JSON(http.StatusOK, transactions) // Retorna as transações em JSON
}
