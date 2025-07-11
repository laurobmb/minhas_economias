package investimentos

import (
	"bytes"
	"encoding/json"
	// "fmt" foi removido pois não estava sendo utilizado
	"minhas_economias/auth"
	"minhas_economias/database"
	"minhas_economias/models"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3" // Driver para o banco de dados de teste
)

// testUserID é o ID do usuário que usaremos para todos os testes de investimento.
const testUserID int64 = 1001

// mockAuthMiddleware simula um usuário logado para os testes.
func mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		testUser := &models.User{
			ID:              testUserID,
			Email:           "test.invest@user.com",
			IsAdmin:         false,
			DarkModeEnabled: false,
		}
		c.Set("userID", testUserID)
		c.Set("user", testUser)
		c.Next()
	}
}

// setupInvestimentosTestDB configura um banco de dados de teste limpo.
func setupInvestimentosTestDB(t *testing.T) {
	// Força o uso do SQLite em memória para os testes
	os.Setenv("DB_TYPE", "sqlite3")
	os.Setenv("DB_NAME", "file::memory:?cache=shared")

	_, err := database.InitDB()
	if err != nil {
		t.Fatalf("Falha ao inicializar o banco de dados de teste: %v", err)
	}
	db := database.GetDB()

	// Cria as tabelas necessárias
	createTablesSQL := `
        CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT UNIQUE, password_hash TEXT, is_admin BOOLEAN, dark_mode_enabled BOOLEAN);
        CREATE TABLE investimentos_nacionais (user_id INTEGER, ticker TEXT, tipo TEXT, quantidade INTEGER, PRIMARY KEY (user_id, ticker));
        CREATE TABLE investimentos_internacionais (user_id INTEGER, ticker TEXT, descricao TEXT, quantidade REAL, moeda TEXT, PRIMARY KEY (user_id, ticker));
    `
	if _, err := db.Exec(createTablesSQL); err != nil {
		t.Fatalf("Falha ao criar tabelas de teste: %v", err)
	}

	// Insere o usuário de teste
	hashedPassword, _ := auth.HashPassword("password123")
	db.Exec("INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)", testUserID, "test.invest@user.com", hashedPassword)

	// Insere dados iniciais para testes de update e delete
	db.Exec("INSERT INTO investimentos_nacionais (user_id, ticker, tipo, quantidade) VALUES (?, ?, ?, ?)", testUserID, "PETR4", "ACAO", 100)
	db.Exec("INSERT INTO investimentos_nacionais (user_id, ticker, tipo, quantidade) VALUES (?, ?, ?, ?)", testUserID, "MXRF11", "FII", 50) // FII para teste de exclusão
	db.Exec("INSERT INTO investimentos_internacionais (user_id, ticker, descricao, quantidade, moeda) VALUES (?, ?, ?, ?, ?)", testUserID, "VOO", "ETF S&P 500", 10.5, "USD")
}

// teardownInvestimentosTestDB fecha a conexão com o banco de dados de teste.
func teardownInvestimentosTestDB() {
	database.CloseDB()
}

// createInvestimentosTestRouter cria um roteador Gin para os testes de investimentos.
func createInvestimentosTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	authorized := r.Group("/")
	authorized.Use(mockAuthMiddleware())
	{
		// Adiciona todas as rotas de investimento que queremos testar
		authorized.POST("/investimentos/nacional", AddAtivoNacional)
		authorized.POST("/investimentos/nacional/:ticker", UpdateAtivoNacional)
		authorized.DELETE("/investimentos/nacional/:ticker", DeleteAtivoNacional)
		authorized.POST("/investimentos/internacional", AddAtivoInternacional)
		authorized.POST("/investimentos/internacional/:ticker", UpdateAtivoInternacional)
		authorized.DELETE("/investimentos/internacional/:ticker", DeleteAtivoInternacional)
		authorized.GET("/api/investimentos/precos", GetPrecosInvestimentosAPI)
	}
	return r
}

// performInvestimentosJSONRequest é um helper para simular requisições HTTP com corpo JSON.
func performInvestimentosJSONRequest(r http.Handler, method, path string, payload interface{}) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Testes para Ativos Nacionais ---

func TestAddAtivoNacional_Success(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	payload := AddNacionalPayload{
		Ticker:     "ITSA4",
		Tipo:       "ACAO",
		Quantidade: 200,
	}
	w := performInvestimentosJSONRequest(router, "POST", "/investimentos/nacional", payload)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAtivoNacional_Success(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	payload := UpdatePayload{Quantidade: 150}
	w := performInvestimentosJSONRequest(router, "POST", "/investimentos/nacional/PETR4", payload)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	// Verifica se o valor foi realmente atualizado no banco
	var novaQuantidade int
	db := database.GetDB()
	err := db.QueryRow("SELECT quantidade FROM investimentos_nacionais WHERE user_id = ? AND ticker = ?", testUserID, "PETR4").Scan(&novaQuantidade)
	if err != nil {
		t.Fatalf("Erro ao verificar a atualização no banco: %v", err)
	}
	if novaQuantidade != 150 {
		t.Errorf("Esperado quantidade 150, mas obteve %d", novaQuantidade)
	}
}

func TestDeleteAtivoNacional_Success(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	req, _ := http.NewRequest("DELETE", "/investimentos/nacional/PETR4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
}

// --- NOVOS TESTES PARA FIIs ---

func TestAddFII_Success(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	payload := AddNacionalPayload{
		Ticker:     "VGHF11",
		Tipo:       "FII",
		Quantidade: 150,
	}
	w := performInvestimentosJSONRequest(router, "POST", "/investimentos/nacional", payload)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 ao adicionar FII, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFII_Success(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	// Deleta o FII "MXRF11" que foi inserido no setup
	req, _ := http.NewRequest("DELETE", "/investimentos/nacional/MXRF11", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200 ao deletar FII, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
}


// --- Testes para Ativos Internacionais ---

func TestAddAtivoInternacional_Success(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	payload := AddInternacionalPayload{
		Ticker:     "AAPL",
		Descricao:  "Ações da Apple",
		Quantidade: 5.5,
	}
	w := performInvestimentosJSONRequest(router, "POST", "/investimentos/internacional", payload)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAtivoInternacional_Success(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	payload := UpdatePayload{Quantidade: 12.75}
	w := performInvestimentosJSONRequest(router, "POST", "/investimentos/internacional/VOO", payload)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAtivoInternacional_NotFound(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	// Tenta deletar um ativo que não existe
	req, _ := http.NewRequest("DELETE", "/investimentos/internacional/TSLA", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Esperado status 404, mas obteve %d.", w.Code)
	}
}

// --- Teste para a API de Preços Assíncrona ---
// Nota: Este teste não verifica o scraping, apenas se a API responde corretamente.
func TestGetPrecosInvestimentosAPI_Success(t *testing.T) {
	setupInvestimentosTestDB(t)
	defer teardownInvestimentosTestDB()
	router := createInvestimentosTestRouter()

	req, _ := http.NewRequest("GET", "/api/investimentos/precos", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d.", w.Code)
	}

	// Verifica se a resposta é um JSON válido
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("A resposta da API não é um JSON válido: %v", err)
	}

	// Verifica se as chaves principais existem na resposta
	keys := []string{"acoes", "fiis", "internacionais", "cotacaoDolar"}
	for _, key := range keys {
		if _, ok := response[key]; !ok {
			t.Errorf("A chave '%s' não foi encontrada na resposta da API de preços.", key)
		}
	}
}
