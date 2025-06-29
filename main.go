// main.go
package main

import (
	"database/sql"
	"encoding/json" // Importar para usar json.Marshal
	"fmt"
	"log"
	"net/http"
	"html/template"
	"os"
	"strings"
	"time" // Importar o pacote time
	"strconv" // Para strconv.ParseFloat

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3" // Driver SQLite
)

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

var db *sql.DB

const tableName = "movimentacoes" // Definindo o nome da tabela para ser 'movimentacoes'

// initDB inicializa a conexão com o banco de dados. Assume que o banco de dados e a tabela já existem.
func initDB() {
	var err error
	// Abrir o arquivo do banco de dados SQLite.
	// O arquivo 'extratos.db' e a tabela 'movimentacoes' devem ser criados pelo programa 'minhas_economias_import_csv.go'.
	db, err = sql.Open("sqlite3", "./extratos.db")
	if err != nil {
		log.Fatalf("Erro ao abrir o banco de dados 'extratos.db': %v", err)
	}

	// Ping para verificar a conexão
	err = db.Ping()
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados 'extratos.db': %v", err)
	}
	log.Println("Conectado ao banco de dados 'extratos.db' com sucesso.")
}

// getMovimentacoes busca os registros de movimentacoes do banco de dados, com filtros opcionais
func getMovimentacoes(c *gin.Context) {
	// Obter parâmetros de filtro da URL
	selectedCategories := c.QueryArray("category")
	selectedStartDate := c.Query("start_date")
	selectedEndDate := c.Query("end_date")
	selectedConsolidated := c.Query("consolidated_filter")
	selectedAccounts := c.QueryArray("account") // Slice para múltiplas contas

	// Se não houver filtros de data na URL, define para o mês corrente
	if selectedStartDate == "" && selectedEndDate == "" {
		now := time.Now()
		// Primeiro dia do mês corrente
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		selectedStartDate = firstOfMonth.Format("2006-01-02")

		// Último dia do mês corrente
		lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
		selectedEndDate = lastOfMonth.Format("2006-01-02")

		log.Printf("Nenhum filtro de data fornecido. Default para mês corrente: %s a %s\n", selectedStartDate, selectedEndDate)
	}

	// Construir a consulta SQL base
	query := fmt.Sprintf("SELECT id, data_ocorrencia, descricao, valor, categoria, conta, consolidado FROM %s", tableName)
	var args []interface{}
	var whereClauses []string

	// Adicionar filtros à consulta se os parâmetros existirem
	if len(selectedCategories) > 0 && selectedCategories[0] != "" { // Verifica se há categorias selecionadas (o primeiro item não pode ser vazio)
		placeholders := make([]string, len(selectedCategories))
		for i := range selectedCategories {
			placeholders[i] = "?"
			args = append(args, selectedCategories[i])
		}
		whereClauses = append(whereClauses, fmt.Sprintf("categoria IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(selectedAccounts) > 0 && selectedAccounts[0] != "" { // Verifica se há contas selecionadas
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
	if selectedConsolidated != "" {
		if selectedConsolidated == "true" {
			whereClauses = append(whereClauses, "consolidado = 1")
		} else if selectedConsolidated == "false" {
			whereClauses = append(whereClauses, "consolidado = 0")
		}
	}

	// Adicionar cláusula WHERE se houver filtros
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Adicionar ordenação
	query += " ORDER BY data_ocorrencia DESC"

	// Executar a consulta para as movimentações
	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao buscar movimentações: %v", err)})
		return
	}
	defer rows.Close()

	var movimentacoes []Movimentacao
	var totalValor float64
	var totalEntradas float64 // Soma dos valores positivos
	var totalSaidas float64   // Soma dos valores negativos

	for rows.Next() {
		var mov Movimentacao
		err := rows.Scan(&mov.ID, &mov.DataOcorrencia, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta, &mov.Consolidado)
		if err != nil {
			log.Printf("Erro ao escanear linha da movimentação: %v", err)
			continue
		}
		movimentacoes = append(movimentacoes, mov)
		totalValor += mov.Valor // Soma para o total consolidado

		// Separação de entradas e saídas
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
	// NOTA: Com o campo categoria agora sendo preenchível, esta lista pode não ser exaustiva
	categoryRows, err := db.Query(fmt.Sprintf("SELECT DISTINCT categoria FROM %s ORDER BY categoria ASC", tableName))
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
	// NOTA: Com o campo conta agora sendo preenchível, esta lista pode não ser exaustiva
	accountRows, err := db.Query(fmt.Sprintf("SELECT DISTINCT conta FROM %s ORDER BY conta ASC", tableName))
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


	// Opções para o filtro de Consolidado
	consolidatedOptions := []struct {
		Value string
		Label string
	}{
		{"", "Todos"},
		{"true", "Sim"},
		{"false", "Não"},
	}

	// Data atual para preencher o campo de data no formulário de adição
	currentDate := time.Now().Format("2006-01-02")


	// Se for uma requisição de API, retorna JSON
	if c.Request.URL.Path == "/api/movimentacoes" {
		c.JSON(http.StatusOK, gin.H{
			"movimentacoes": movimentacoes,
			"totalValor":    totalValor,
			"totalEntradas": totalEntradas,
			"totalSaidas":   totalSaidas,
		})
	} else {
		// Senão, renderiza a página HTML
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Movimentacoes":        movimentacoes,
			"Titulo":               "Extratos Financeiros",
			"SelectedCategories":   selectedCategories,
			"SelectedStartDate":    selectedStartDate,
			"SelectedEndDate":      selectedEndDate,
			"SelectedConsolidated": selectedConsolidated,
			"SelectedAccounts":     selectedAccounts,
			"Categories":           categories,       // Para o filtro e formulário de adição
			"Accounts":             accounts,         // Para o filtro e formulário de adição
			"ConsolidatedOptions":  consolidatedOptions,
			"TotalValor":           totalValor,
			"TotalEntradas":        totalEntradas,
			"TotalSaidas":          totalSaidas,
			"CurrentDate":          currentDate,      // Passa a data atual para o formulário
		})
	}
}

// addMovimentacao lida com a inserção de uma nova movimentação via formulário POST
func addMovimentacao(c *gin.Context) {
	// Parsear os dados do formulário
	dataOcorrencia := c.PostForm("data_ocorrencia")
	descricao := c.PostForm("descricao")
	valorStr := c.PostForm("valor")
	categoria := c.PostForm("categoria")
	conta := c.PostForm("conta")
	consolidadoStr := c.PostForm("consolidado") // Checkbox retorna "on" se marcado

	// --- Novas Lógicas de Validação e Padronização ---

	// Categoria: Se vazio, "Sem Categoria"
	if strings.TrimSpace(categoria) == "" {
		categoria = "Sem Categoria"
	}

	// Conta: Obrigatório
	if strings.TrimSpace(conta) == "" {
		log.Println("ERRO: Campo 'Conta' é obrigatório.")
		c.JSON(http.StatusBadRequest, gin.H{"error": "O campo 'Conta' é obrigatório."})
		return
	}

	// Valor: Se vazio, 0
	var valor float64
	if strings.TrimSpace(valorStr) == "" {
		valor = 0.0
	} else {
		// Substitui vírgula por ponto antes de parsear para float (formato esperado por strconv.ParseFloat)
		valorParseable := strings.Replace(valorStr, ",", ".", -1)
		parsedValor, err := strconv.ParseFloat(valorParseable, 64)
		if err != nil {
			log.Printf("Erro ao converter Valor '%s': %v", valorStr, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Valor inválido: formato numérico incorreto."})
			return
		}
		valor = parsedValor
	}

	// Consolidado: Padrão "Não Consolidado" (false) se não marcado
	consolidado := (consolidadoStr == "on") // Se o checkbox está "on", é true, caso contrário é false

	// Inserir no banco de dados
	stmt, err := db.Prepare(fmt.Sprintf(
		`INSERT INTO %s (data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?)`, tableName))
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

	// Redirecionar de volta para a página principal após a adição bem-sucedida
	c.Redirect(http.StatusFound, "/")
}


// jsonify é uma função auxiliar para converter um valor para JSON.
func jsonify(data interface{}) (template.JS, error) {
    b, err := json.Marshal(data)
    if err != nil {
        return "", err
    }
    return template.JS(string(b)), nil
}


func main() {
	var err error

	initDB()
	defer db.Close()

	r := gin.Default()

	// Criar um novo FuncMap e adicionar a função jsonify
    funcMap := template.FuncMap{
        "jsonify": jsonify,
    }
    // SetHTMLTemplate agora usa o FuncMap
    r.SetHTMLTemplate(template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*")))
    log.Println("Templates HTML carregados de 'templates/' com funções customizadas.")


	if _, err = os.Stat("templates"); os.IsNotExist(err) {
		err = os.Mkdir("templates", 0755)
		if err != nil {
			log.Fatalf("Erro ao criar diretório 'templates': %v", err)
		}
	} else if err != nil {
		log.Fatalf("Erro ao verificar diretório 'templates': %v", err)
	}
	log.Println("Diretório 'templates' verificado/criado com sucesso.")

	if _, err = os.Stat("static"); os.IsNotExist(err) {
		err = os.Mkdir("static", 0755)
		if err != nil {
			log.Fatalf("Erro ao criar diretório 'static': %v", err)
		}
	} else if err != nil {
		log.Fatalf("Erro ao verificar diretório 'static': %v", err)
	}
	log.Println("Diretório 'static' verificado/criado com sucesso.")

	if _, err = os.Stat("static/css"); os.IsNotExist(err) {
		err = os.Mkdir("static/css", 0755)
		if err != nil {
			log.Fatalf("Erro ao criar diretório 'static/css': %v", err)
		}
	} else if err != nil {
		log.Fatalf("Erro ao verificar diretório 'static/css': %v", err)
	}
	log.Println("Diretório 'static/css' verificado/criado com sucesso.")

	log.Println("Esperando que 'templates/index.html' e 'static/css/style.css' existam.")

	r.Static("/static", "./static")
	log.Println("Servindo arquivos estáticos de '/static' para './static'.")

	r.GET("/", getMovimentacoes)
	r.GET("/api/movimentacoes", getMovimentacoes)
	r.POST("/movimentacoes", addMovimentacao) // Nova rota para adicionar movimentação

	log.Println("Servidor Gin iniciado na porta :8080")
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("Erro ao iniciar o servidor Gin: %v", err)
	}
}
