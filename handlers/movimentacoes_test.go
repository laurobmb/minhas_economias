package handlers

import (
	"database/sql"
	"fmt"
	"minhas_economias/auth"
	"minhas_economias/database"
	"minhas_economias/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// testUserID é o ID do usuário que usaremos para todos os testes.
const testUserID int64 = 999

// mockAuthMiddleware simula um usuário logado.
func mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		testUser := &models.User{
			ID:              testUserID,
			Email:           "test@user.com",
			IsAdmin:         false,
			DarkModeEnabled: false,
		}
		c.Set("userID", testUserID)
		c.Set("user", testUser)
		c.Next()
	}
}

// setupTestDB configura um banco de dados de teste limpo.
func setupTestDB(t *testing.T) {
	_, err := database.InitDB()
	if err != nil {
		t.Fatalf("Falha ao inicializar o banco de dados de teste: %v", err)
	}
	db := database.GetDB()

	tables := []string{"movimentacoes", "contas", "users"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		}
	}

	createUsersSQL := `
	CREATE TABLE users (
			id BIGINT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			is_admin BOOLEAN DEFAULT FALSE,
			dark_mode_enabled BOOLEAN DEFAULT FALSE
	);`
	createMovimentacoesSQL_sqlite := fmt.Sprintf(`
	CREATE TABLE %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id BIGINT NOT NULL,
			data_ocorrencia DATE NOT NULL,
			descricao TEXT,
			valor NUMERIC(10, 2),
			categoria TEXT,
			conta TEXT,
			consolidado BOOLEAN DEFAULT FALSE,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`, database.TableName)
	createMovimentacoesSQL_postgres := fmt.Sprintf(`
	CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			data_ocorrencia DATE NOT NULL,
			descricao TEXT,
			valor NUMERIC(10, 2),
			categoria TEXT,
			conta TEXT,
			consolidado BOOLEAN DEFAULT FALSE,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`, database.TableName)
	createContasSQL := `
	CREATE TABLE contas (
			user_id BIGINT NOT NULL,
			nome TEXT NOT NULL,
			saldo_inicial NUMERIC(10, 2) NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, nome),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	if _, err := db.Exec(createUsersSQL); err != nil {
		t.Fatalf("Falha ao criar a tabela de teste 'users': %v", err)
	}
	if database.DriverName == "postgres" {
		if _, err := db.Exec(createMovimentacoesSQL_postgres); err != nil {
			t.Fatalf("Falha ao criar a tabela de teste 'movimentacoes' para postgres: %v", err)
		}
	} else {
		if _, err := db.Exec(createMovimentacoesSQL_sqlite); err != nil {
			t.Fatalf("Falha ao criar a tabela de teste 'movimentacoes' para sqlite: %v", err)
		}
	}
	if _, err := db.Exec(createContasSQL); err != nil {
		t.Fatalf("Falha ao criar a tabela de teste 'contas': %v", err)
	}

	hashedPassword, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("Falha ao gerar hash da senha de teste: %v", err)
	}
	insertUserSQL := database.Rebind("INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)")
	if _, err := db.Exec(insertUserSQL, testUserID, "test@user.com", hashedPassword); err != nil {
		t.Fatalf("Falha ao inserir usuário de teste: %v", err)
	}

	insertMovimentacoesSQL := database.Rebind(fmt.Sprintf(`
INSERT INTO %s (id, user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES
(?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?);`, database.TableName))

	if database.DriverName == "postgres" {
		db.Exec(fmt.Sprintf("SELECT setval('%s_id_seq', 2, true);", database.TableName))
	}

	_, err = db.Exec(insertMovimentacoesSQL,
		1, testUserID, "2025-01-10", "Aluguel", -1500.00, "Moradia", "Banco A", true,
		2, testUserID, "2025-01-15", "Salario", 3000.00, "Renda", "Banco A", true)
	if err != nil {
		t.Fatalf("Falha ao inserir dados de teste em 'movimentacoes': %v", err)
	}
}

// teardownTestDB fecha a conexão com o banco de dados de teste.
func teardownTestDB() {
	database.CloseDB()
}

// createTestRouter cria um roteador Gin para testes com o middleware de autenticação mockado.
func createTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	renderer := multitemplate.NewRenderer()
	layouts, err := filepath.Glob("../templates/_layout.html")
	if err != nil || len(layouts) == 0 {
		panic("Erro: não foi possível encontrar o arquivo de layout _layout.html. " + err.Error())
	}

	pages, err := filepath.Glob("../templates/*.html")
	if err != nil {
		panic("Erro: não foi possível encontrar os templates de página. " + err.Error())
	}

	for _, page := range pages {
		pageName := filepath.Base(page)
		if pageName != "_layout.html" && pageName != "login.html" && pageName != "register.html" {
			renderer.AddFromFiles(pageName, append(layouts, page)...)
		}
	}
	r.HTMLRender = renderer

	authorized := r.Group("/")
	authorized.Use(mockAuthMiddleware())
	{
		authorized.GET("/", GetIndexPage)
		authorized.GET("/transacoes", GetTransacoesPage)
		authorized.GET("/relatorio", GetRelatorio)
		authorized.GET("/configuracoes", GetConfiguracoesPage)
		authorized.POST("/movimentacoes", AddMovimentacao)
		authorized.POST("/movimentacoes/update/:id", UpdateMovimentacao)
		authorized.DELETE("/movimentacoes/:id", DeleteMovimentacao)
		authorized.POST("/api/user/settings", UpdateUserSettings)
		authorized.POST("/movimentacoes/transferencia", AddTransferencia) // <-- ROTA ADICIONADA
	}

	return r
}

// performRequest é um helper para simular requisições HTTP.
func performRequest(r http.Handler, method, path string, body url.Values, headers http.Header) *httptest.ResponseRecorder {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, path, strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	if err != nil {
		panic(fmt.Sprintf("Falha ao criar a requisição: %v", err))
	}

	if headers != nil {
		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
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

// =============================================================================
// >> NOVOS TESTES PARA TRANSFERÊNCIA <<
// =============================================================================

func TestAddTransferencia_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	formData := url.Values{}
	formData.Set("data_ocorrencia", "2025-07-01")
	formData.Set("descricao", "Pagamento do Cartão")
	formData.Set("valor", "250,75") // Usa vírgula como no formulário
	formData.Set("conta_origem", "Banco A")
	formData.Set("conta_destino", "Corretora B")

	w := performRequest(router, "POST", "/movimentacoes/transferencia", formData, nil)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302 (redirecionamento), mas obteve %d.", w.Code)
	}

	// Verificação no banco de dados
	db := database.GetDB()
	var count int
	query := database.Rebind("SELECT COUNT(*) FROM movimentacoes WHERE categoria = ? AND user_id = ?")
	err := db.QueryRow(query, "Transferência", testUserID).Scan(&count)
	if err != nil {
		t.Fatalf("Erro ao verificar a inserção no banco de dados: %v", err)
	}
	if count != 2 {
		t.Errorf("Esperado encontrar 2 movimentações de transferência, mas encontrou %d", count)
	}

	// Verifica débito da origem
	var debitValue float64
	// CORREÇÃO: A query agora é específica para a transação de transferência.
	debitQuery := database.Rebind("SELECT valor FROM movimentacoes WHERE conta = ? AND user_id = ? AND categoria = ? AND valor = ?")
	err = db.QueryRow(debitQuery, "Banco A", testUserID, "Transferência", -250.75).Scan(&debitValue)
	if err != nil {
		t.Fatalf("Erro ao buscar a movimentação de débito da transferência: %v", err)
	}
	if debitValue != -250.75 {
		t.Errorf("Esperado valor de débito -250.75, mas obteve %.2f", debitValue)
	}

	// Verifica crédito no destino
	var creditValue float64
	// CORREÇÃO: A query agora é específica para a transação de transferência.
	creditQuery := database.Rebind("SELECT valor FROM movimentacoes WHERE conta = ? AND user_id = ? AND categoria = ? AND valor = ?")
	err = db.QueryRow(creditQuery, "Corretora B", testUserID, "Transferência", 250.75).Scan(&creditValue)
	if err != nil {
		t.Fatalf("Erro ao buscar a movimentação de crédito da transferência: %v", err)
	}
	if creditValue != 250.75 {
		t.Errorf("Esperado valor de crédito 250.75, mas obteve %.2f", creditValue)
	}
}

func TestAddTransferencia_Validation(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	testCases := []struct {
		name          string
		formData      url.Values
		expectedError string
	}{
		{
			name: "Contas Iguais",
			formData: url.Values{
				"data_ocorrencia": {"2025-07-02"},
				"descricao":       {"Teste Contas Iguais"},
				"valor":           {"100.00"},
				"conta_origem":    {"Banco A"},
				"conta_destino":   {"Banco A"},
			},
			expectedError: "A conta de origem e destino não podem ser a mesma.",
		},
		{
			name: "Valor Zero",
			formData: url.Values{
				"data_ocorrencia": {"2025-07-02"},
				"descricao":       {"Teste Valor Zero"},
				"valor":           {"0"},
				"conta_origem":    {"Banco A"},
				"conta_destino":   {"Banco B"},
			},
			expectedError: "O valor da transferência deve ser um número positivo.",
		},
		{
			name: "Valor Negativo",
			formData: url.Values{
				"data_ocorrencia": {"2025-07-02"},
				"descricao":       {"Teste Valor Negativo"},
				"valor":           {"-50.00"},
				"conta_origem":    {"Banco A"},
				"conta_destino":   {"Banco B"},
			},
			expectedError: "O valor da transferência deve ser um número positivo.",
		},
		{
			name: "Falta Conta de Origem",
			formData: url.Values{
				"data_ocorrencia": {"2025-07-02"},
				"descricao":       {"Teste Falta Origem"},
				"valor":           {"50.00"},
				"conta_destino":   {"Banco B"},
			},
			expectedError: "Todos os campos da transferência são obrigatórios.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := performRequest(router, "POST", "/movimentacoes/transferencia", tc.formData, nil)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Esperado status 400 Bad Request, mas obteve %d.", w.Code)
			}
			if !strings.Contains(w.Body.String(), tc.expectedError) {
				t.Errorf("Esperava a mensagem de erro '%s', mas obteve: '%s'", tc.expectedError, w.Body.String())
			}
		})
	}
}

// --- Funções de Teste Existentes (sem alterações) ---

func TestGetIndexPage_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	w := performRequest(router, "GET", "/", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d.", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Painel de Saldos") {
		t.Errorf("Corpo da resposta não contém o título esperado para a página inicial.")
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

	w := performRequest(router, "POST", "/movimentacoes", formData, nil)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302 (redirecionamento), mas obteve %d.", w.Code)
	}

	db := database.GetDB()
	var count int
	query := database.Rebind("SELECT COUNT(*) FROM movimentacoes WHERE descricao = ? AND user_id = ?")
	err := db.QueryRow(query, "Nova Despesa", testUserID).Scan(&count)
	if err != nil {
		t.Fatalf("Erro ao verificar a inserção no banco de dados: %v", err)
	}
	if count != 1 {
		t.Errorf("Esperado encontrar 1 movimentação inserida, mas encontrou %d", count)
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

	w := performRequest(router, "POST", "/movimentacoes/update/2", formData, nil)

	if w.Code != http.StatusFound {
		t.Errorf("Esperado status 302 (redirecionamento), mas obteve %d.", w.Code)
	}

	db := database.GetDB()
	var desc string
	query := database.Rebind("SELECT descricao FROM movimentacoes WHERE id = ? AND user_id = ?")
	err := db.QueryRow(query, 2, testUserID).Scan(&desc)
	if err != nil {
		t.Fatalf("Erro ao verificar a atualização no banco de dados: %v", err)
	}
	if desc != "Salario Atualizado" {
		t.Errorf("Esperava descrição 'Salario Atualizado', mas obteve '%s'", desc)
	}
}

func TestDeleteMovimentacao_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	w := performRequest(router, "DELETE", "/movimentacoes/1", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d.", w.Code)
	}

	db := database.GetDB()
	var count int
	query := database.Rebind("SELECT COUNT(*) FROM movimentacoes WHERE id = ? AND user_id = ?")
	err := db.QueryRow(query, 1, testUserID).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		if err == sql.ErrNoRows {
			count = 0
		} else {
			t.Fatalf("Erro ao verificar a deleção no banco de dados: %v", err)
		}
	}
	if count != 0 {
		t.Errorf("Esperado que a movimentação fosse deletada (contagem 0), mas a contagem é %d", count)
	}
}

func TestUpdateMovimentacao_Validation(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()
	router := createTestRouter()

	formData := url.Values{}
	formData.Set("data_ocorrencia", "2025-01-10")
	formData.Set("descricao", strings.Repeat("a", 61)) // Descrição muito longa
	formData.Set("valor", "-1500.00")
	formData.Set("conta", "Banco A")

	w := performRequest(router, "POST", "/movimentacoes/update/1", formData, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400, mas obteve %d.", w.Code)
	}
	if !strings.Contains(w.Body.String(), "A descrição não pode ter mais de 60 caracteres.") {
		t.Errorf("Mensagem de erro para descrição longa não encontrada no corpo: %s", w.Body.String())
	}
}