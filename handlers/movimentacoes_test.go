// handlers/movimentacoes_test.go
package handlers

import (
	"bytes"
	// "encoding/json" // REMOVIDO
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
	// "minhas_economias/models" // REMOVIDO
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

	// Cria a nova tabela 'contas'
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

	// Popula a tabela 'contas' com saldos iniciais de teste
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

	// Adicionado o template 'transacoes.html' e 'sobre.html' para os testes
	htmlTemplates := template.Must(template.New("index.html").Parse(`
		{{define "index.html"}}<!DOCTYPE html><html><body><h1>{{ .Titulo }}</h1></body></html>{{end}}
		{{define "transacoes.html"}}<!DOCTYPE html><html><body><h1>{{ .Titulo }}</h1></body></html>{{end}}
		{{define "relatorio.html"}}<!DOCTYPE html><html><head><title>{{ .Titulo }}</title></head><body></body></html>{{end}}
        {{define "sobre.html"}}<!DOCTYPE html><html><body><h1>{{ .Titulo }}</h1></body></html>{{end}}
	`))
	r.SetHTMLTemplate(htmlTemplates)

	// Rotas atualizadas para refletir a nova estrutura da aplicação
	r.GET("/", GetIndexPage)
	r.GET("/transacoes", GetTransacoesPage)
	r.GET("/api/movimentacoes", GetTransacoesPage)
	r.GET("/sobre", GetSobrePage)
	r.POST("/movimentacoes", AddMovimentacao)
	r.DELETE("/movimentacoes/:id", DeleteMovimentacao)
	r.POST("/movimentacoes/update/:id", UpdateMovimentacao)
	r.GET("/relatorio", GetRelatorio)
	r.GET("/relatorio/transactions", GetTransactionsByCategory)
	return r
}

// TestGetIndexPage testa a nova página inicial de saldos.
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

// TestGetTransacoesPage testa a nova página de transações.
func TestGetTransacoesPage(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := createTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/transacoes?start_date=2025-01-01&end_date=2025-01-31", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 OK, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Transações Financeiras") {
		t.Errorf("O corpo da resposta HTML não contém o título esperado para a página de transações. Conteúdo: %s", w.Body.String())
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
	location := w.Header().Get("Location")
	if location != "/transacoes" {
		t.Errorf("Esperado redirecionamento para '/transacoes', mas foi para '%s'", location)
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

// TestAddMovimentacao_Validation testa as validações de erro no formulário.
func TestAddMovimentacao_Validation(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	// Teste: Conta obrigatória
	formData := url.Values{}
	formData.Set("data_ocorrencia", "2025-06-30")
	formData.Set("descricao", "Despesa sem conta")
	formData.Set("valor", "-10.00")
	// formData.Set("conta", "") // Conta vazia

	w := performRequest(router, "POST", "/movimentacoes", formData)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400 Bad Request para conta vazia, mas obteve %d.", w.Code)
	}
	if !strings.Contains(w.Body.String(), "O campo 'Conta' é obrigatório.") {
		t.Errorf("Mensagem de erro esperada para conta vazia não encontrada.")
	}
}

// TestAddMovimentacao_Sanitization testa as novas validações de campos.
func TestAddMovimentacao_Sanitization(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	// Casos de teste para as novas validações
	testCases := []struct {
		name                 string
		formData             url.Values
		expectedStatusCode   int
		expectedErrorMessage string
	}{
		{
			name: "Descrição muito longa",
			formData: url.Values{
				"data_ocorrencia": {"2025-07-20"},
				"descricao":       {strings.Repeat("a", 61)}, // > 60 caracteres
				"valor":           {"10.00"},
				"conta":           {"Conta Teste"},
			},
			expectedStatusCode:   http.StatusBadRequest,
			expectedErrorMessage: "A descrição não pode ter mais de 60 caracteres.",
		},
		{
			name: "Valor com muitas casas decimais",
			formData: url.Values{
				"data_ocorrencia": {"2025-07-20"},
				"descricao":       {"Decimais a mais"},
				"valor":           {"123.456"},
				"conta":           {"Conta Teste"},
			},
			expectedStatusCode:   http.StatusBadRequest,
			expectedErrorMessage: "Valor inválido. Use um formato como 1234.56 ou -123.45.",
		},
		{
			name: "Valor excede o limite máximo",
			formData: url.Values{
				"data_ocorrencia": {"2025-07-20"},
				"descricao":       {"Valor muito alto"},
				"valor":           {"100000000"}, // 100 milhões
				"conta":           {"Conta Teste"},
			},
			expectedStatusCode:   http.StatusBadRequest,
			expectedErrorMessage: "O valor excede o limite máximo permitido.",
		},
		{
			name: "Valor com formato de texto",
			formData: url.Values{
				"data_ocorrencia": {"2025-07-20"},
				"descricao":       {"Valor com texto"},
				"valor":           {"abc"},
				"conta":           {"Conta Teste"},
			},
			expectedStatusCode:   http.StatusBadRequest,
			expectedErrorMessage: "Valor inválido. Use um formato como 1234.56 ou -123.45.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := performRequest(router, "POST", "/movimentacoes", tc.formData)

			if w.Code != tc.expectedStatusCode {
				t.Errorf("Esperado status %d, mas obteve %d. Corpo: %s", tc.expectedStatusCode, w.Code, w.Body.String())
			}

			if !strings.Contains(w.Body.String(), tc.expectedErrorMessage) {
				t.Errorf("Mensagem de erro esperada '%s' não encontrada no corpo: %s", tc.expectedErrorMessage, w.Body.String())
			}
		})
	}
}