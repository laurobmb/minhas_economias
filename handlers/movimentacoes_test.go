// handlers/movimentacoes_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
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
	_, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Falha ao inicializar o banco de dados em memória: %v", err)
	}

	db := database.GetDB()

	// Cria a tabela 'movimentacoes'
	createMovimentacoesSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data_ocorrencia TEXT NOT NULL,
			descricao TEXT,
			valor REAL,
			categoria TEXT,
			conta TEXT,
			consolidado BOOLEAN DEFAULT FALSE
		);`, database.TableName)
	_, err = db.Exec(createMovimentacoesSQL)
	if err != nil {
		t.Fatalf("Falha ao criar a tabela de teste 'movimentacoes': %v", err)
	}

	// ALTERAÇÃO: Cria a nova tabela 'contas'
	createContasSQL := `
		CREATE TABLE IF NOT EXISTS contas (
			nome TEXT PRIMARY KEY,
			saldo_inicial REAL NOT NULL DEFAULT 0
		);`
	_, err = db.Exec(createContasSQL)
	if err != nil {
		t.Fatalf("Falha ao criar a tabela de teste 'contas': %v", err)
	}

	// Popula a tabela 'movimentacoes' com dados de teste
	insertMovimentacoesSQL := fmt.Sprintf(`
		INSERT INTO %s (data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES
		(?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?), 
		(?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?);
	`, database.TableName)
	_, err = db.Exec(insertMovimentacoesSQL,
		"2025-01-10", "Aluguel", -1500.00, "Moradia", "Banco A", true,
		"2025-01-15", "Salario", 3000.00, "Renda", "Banco A", true,
		"2025-01-20", "Supermercado", -250.50, "Alimentacao", "Cartao B", false,
		"2025-02-05", "Transporte", -50.00, "Transporte", "Banco A", true,
		"2025-02-10", "Restaurante", -80.00, "Alimentacao", "Cartao B", true,
		"2025-03-01", "Bonus", 500.00, "Renda Extra", "Banco A", false,
	)
	if err != nil {
		t.Fatalf("Falha ao inserir dados de teste em 'movimentacoes': %v", err)
	}

	// ALTERAÇÃO: Popula a tabela 'contas' com saldos iniciais de teste
	insertContasSQL := `INSERT INTO contas (nome, saldo_inicial) VALUES (?, ?), (?, ?);`
	_, err = db.Exec(insertContasSQL, "Banco A", 100.0, "Cartao B", -50.0)
	if err != nil {
		t.Fatalf("Falha ao inserir dados de teste em 'contas': %v", err)
	}
}

// teardownTestDB fecha a conexão com o banco de dados em memória.
func teardownTestDB() {
	database.CloseDB()
}

// createTestRouter cria um roteador Gin para testes e configura o renderizador de HTML.
func createTestRouter() *gin.Engine {
	r := gin.Default()

	funcMap := template.FuncMap{
		"jsonify": func(data interface{}) (template.JS, error) {
			b, err := json.Marshal(data)
			if err != nil {
				return "", err
			}
			return template.JS(string(b)), nil
		},
	}

	// ALTERAÇÃO: Adicionado o template 'transacoes.html' para os testes
	htmlTemplates := template.Must(template.New("index.html").Funcs(funcMap).Parse(`
		{{define "index.html"}}<!DOCTYPE html><html><body><h1>{{ .Titulo }}</h1></body></html>{{end}}
		{{define "transacoes.html"}}<!DOCTYPE html><html><body><h1>{{ .Titulo }}</h1></body></html>{{end}}
		{{define "relatorio.html"}}<!DOCTYPE html><html><head><title>{{ .Titulo }}</title><canvas id="expensesPieChart"></canvas></body></html>{{end}}
	`))
	r.SetHTMLTemplate(htmlTemplates)

	// ALTERAÇÃO: Rotas atualizadas para refletir a nova estrutura da aplicação
	r.GET("/", GetIndexPage)
	r.GET("/transacoes", GetTransacoesPage)
	r.GET("/api/movimentacoes", GetTransacoesPage)
	r.POST("/movimentacoes", AddMovimentacao)
	r.DELETE("/movimentacoes/:id", DeleteMovimentacao)
	r.POST("/movimentacoes/update/:id", UpdateMovimentacao)
	r.GET("/relatorio", GetRelatorio)
	r.GET("/relatorio/transactions", GetTransactionsByCategory)
	return r
}

// NOVO TESTE: TestGetIndexPage testa a nova página inicial de saldos.
func TestGetIndexPage(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Minhas Economias - Saldos") {
		t.Errorf("O corpo da resposta HTML não contém o título esperado para a página de saldos. Conteúdo: %s", w.Body.String())
	}
}

// ALTERAÇÃO: Teste renomeado e ajustado para a nova página de transações.
func TestGetTransacoesPage(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	// Teste 1: Página de transações com filtros
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/transacoes?start_date=2025-01-01&end_date=2025-01-31", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Transações Financeiras") {
		t.Errorf("O corpo da resposta HTML não contém o título esperado para a página de transações. Conteúdo: %s", w.Body.String())
	}

	// Teste 2: API com filtro de categoria
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/movimentacoes?category=Alimentacao&start_date=2025-01-01&end_date=2025-01-31", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK para API, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	var apiResponse struct {
		Movimentacoes []models.Movimentacao `json:"movimentacoes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResponse); err != nil {
		t.Fatalf("Falha ao fazer unmarshal da resposta JSON da API: %v. Corpo: %s", err, w.Body.String())
	}
	if len(apiResponse.Movimentacoes) != 1 {
		t.Errorf("Esperado 1 movimentação para 'Alimentacao' em janeiro, obteve %d.", len(apiResponse.Movimentacoes))
	}
}

// TestAddMovimentacao testa a função AddMovimentacao.
func TestAddMovimentacao(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	formData := url.Values{}
	testDate := "2025-06-29"
	formData.Set("data_ocorrencia", testDate)
	formData.Set("descricao", "Nova Despesa de Teste")
	formData.Set("valor", "-75.00")
	formData.Set("categoria", "Testes")
	formData.Set("conta", "Conta Teste")
	formData.Set("consolidado", "on")

	w := performRequest(router, "POST", "/movimentacoes", formData)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302 Found, mas obteve %d.", w.Code)
	}

	// ALTERAÇÃO: Verifica se o redirecionamento foi para a página correta
	location := w.Header().Get("Location")
	if location != "/transacoes" {
		t.Errorf("Esperado redirecionamento para '/transacoes', mas foi para '%s'", location)
	}

	// Verificar se o item foi realmente adicionado
	w = performRequest(router, "GET", fmt.Sprintf("/api/movimentacoes?category=Testes&start_date=%s&end_date=%s", testDate, testDate), nil)
	var apiResponse struct {
		Movimentacoes []models.Movimentacao `json:"movimentacoes"`
	}
	json.Unmarshal(w.Body.Bytes(), &apiResponse)
	if len(apiResponse.Movimentacoes) != 1 {
		t.Errorf("Esperado 1 movimentação adicionada, obteve %d.", len(apiResponse.Movimentacoes))
	}
}

// TestUpdateMovimentacao testa a função UpdateMovimentacao.
func TestUpdateMovimentacao(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	movID := 2
	originalDate := "2025-01-15"
	formData := url.Values{}
	formData.Set("data_ocorrencia", originalDate)
	formData.Set("descricao", "Salario Atualizado")
	formData.Set("valor", "3500.00")
	formData.Set("categoria", "Renda Principal")
	formData.Set("conta", "Banco A")
	formData.Set("consolidado", "off")

	w := performRequest(router, "POST", fmt.Sprintf("/movimentacoes/update/%d", movID), formData)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302 Found, mas obteve %d.", w.Code)
	}

	// ALTERAÇÃO: Verifica se o redirecionamento foi para a página correta
	location := w.Header().Get("Location")
	if location != "/transacoes" {
		t.Errorf("Esperado redirecionamento para '/transacoes', mas foi para '%s'", location)
	}
}

// TestDeleteMovimentacao testa a função DeleteMovimentacao.
func TestDeleteMovimentacao(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()
	movID := 1

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/movimentacoes/%d", movID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d.", w.Code)
	}
}


// O resto dos testes (GetRelatorio, validações, etc.) não precisam de grandes mudanças e foram mantidos.

// TestGetRelatorio testa a função GetRelatorio.
func TestGetRelatorio(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB()

    router := createTestRouter()

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/relatorio?start_date=2025-01-01&end_date=2025-02-28", nil)
    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
    }
    if !strings.Contains(w.Body.String(), "Relatório de Despesas por Categoria") {
        t.Errorf("O corpo da resposta HTML não contém o título do relatório esperado. Conteúdo: %s", w.Body.String())
    }
}

// TestMain executa setup e teardown para todos os testes.
func TestMain(m *testing.M) {
    gin.SetMode(gin.ReleaseMode)
    code := m.Run()
    os.Exit(code)
}

// Helper para simular requisições HTTP.
func performRequest(r http.Handler, method, path string, body url.Values) *httptest.ResponseRecorder {
    var req *http.Request
    var err error

    if body != nil {
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
    
    formData := url.Values{}
    formData.Set("data_ocorrencia", "2025-06-30")
    formData.Set("descricao", "Despesa sem conta")
    formData.Set("valor", "-10.00")
    
    w := performRequest(router, "POST", "/movimentacoes", formData)

    if w.Code != http.StatusBadRequest {
        t.Errorf("Esperado status 400 Bad Request para conta vazia, mas obteve %d.", w.Code)
    }
    if !strings.Contains(w.Body.String(), "O campo 'Conta' é obrigatório.") {
        t.Errorf("Mensagem de erro esperada para conta vazia não encontrada.")
    }
}

// TestUpdateMovimentacao_Validation testa as validações da função UpdateMovimentacao.
func TestUpdateMovimentacao_Validation(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB()
    router := createTestRouter()
    movID := 1
    
    formData := url.Values{}
    formData.Set("data_ocorrencia", "2025-01-10")
    formData.Set("valor", "xyz")
    formData.Set("conta", "Banco A")

    w := performRequest(router, "POST", fmt.Sprintf("/movimentacoes/update/%d", movID), formData)

    if w.Code != http.StatusBadRequest {
        t.Errorf("Esperado status 400 Bad Request para valor inválido, mas obteve %d.", w.Code)
    }
    if !strings.Contains(w.Body.String(), "Valor inválido: formato numérico incorreto.") {
        t.Errorf("Mensagem de erro esperada para valor inválido não encontrada.")
    }
}