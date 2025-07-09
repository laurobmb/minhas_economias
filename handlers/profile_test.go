package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"minhas_economias/auth"
	"minhas_economias/database"
	"minhas_economias/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// --- Funções de Setup (Adaptadas de movimentacoes_test.go) ---
// Nota: Em um projeto maior, essas funções de setup poderiam ser movidas para um pacote de teste compartilhado.

// setupProfileTestDB limpa e configura o banco de dados especificamente para os testes de perfil.
func setupProfileTestDB(t *testing.T) {
	_, err := database.InitDB()
	if err != nil {
		t.Fatalf("Falha ao inicializar o banco de dados de teste: %v", err)
	}
	db := database.GetDB()

	// Limpa as tabelas na ordem correta para evitar erros de chave estrangeira
	tables := []string{"movimentacoes", "contas", "user_profiles", "users"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		}
	}

	// Recria as tabelas necessárias para os testes de perfil
	createUsersSQL := `
	CREATE TABLE users (
		id BIGINT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		dark_mode_enabled BOOLEAN DEFAULT FALSE
	);`
	createUserProfilesSQL := `
	CREATE TABLE user_profiles (
		user_id BIGINT PRIMARY KEY,
		date_of_birth DATE,
		gender TEXT,
		marital_status TEXT,
		children_count INTEGER,
		country TEXT,
		state TEXT,
		city TEXT,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	if _, err := db.Exec(createUsersSQL); err != nil {
		t.Fatalf("Falha ao criar a tabela de teste 'users': %v", err)
	}
	if _, err := db.Exec(createUserProfilesSQL); err != nil {
		t.Fatalf("Falha ao criar a tabela de teste 'user_profiles': %v", err)
	}

	// Insere o usuário de teste
	hashedPassword, err := auth.HashPassword("senha_antiga_123")
	if err != nil {
		t.Fatalf("Falha ao gerar hash da senha de teste: %v", err)
	}
	insertUserSQL := database.Rebind("INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)")
	if _, err := db.Exec(insertUserSQL, testUserID, "test@user.com", hashedPassword); err != nil {
		t.Fatalf("Falha ao inserir usuário de teste: %v", err)
	}
}

// createProfileTestRouter cria um roteador Gin com as rotas de perfil e o mock de autenticação.
func createProfileTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// As rotas de perfil são APIs, não precisam de renderização de HTML
	authorized := r.Group("/")
	authorized.Use(mockAuthMiddleware())
	{
		authorized.POST("/api/user/profile", UpdateUserProfile)
		authorized.POST("/api/user/password", ChangePassword)
	}
	return r
}

// performJSONRequest é um helper para simular requisições HTTP com corpo JSON.
func performJSONRequest(r http.Handler, method, path string, payload interface{}) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Testes para o Perfil do Usuário ---

func TestUpdateAndGetUserProfile_Success(t *testing.T) {
	setupProfileTestDB(t)
	defer teardownTestDB()
	router := createProfileTestRouter()

	// 1. Atualiza o perfil com novos dados
	profileData := models.UserProfile{
		DateOfBirth:   "1990-05-15",
		Gender:        "Masculino",
		MaritalStatus: "Casado(a)",
		ChildrenCount: 2,
		Country:       "Brasil",
		State:         "São Paulo",
		City:          "São Paulo",
	}

	w := performJSONRequest(router, "POST", "/api/user/profile", profileData)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	// 2. Verifica se os dados foram salvos corretamente no banco
	savedProfile, err := GetUserProfileByUserID(testUserID)
	if err != nil {
		t.Fatalf("Erro ao buscar o perfil salvo: %v", err)
	}

	if savedProfile.DateOfBirth != profileData.DateOfBirth {
		t.Errorf("Esperado DateOfBirth '%s', mas obteve '%s'", profileData.DateOfBirth, savedProfile.DateOfBirth)
	}
	if savedProfile.City != profileData.City {
		t.Errorf("Esperado City '%s', mas obteve '%s'", profileData.City, savedProfile.City)
	}
	if savedProfile.ChildrenCount != profileData.ChildrenCount {
		t.Errorf("Esperado ChildrenCount '%d', mas obteve '%d'", profileData.ChildrenCount, savedProfile.ChildrenCount)
	}
}

// --- Testes para Alteração de Senha ---

func TestChangePassword_Success(t *testing.T) {
	setupProfileTestDB(t)
	defer teardownTestDB()
	router := createProfileTestRouter()

	payload := ChangePasswordPayload{
		CurrentPassword:    "senha_antiga_123",
		NewPassword:        "nova_senha_segura_456",
		ConfirmNewPassword: "nova_senha_segura_456",
	}

	w := performJSONRequest(router, "POST", "/api/user/password", payload)

	if w.Code != http.StatusOK {
		t.Errorf("Esperado status 200, mas obteve %d. Corpo: %s", w.Code, w.Body.String())
	}

	// Verifica se a nova senha funciona
	user, err := auth.GetUserByEmail("test@user.com")
	if err != nil {
		t.Fatalf("Não foi possível buscar o usuário após a alteração de senha: %v", err)
	}

	// Compara o hash da nova senha
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("nova_senha_segura_456"))
	if err != nil {
		t.Errorf("A nova senha não corresponde ao hash salvo no banco de dados.")
	}

	// Garante que a senha antiga não funciona mais
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("senha_antiga_123"))
	if err == nil {
		t.Errorf("A senha antiga ainda funciona, mas não deveria.")
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	setupProfileTestDB(t)
	defer teardownTestDB()
	router := createProfileTestRouter()

	payload := ChangePasswordPayload{
		CurrentPassword:    "senha_errada", // Senha atual incorreta
		NewPassword:        "nova_senha_456",
		ConfirmNewPassword: "nova_senha_456",
	}

	w := performJSONRequest(router, "POST", "/api/user/password", payload)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Esperado status 401 Unauthorized, mas obteve %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	expectedError := "A senha atual está incorreta."
	if response["error"] != expectedError {
		t.Errorf("Esperado erro '%s', mas obteve '%s'", expectedError, response["error"])
	}
}

func TestChangePassword_MismatchNewPassword(t *testing.T) {
	setupProfileTestDB(t)
	defer teardownTestDB()
	router := createProfileTestRouter()

	payload := ChangePasswordPayload{
		CurrentPassword:    "senha_antiga_123",
		NewPassword:        "nova_senha_456",
		ConfirmNewPassword: "confirmacao_diferente_789", // Confirmação não corresponde
	}

	w := performJSONRequest(router, "POST", "/api/user/password", payload)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400 Bad Request, mas obteve %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	expectedError := "A nova senha e a confirmação não correspondem."
	if response["error"] != expectedError {
		t.Errorf("Esperado erro '%s', mas obteve '%s'", expectedError, response["error"])
	}
}

func TestChangePassword_ShortNewPassword(t *testing.T) {
	setupProfileTestDB(t)
	defer teardownTestDB()
	router := createProfileTestRouter()

	payload := ChangePasswordPayload{
		CurrentPassword:    "senha_antiga_123",
		NewPassword:        "123", // Senha muito curta
		ConfirmNewPassword: "123",
	}

	// CORRIGIDO: Removido o 'e' extra do nome da função
	w := performJSONRequest(router, "POST", "/api/user/password", payload)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperado status 400 Bad Request, mas obteve %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	expectedError := "Todos os campos são obrigatórios e a nova senha deve ter no mínimo 6 caracteres."
	if response["error"] != expectedError {
		t.Errorf("Esperado erro '%s', mas obteve '%s'", expectedError, response["error"])
	}
}
