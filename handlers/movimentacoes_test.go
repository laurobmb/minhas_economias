package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // Import do driver PostgreSQL
	"minhas_economias/database"
)

// setupTestDB configura um banco de dados PostgreSQL para testes.
// Limpa as tabelas antes de cada execução de teste.
func setupTestDB(t *testing.T) {
	// A função InitDB agora lê das variáveis de ambiente.
	// Assegure-se de que elas apontem para um banco de dados de TESTE.
	_, err := database.InitDB()
	if err != nil {
		t.Fatalf("Falha ao inicializar o banco de dados de teste: %v", err)
	}

	db := database.GetDB()

	// Limpa as tabelas para garantir um estado limpo
	_, err = db.Exec(fmt.Sprintf("TRUNCATE TABLE %s, contas RESTART IDENTITY", database.TableName))
	if err != nil {
		// Se TRUNCATE falhar (ex: primeira execução), tenta criar as tabelas.
		createMovimentacoesSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY, data_ocorrencia DATE NOT NULL,
			descricao TEXT, valor NUMERIC(10, 2), categoria TEXT, conta TEXT, consolidado BOOLEAN DEFAULT FALSE
		);`, database.TableName)
		_, err = db.Exec(createMovimentacoesSQL)
		if err != nil {
			t.Fatalf("Falha ao criar a tabela de teste 'movimentacoes': %v", err)
		}

		createContasSQL := `
		CREATE TABLE IF NOT EXISTS contas (
			nome TEXT PRIMARY KEY, saldo_inicial NUMERIC(10, 2) NOT NULL DEFAULT 0
		);`
		_, err = db.Exec(createContasSQL)
		if err != nil {
			t.Fatalf("Falha ao criar a tabela de teste 'contas': %v", err)
		}
	}

	// Insere dados de teste
	insertMovimentacoesSQL := fmt.Sprintf(`
	INSERT INTO %s (id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES
	($1, $2, $3, $4, $5, $6, $7), ($8, $9, $10, $11, $12, $13, $14);`, database.TableName)

	// Precisamos usar `ALTER SEQUENCE` para reiniciar o ID e inserir valores fixos
	db.Exec(fmt.Sprintf("ALTER SEQUENCE %s_id_seq RESTART WITH 3;", database.TableName))

	_, err = db.Exec(insertMovimentacoesSQL,
		1, "2025-01-10", "Aluguel", -1500.00, "Moradia", "Banco A", true,
		2, "2025-01-15", "Salario", 3000.00, "Renda", "Banco A", true)
	if err != nil {
		t.Fatalf("Falha ao inserir dados de teste em 'movimentacoes': %v", err)
	}
}

// teardownTestDB fecha a conexão com o banco de dados de teste.
func teardownTestDB() {
	database.CloseDB()
}

// createTestRouter cria um roteador Gin para testes e configura o renderizador de HTML.
func createTestRouter() *gin.Engine {
	r := gin.Default()

	// Definindo os templates necessários para os testes
	htmlTemplates := template.Must(template.New("").Parse(`
			{{define "error.html"}}<!DOCTYPE html><html><body><h1>Erro {{.StatusCode}}</h1><p>{{.ErrorMessage}}</p></body></html>{{end}}
			{{define "index.html"}}<!DOCTYPE html><html><body><h1>Minhas Economias - Saldos</h1></body></html>{{end}}
			{{define "transacoes.html"}}<!DOCTYPE html><html><body><h1>Transações Financeiras</h1></body></html>{{end}}
			{{define "relatorio.html"}}<!DOCTYPE html><html><body><h1>Relatório de Despesas por Categoria</h1></body></html>{{end}}
	`))
	r.SetHTMLTemplate(htmlTemplates)

	// Registrando todas as rotas que serão testadas
	r.GET("/", GetIndexPage)
	r.GET("/transacoes", GetTransacoesPage)
	r.GET("/relatorio", GetRelatorio)
	r.POST("/movimentacoes", AddMovimentacao)
	r.POST("/movimentacoes/update/:id", UpdateMovimentacao)
	r.DELETE("/movimentacoes/:id", DeleteMovimentacao)

	return r
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

// TestMain executa setup e teardown para todos os testes.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	code := m.Run()
	os.Exit(code)
}

// --- Funções de Teste (permanecem as mesmas, mas agora rodam em PostgreSQL) ---

func TestGetIndexPage_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	w := performRequest(router, "GET", "/", nil)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d.", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Minhas Economias - Saldos") {
		t.Errorf("Corpo da resposta não contém o título esperado para a página inicial.")
	}
}

func TestGetTransacoesPage_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	w := performRequest(router, "GET", "/transacoes", nil)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d.", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Transações Financeiras") {
		t.Errorf("Corpo da resposta não contém o título esperado para a página de transações.")
	}
}

func TestGetRelatorio_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	w := performRequest(router, "GET", "/relatorio", nil)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d.", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Relatório de Despesas por Categoria") {
		t.Errorf("Corpo da resposta não contém o título esperado para a página de relatório.")
	}
}

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
		t.Errorf("Esperado status 400, mas obteve %d.", w.Code)
	}
	if !strings.Contains(w.Body.String(), "O campo &#39;Conta&#39; é obrigatório.") {
		t.Errorf("Mensagem de erro para conta vazia não encontrada.")
	}
}

func TestUpdateMovimentacao_Validation(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	formData := url.Values{}
	formData.Set("data_ocorrencia", "2025-01-10")
	formData.Set("descricao", strings.Repeat("a", 61))
	formData.Set("valor", "-1500.00")
	formData.Set("conta", "Banco A")

	w := performRequest(router, "POST", "/movimentacoes/update/1", formData)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400, mas obteve %d.", w.Code)
	}
	if !strings.Contains(w.Body.String(), "A descrição não pode ter mais de 60 caracteres.") {
		t.Errorf("Mensagem de erro para descrição longa não encontrada.")
	}
}

func TestAddMovimentacao_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	formData := url.Values{}
	formData.Set("data_ocorrencia", "2025-06-29")
	formData.Set("descricao", "Nova Despesa")
	formData.Set("valor", "-75.00")
	formData.Set("categoria", "Testes")
	formData.Set("conta", "Conta Teste")

	w := performRequest(router, "POST", "/movimentacoes", formData)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302, mas obteve %d.", w.Code)
	}
}

func TestUpdateMovimentacao_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	formData := url.Values{}
	formData.Set("data_ocorrencia", "2025-01-15")
	formData.Set("descricao", "Salario Atualizado")
	formData.Set("valor", "3500.00")
	formData.Set("categoria", "Renda Principal")
	formData.Set("conta", "Banco A")

	w := performRequest(router, "POST", "/movimentacoes/update/2", formData)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302, mas obteve %d.", w.Code)
	}
}

func TestDeleteMovimentacao_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	w := performRequest(router, "DELETE", "/movimentacoes/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d.", w.Code)
	}
}