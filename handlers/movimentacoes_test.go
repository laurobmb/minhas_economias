// handlers/movimentacoes_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template" // Re-adicionada para configurar o renderizador de HTML
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"minhas_economias/database"
	"minhas_economias/models"
)

// setupTestDB configura um banco de dados SQLite em memória para testes.
func setupTestDB(t *testing.T) {
	// Inicializa o DB em memória.
	// O nome do DB ":memory:" cria um DB temporário que é destruído ao fechar a conexão.
	_, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Falha ao inicializar o banco de dados em memória: %v", err)
	}

	// Obtém a conexão do DB.
	db := database.GetDB()

	// Cria a tabela 'movimentacoes' no DB em memória.
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data_ocorrencia TEXT NOT NULL,
			descricao TEXT,
			valor REAL,
			categoria TEXT,
			conta TEXT,
			consolidado BOOLEAN DEFAULT FALSE
		);`, database.TableName)

	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("Falha ao criar a tabela de teste: %v", err)
	}

	// Popula o DB com dados de teste.
	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES
		(?, ?, ?, ?, ?, ?),
		(?, ?, ?, ?, ?, ?),
		(?, ?, ?, ?, ?, ?),
		(?, ?, ?, ?, ?, ?),
		(?, ?, ?, ?, ?, ?),
		(?, ?, ?, ?, ?, ?);
	`, database.TableName)

	_, err = db.Exec(insertSQL,
		"2025-01-10", "Aluguel", -1500.00, "Moradia", "Banco A", true,
		"2025-01-15", "Salario", 3000.00, "Renda", "Banco A", true,
		"2025-01-20", "Supermercado", -250.50, "Alimentacao", "Cartao B", false,
		"2025-02-05", "Transporte", -50.00, "Transporte", "Banco A", true,
		"2025-02-10", "Restaurante", -80.00, "Alimentacao", "Cartao B", true,
		"2025-03-01", "Bonus", 500.00, "Renda Extra", "Banco A", false,
	)
	if err != nil {
		t.Fatalf("Falha ao inserir dados de teste: %v", err)
	}
}

// teardownTestDB fecha a conexão com o banco de dados em memória.
func teardownTestDB() {
	database.CloseDB()
}

// createTestRouter cria um roteador Gin para testes e configura o renderizador de HTML.
func createTestRouter() *gin.Engine {
	r := gin.Default()

	// Configurar um renderizador de HTML para testes.
	// Usamos um template mínimo para 'index.html' e 'relatorio.html'
	// e registramos a função 'jsonify' (que é usada nos templates).
	funcMap := template.FuncMap{
		"jsonify": func(data interface{}) (template.JS, error) {
			b, err := json.Marshal(data)
			if err != nil {
				return "", err
			}
			return template.JS(string(b)), nil
		},
	}

	// Cria um novo template e associa as funções, depois parseia os templates.
	// É importante que o nome do template corresponda ao que é chamado em c.HTML.
	// Para testes, podemos usar um único string que define ambos os templates.
	htmlTemplates := template.Must(template.New("index.html").Funcs(funcMap).Parse(`
		{{define "index.html"}}<!DOCTYPE html><html><head><title>{{ .Titulo }}</title></head><body><h1>{{ .Titulo }}</h1></body></html>{{end}}
		{{define "relatorio.html"}}<!DOCTYPE html><html><head><title>{{ .Titulo }}</title><canvas id="expensesPieChart"></canvas></body></html>{{end}}
	`))
	r.SetHTMLTemplate(htmlTemplates)

	// Configura as rotas que serão testadas.
	r.GET("/", GetMovimentacoes)
	r.GET("/api/movimentacoes", GetMovimentacoes) // A mesma função para API
	r.POST("/movimentacoes", AddMovimentacao)
	r.DELETE("/movimentacoes/:id", DeleteMovimentacao)
	r.POST("/movimentacoes/update/:id", UpdateMovimentacao)
	r.GET("/relatorio", GetRelatorio)
	r.GET("/relatorio/transactions", GetTransactionsByCategory)
	return r
}

// TestGetMovimentacoes testa a função GetMovimentacoes (página principal e API).
func TestGetMovimentacoes(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	// Teste 1: Página principal com filtros de data para garantir dados determinísticos
	// Dados de teste: 2025-01-10, 2025-01-15, 2025-01-20
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?start_date=2025-01-01&end_date=2025-01-31", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	// O handler passa "Extratos Financeiros" para o título.
	if !strings.Contains(w.Body.String(), "Extratos Financeiros") {
		t.Errorf("O corpo da resposta HTML não contém o título esperado. Conteúdo: %s", w.Body.String())
	}

	// Teste 2: API com filtro de categoria (e datas para ser determinístico)
	w = httptest.NewRecorder()
	// Dados de teste: Supermercado (2025-01-20), Restaurante (2025-02-10)
	// Vamos filtrar para janeiro para pegar apenas o Supermercado
	req, _ = http.NewRequest("GET", "/api/movimentacoes?category=Alimentacao&start_date=2025-01-01&end_date=2025-01-31", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	var apiResponse struct {
		Movimentacoes []models.Movimentacao `json:"movimentacoes"`
		TotalValor    float64               `json:"totalValor"`
		TotalEntradas float64               `json:"totalEntradas"`
		TotalSaidas   float64               `json:"totalSaidas"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResponse); err != nil {
		t.Fatalf("Falha ao fazer unmarshal da resposta JSON: %v. Corpo: %s", err, w.Body.String())
	}

	if len(apiResponse.Movimentacoes) != 1 { // Apenas Supermercado
		t.Errorf("Esperado 1 movimentação para 'Alimentacao' em janeiro, obteve %d. Movimentacoes: %+v", len(apiResponse.Movimentacoes), apiResponse.Movimentacoes)
	}
	if len(apiResponse.Movimentacoes) > 0 && apiResponse.Movimentacoes[0].Categoria != "Alimentacao" {
		t.Errorf("Categoria esperada 'Alimentacao', obteve '%s'", apiResponse.Movimentacoes[0].Categoria)
	}
}

// TestAddMovimentacao testa a função AddMovimentacao.
func TestAddMovimentacao(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	// Dados válidos para adicionar
	formData := url.Values{}
	testDate := "2025-06-29" // Data da nova movimentação
	formData.Set("data_ocorrencia", testDate)
	formData.Set("descricao", "Nova Despesa de Teste")
	formData.Set("valor", "-75.00")
	formData.Set("categoria", "Testes")
	formData.Set("conta", "Conta Teste")
	formData.Set("consolidado", "on")

	w := performRequest(router, "POST", "/movimentacoes", formData)

	if w.Code != http.StatusFound { // Espera redirecionamento
		t.Errorf("Esperado status 302 Found, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	// Verificar se o item foi realmente adicionado (via API), usando filtro de data
	w = performRequest(router, "GET", fmt.Sprintf("/api/movimentacoes?category=Testes&start_date=%s&end_date=%s", testDate, testDate), nil)

	var apiResponse struct {
		Movimentacoes []models.Movimentacao `json:"movimentacoes"`
	}
	json.Unmarshal(w.Body.Bytes(), &apiResponse)

	if len(apiResponse.Movimentacoes) != 1 {
		t.Errorf("Esperado 1 movimentação adicionada, obteve %d. Movimentacoes: %+v", len(apiResponse.Movimentacoes), apiResponse.Movimentacoes)
	}
	if len(apiResponse.Movimentacoes) > 0 && apiResponse.Movimentacoes[0].Descricao != "Nova Despesa de Teste" {
		t.Errorf("Descrição esperada 'Nova Despesa de Teste', obteve '%s'", apiResponse.Movimentacoes[0].Descricao)
	}

	// Teste de valor padrão 0 e categoria "Sem Categoria"
	formData = url.Values{}
	testDate2 := "2025-07-01" // Data da nova movimentação
	formData.Set("data_ocorrencia", testDate2)
	formData.Set("descricao", "Item sem valor e categoria")
	// formData.Set("valor", "") // Valor vazio
	// formData.Set("categoria", "") // Categoria vazia
	formData.Set("conta", "Conta Default")
	formData.Set("consolidado", "off")

	w = performRequest(router, "POST", "/movimentacoes", formData)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302 Found para valores padrão, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	// Verificar se o item foi adicionado com valores padrão, usando filtro de data
	w = performRequest(router, "GET", fmt.Sprintf("/api/movimentacoes?descricao=%s&start_date=%s&end_date=%s", url.QueryEscape("Item sem valor e categoria"), testDate2, testDate2), nil)

	json.Unmarshal(w.Body.Bytes(), &apiResponse)
	if len(apiResponse.Movimentacoes) != 1 {
		t.Errorf("Esperado 1 movimentação com valores padrão, obteve %d. Movimentacoes: %+v", len(apiResponse.Movimentacoes), apiResponse.Movimentacoes)
	}
	if len(apiResponse.Movimentacoes) > 0 && apiResponse.Movimentacoes[0].Valor != 0.0 {
		t.Errorf("Valor esperado 0.0, obteve %.2f", apiResponse.Movimentacoes[0].Valor)
	}
	if len(apiResponse.Movimentacoes) > 0 && apiResponse.Movimentacoes[0].Categoria != "Sem Categoria" {
		t.Errorf("Categoria esperada 'Sem Categoria', obteve '%s'", apiResponse.Movimentacoes[0].Categoria)
	}
}

// TestUpdateMovimentacao testa a função UpdateMovimentacao.
func TestUpdateMovimentacao(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	// ID de uma movimentação existente nos dados de teste (ex: Salario, ID 2)
	movID := 2
	originalDate := "2025-01-15" // Data original do item

	// Dados para atualização
	formData := url.Values{}
	formData.Set("data_ocorrencia", originalDate) // Mantém a mesma data
	formData.Set("descricao", "Salario Atualizado")
	formData.Set("valor", "3500.00")
	formData.Set("categoria", "Renda Principal")
	formData.Set("conta", "Banco A")
	formData.Set("consolidado", "off")

	w := performRequest(router, "POST", fmt.Sprintf("/movimentacoes/update/%d", movID), formData)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302 Found, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	// Verificar se o item foi realmente atualizado, usando filtro de data
	w = performRequest(router, "GET", fmt.Sprintf("/api/movimentacoes?descricao=%s&start_date=%s&end_date=%s", url.QueryEscape("Salario Atualizado"), originalDate, originalDate), nil)

	var apiResponse struct {
		Movimentacoes []models.Movimentacao `json:"movimentacoes"`
	}
	json.Unmarshal(w.Body.Bytes(), &apiResponse)

	if len(apiResponse.Movimentacoes) != 1 {
		t.Fatalf("Esperado 1 movimentação atualizada, obteve %d. Movimentacoes: %+v", len(apiResponse.Movimentacoes), apiResponse.Movimentacoes)
	}
	if len(apiResponse.Movimentacoes) > 0 && apiResponse.Movimentacoes[0].ID != movID {
		t.Errorf("ID esperado %d, obteve %d", movID, apiResponse.Movimentacoes[0].ID)
	}
	if len(apiResponse.Movimentacoes) > 0 && apiResponse.Movimentacoes[0].Valor != 3500.00 {
		t.Errorf("Valor esperado 3500.00, obteve %.2f", apiResponse.Movimentacoes[0].Valor)
	}
	if len(apiResponse.Movimentacoes) > 0 && apiResponse.Movimentacoes[0].Categoria != "Renda Principal" {
		t.Errorf("Categoria esperada 'Renda Principal', obteve '%s'", apiResponse.Movimentacoes[0].Categoria)
	}
	if len(apiResponse.Movimentacoes) > 0 && apiResponse.Movimentacoes[0].Consolidado != false {
		t.Errorf("Consolidado esperado false, obteve %t", apiResponse.Movimentacoes[0].Consolidado)
	}
}

// TestDeleteMovimentacao testa a função DeleteMovimentacao.
func TestDeleteMovimentacao(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	// ID de uma movimentação existente para deletar (ex: Aluguel, ID 1)
	movID := 1
	originalDate := "2025-01-10" // Data original do item

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/movimentacoes/%d", movID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	// Verificar se o item foi realmente deletado (via API), usando filtro de data
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/movimentacoes?id=%d&start_date=%s&end_date=%s", movID, originalDate, originalDate), nil)
	router.ServeHTTP(w, req)

	var apiResponse struct {
		Movimentacoes []models.Movimentacao `json:"movimentacoes"`
	}
	json.Unmarshal(w.Body.Bytes(), &apiResponse)

	if len(apiResponse.Movimentacoes) != 0 {
		t.Errorf("Esperado 0 movimentações após exclusão, obteve %d", len(apiResponse.Movimentacoes))
	}
}

// TestGetRelatorio testa a função GetRelatorio.
func TestGetRelatorio(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	// Teste 1: Página de relatório com filtros de data para garantir dados determinísticos
	// Dados de teste: Aluguel (-1500.00, Moradia), Supermercado (-250.50, Alimentacao), Restaurante (-80.00, Alimentacao)
	// Vamos filtrar para janeiro-fevereiro para pegar Moradia e Alimentacao (Supermercado e Restaurante)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/relatorio?start_date=2025-01-01&end_date=2025-02-28", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	// O handler passa "Relatório de Despesas por Categoria" para o título.
	if !strings.Contains(w.Body.String(), "Relatório de Despesas por Categoria") {
		t.Errorf("O corpo da resposta HTML não contém o título do relatório esperado. Conteúdo: %s", w.Body.String())
	}

	// Teste 2: API de transações por categoria (usada pelo relatório no clique)
	w = httptest.NewRecorder()
	// Vamos buscar as transações da categoria "Alimentacao" no período de janeiro-fevereiro
	req, _ = http.NewRequest("GET", "/relatorio/transactions?category=Alimentacao&start_date=2025-01-01&end_date=2025-02-28", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK para transações por categoria, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	var transactions []models.Movimentacao
	if err := json.Unmarshal(w.Body.Bytes(), &transactions); err != nil {
		t.Fatalf("Falha ao fazer unmarshal das transações por categoria: %v. Corpo: %s", err, w.Body.String())
	}

	// Esperamos 2 transações de Alimentacao (Supermercado e Restaurante)
	if len(transactions) != 2 {
		t.Errorf("Esperado 2 transações para 'Alimentacao', obteve %d", len(transactions))
	}
	if transactions[0].Categoria != "Alimentacao" || transactions[1].Categoria != "Alimentacao" {
		t.Errorf("Categorias esperadas 'Alimentacao', mas obteve categorias diferentes.")
	}
}

// TestGetTransactionsByCategory testa a função GetTransactionsByCategory.
func TestGetTransactionsByCategory(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	// Teste: buscar transações de uma categoria específica com filtros
	w := httptest.NewRecorder()
	// Exemplo: buscar despesas de "Alimentacao" entre 2025-01-01 e 2025-01-31 da conta "Cartao B"
	req, _ := http.NewRequest("GET", "/relatorio/transactions?category=Alimentacao&start_date=2025-01-01&end_date=2025-01-31&account=Cartao+B", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	var transactions []models.Movimentacao
	if err := json.Unmarshal(w.Body.Bytes(), &transactions); err != nil {
		t.Fatalf("Falha ao fazer unmarshal das transações: %v. Corpo: %s", err, w.Body.String())
	}

	// Com base nos dados de teste, esperamos 1 transação: Supermercado (-250.50)
	if len(transactions) != 1 {
		t.Errorf("Esperado 1 transação para os filtros, obteve %d", len(transactions))
	}
	if transactions[0].Descricao != "Supermercado" {
		t.Errorf("Descrição esperada 'Supermercado', obteve '%s'", transactions[0].Descricao)
	}
	if transactions[0].Valor != -250.50 {
		t.Errorf("Valor esperado -250.50, obteve %.2f", transactions[0].Valor)
	}

	// Teste: buscar categoria que não existe ou sem transações
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/relatorio/transactions?category=CategoriaInexistente", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK para categoria inexistente, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	if err := json.Unmarshal(w.Body.Bytes(), &transactions); err != nil {
		t.Fatalf("Falha ao fazer unmarshal de resposta vazia: %v", err)
	}
	if len(transactions) != 0 {
		t.Errorf("Esperado 0 transações para categoria inexistente, obteve %d", len(transactions))
	}
}

// TestMain executa setup e teardown para todos os testes.
func TestMain(m *testing.M) {
	// Desabilitar o modo debug do Gin para não poluir a saída do teste.
	gin.SetMode(gin.ReleaseMode)

	// Executa os testes.
	code := m.Run()

	// Limpeza (se houver algo global para limpar após todos os testes).
	// No nosso caso, o DB em memória é limpo por teardownTestDB em cada teste.

	os.Exit(code)
}

// Helper para simular requisições HTTP.
// O parâmetro 'body' é um url.Values para requisições POST/PUT com form-urlencoded.
// Para GET/DELETE sem corpo, passe nil para 'body'.
func performRequest(r http.Handler, method, path string, body url.Values) *httptest.ResponseRecorder {
	var req *http.Request
	var err error

	if body != nil && (method == "POST" || method == "PUT") {
		req, err = http.NewRequest(method, path, bytes.NewBufferString(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest(method, path, nil)
	}

	if err != nil {
		panic(fmt.Sprintf("Failed to create request: %v", err))
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestAddMovimentacao_Validation testa as validações da função AddMovimentacao.
func TestAddMovimentacao_Validation(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	// Teste: Conta obrigatória
	formData := url.Values{}
	formData.Set("data_ocorrencia", "2025-06-30")
	formData.Set("descricao", "Despesa sem conta")
	formData.Set("valor", "-10.00")
	formData.Set("categoria", "Diversos")
	// formData.Set("conta", "") // Conta vazia
	formData.Set("consolidado", "off")

	w := performRequest(router, "POST", "/movimentacoes", formData)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400 Bad Request para conta vazia, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "O campo 'Conta' é obrigatório.") {
		t.Errorf("Mensagem de erro esperada para conta vazia não encontrada.")
	}

	// Teste: Valor inválido
	formData = url.Values{}
	formData.Set("data_ocorrencia", "2025-06-30")
	formData.Set("descricao", "Despesa com valor inválido")
	formData.Set("valor", "abc") // Valor inválido
	formData.Set("categoria", "Diversos")
	formData.Set("conta", "Conta Valida")
	formData.Set("consolidado", "off")

	w = performRequest(router, "POST", "/movimentacoes", formData)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400 Bad Request para valor inválido, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Valor inválido: formato numérico incorreto.") {
		t.Errorf("Mensagem de erro esperada para valor inválido não encontrada.")
	}
}

// TestUpdateMovimentacao_Validation testa as validações da função UpdateMovimentacao.
func TestUpdateMovimentacao_Validation(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	movID := 1 // ID de uma movimentação existente

	// Teste: Conta obrigatória na atualização
	formData := url.Values{}
	formData.Set("data_ocorrencia", "2025-01-10")
	formData.Set("descricao", "Aluguel Atualizado")
	formData.Set("valor", "-1500.00")
	formData.Set("categoria", "Moradia")
	// formData.Set("conta", "") // Conta vazia
	formData.Set("consolidado", "on")

	w := performRequest(router, "POST", fmt.Sprintf("/movimentacoes/update/%d", movID), formData)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400 Bad Request para conta vazia na atualização, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "O campo 'Conta' é obrigatório.") {
		t.Errorf("Mensagem de erro esperada para conta vazia na atualização não encontrada.")
	}

	// Teste: Valor inválido na atualização
	formData = url.Values{}
	formData.Set("data_ocorrencia", "2025-01-10")
	formData.Set("descricao", "Aluguel Atualizado")
	formData.Set("valor", "xyz") // Valor inválido
	formData.Set("categoria", "Moradia")
	formData.Set("conta", "Banco A")
	formData.Set("consolidado", "on")

	w = performRequest(router, "POST", fmt.Sprintf("/movimentacoes/update/%d", movID), formData)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400 Bad Request para valor inválido na atualização, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Valor inválido: formato numérico incorreto.") {
		t.Errorf("Mensagem de erro esperada para valor inválido na atualização não encontrada.")
	}
}