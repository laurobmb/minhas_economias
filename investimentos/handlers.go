package investimentos

import (
	"log"
	"minhas_economias/database"
	"minhas_economias/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetInvestimentosPage busca os dados do BD, enriquece e renderiza a página.
func GetInvestimentosPage(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	acoes, errAcoes := GetAcoesNacionais(user.ID)
	if errAcoes != nil {
		log.Printf("ERRO ao carregar ações nacionais: %v", errAcoes)
	}
	fiis, errFIIs := GetFIIsNacionais(user.ID)
	if errFIIs != nil {
		log.Printf("ERRO ao carregar FIIs: %v", errFIIs)
	}
	internacionais, cotacaoDolar, errInt := GetAtivosInternacionais(user.ID)
	if errInt != nil {
		log.Printf("ERRO ao carregar investimentos internacionais: %v", errInt)
	}
	c.HTML(http.StatusOK, "investimentos.html", gin.H{
		"Titulo":         "Meus Investimentos",
		"Acoes":          acoes,
		"FIIs":           fiis,
		"Internacionais": internacionais,
		"CotacaoDolar":   cotacaoDolar,
		"User":           user,
	})
}

// --- Payloads para requisições ---

type UpdatePayload struct {
	Quantidade float64 `json:"quantidade" binding:"required"`
}

type AddNacionalPayload struct {
	Ticker     string `json:"ticker" binding:"required"`
	Tipo       string `json:"tipo" binding:"required"`
	Quantidade int    `json:"quantidade" binding:"required"`
}

type AddInternacionalPayload struct {
	Ticker      string  `json:"ticker" binding:"required"`
	Descricao   string  `json:"descricao" binding:"required"`
	Quantidade  float64 `json:"quantidade" binding:"required"`
}


// --- Handlers de Adição ---

func AddAtivoNacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var payload AddNacionalPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}
	payload.Ticker = strings.ToUpper(strings.TrimSpace(payload.Ticker))
	if payload.Ticker == "" || payload.Quantidade <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker e quantidade são obrigatórios e a quantidade deve ser positiva."})
		return
	}
	db := database.GetDB()
	var query string
	if database.DriverName == "postgres" {
		query = `
            INSERT INTO investimentos_nacionais (user_id, ticker, tipo, quantidade)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (user_id, ticker) DO UPDATE
            SET quantidade = investimentos_nacionais.quantidade + EXCLUDED.quantidade;`
	} else {
		query = `INSERT OR REPLACE INTO investimentos_nacionais (user_id, ticker, tipo, quantidade) VALUES (?, ?, ?, ?);`
	}
	_, err := db.Exec(query, userID, payload.Ticker, payload.Tipo, payload.Quantidade)
	if err != nil {
		log.Printf("Erro ao adicionar/atualizar ativo nacional: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar o ativo no banco de dados."})
		return
	}
	ClearNacionalCache() // Limpa o cache
	c.JSON(http.StatusOK, gin.H{"message": "Ativo adicionado/atualizado com sucesso!"})
}

func AddAtivoInternacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var payload AddInternacionalPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}
	payload.Ticker = strings.ToUpper(strings.TrimSpace(payload.Ticker))
	if payload.Ticker == "" || payload.Descricao == "" || payload.Quantidade <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Todos os campos são obrigatórios e a quantidade deve ser positiva."})
		return
	}
	db := database.GetDB()
	var query string
	if database.DriverName == "postgres" {
		query = `
            INSERT INTO investimentos_internacionais (user_id, ticker, descricao, quantidade, moeda)
            VALUES ($1, $2, $3, $4, 'USD')
            ON CONFLICT (user_id, ticker) DO UPDATE
            SET quantidade = investimentos_internacionais.quantidade + EXCLUDED.quantidade,
                descricao = EXCLUDED.descricao;`
	} else {
		query = `INSERT OR REPLACE INTO investimentos_internacionais (user_id, ticker, descricao, quantidade, moeda) VALUES (?, ?, ?, ?, 'USD');`
	}
	_, err := db.Exec(query, userID, payload.Ticker, payload.Descricao, payload.Quantidade)
	if err != nil {
		log.Printf("Erro ao adicionar/atualizar ativo internacional: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar o ativo no banco de dados."})
		return
	}
	ClearInternacionalCache() // Limpa o cache
	c.JSON(http.StatusOK, gin.H{"message": "Ativo adicionado/atualizado com sucesso!"})
}


// --- Handlers de Edição e Exclusão (Existentes) ---

func UpdateAtivoNacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	ticker := c.Param("ticker")
	var payload UpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}
	db := database.GetDB()
	query := database.Rebind("UPDATE investimentos_nacionais SET quantidade = ? WHERE user_id = ? AND ticker = ?")
	result, err := db.Exec(query, int(payload.Quantidade), userID, ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar o ativo no banco de dados."})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ativo não encontrado ou não pertence a este usuário."})
		return
	}
	ClearNacionalCache() // Limpa o cache
	c.JSON(http.StatusOK, gin.H{"message": "Ativo atualizado com sucesso!"})
}

func DeleteAtivoNacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	ticker := c.Param("ticker")
	db := database.GetDB()
	query := database.Rebind("DELETE FROM investimentos_nacionais WHERE user_id = ? AND ticker = ?")
	result, err := db.Exec(query, userID, ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir o ativo do banco de dados."})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ativo não encontrado ou não pertence a este usuário."})
		return
	}
	ClearNacionalCache() // Limpa o cache
	c.JSON(http.StatusOK, gin.H{"message": "Ativo excluído com sucesso!"})
}

func UpdateAtivoInternacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	ticker := c.Param("ticker")
	var payload UpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}
	db := database.GetDB()
	query := database.Rebind("UPDATE investimentos_internacionais SET quantidade = ? WHERE user_id = ? AND ticker = ?")
	result, err := db.Exec(query, payload.Quantidade, userID, ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar o ativo no banco de dados."})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ativo não encontrado ou não pertence a este usuário."})
		return
	}
	ClearInternacionalCache() // Limpa o cache
	c.JSON(http.StatusOK, gin.H{"message": "Ativo atualizado com sucesso!"})
}

func DeleteAtivoInternacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	ticker := c.Param("ticker")
	db := database.GetDB()
	query := database.Rebind("DELETE FROM investimentos_internacionais WHERE user_id = ? AND ticker = ?")
	result, err := db.Exec(query, userID, ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir o ativo do banco de dados."})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ativo não encontrado ou não pertence a este usuário."})
		return
	}
	ClearInternacionalCache() // Limpa o cache
	c.JSON(http.StatusOK, gin.H{"message": "Ativo excluído com sucesso!"})
}
