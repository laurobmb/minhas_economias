package investimentos

import (
	"log"
	"minhas_economias/database"
	"minhas_economias/models"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// GetInvestimentosPage agora carrega apenas os dados básicos do banco de dados.
// A página será renderizada instantaneamente, e o JavaScript buscará os preços.
func GetInvestimentosPage(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	user := c.MustGet("user").(*models.User)
	db := database.GetDB()

	var acoes []AcaoNacional
	var fiis []FundoImobiliario
	var internacionais []AtivoInternacional

	// Busca apenas os ativos nacionais (Ações e FIIs) do banco
	queryNacionais := database.Rebind("SELECT ticker, tipo, quantidade FROM investimentos_nacionais WHERE user_id = ? ORDER BY ticker")
	rowsNacionais, errNac := db.Query(queryNacionais, userID)
	if errNac != nil {
		log.Printf("ERRO ao carregar ativos nacionais do BD: %v", errNac)
	} else {
		defer rowsNacionais.Close()
		for rowsNacionais.Next() {
			var ativo AcaoNacional // Usamos AcaoNacional como base
			if err := rowsNacionais.Scan(&ativo.Ticker, &ativo.Tipo, &ativo.Quantidade); err == nil {
				if ativo.Tipo == "ACAO" {
					acoes = append(acoes, ativo)
				} else if ativo.Tipo == "FII" {
					fiis = append(fiis, FundoImobiliario{Ticker: ativo.Ticker, Tipo: ativo.Tipo, Quantidade: ativo.Quantidade})
				}
			}
		}
	}

	// Busca apenas os ativos internacionais do banco
	queryInt := database.Rebind("SELECT ticker, descricao, quantidade, moeda FROM investimentos_internacionais WHERE user_id = ? ORDER BY ticker")
	rowsInt, errInt := db.Query(queryInt, userID)
	if errInt != nil {
		log.Printf("ERRO ao carregar ativos internacionais do BD: %v", errInt)
	} else {
		defer rowsInt.Close()
		for rowsInt.Next() {
			var ativo AtivoInternacional
			if err := rowsInt.Scan(&ativo.Ticker, &ativo.Descricao, &ativo.Quantidade, &ativo.Moeda); err == nil {
				internacionais = append(internacionais, ativo)
			}
		}
	}

	c.HTML(http.StatusOK, "investimentos.html", gin.H{
		"Titulo":         "Meus Investimentos",
		"Acoes":          acoes,
		"FIIs":           fiis,
		"Internacionais": internacionais,
		"User":           user,
	})
}

// GetPrecosInvestimentosAPI é o novo handler que faz o trabalho pesado em background.
// Ele busca os preços e retorna tudo em um único JSON.
func GetPrecosInvestimentosAPI(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var wg sync.WaitGroup
	var errAcoes, errFIIs, errInt error

	var acoes []AcaoNacional
	var fiis []FundoImobiliario
	var internacionais []AtivoInternacional
	var cotacaoDolar float64

	wg.Add(3) // Temos 3 tarefas concorrentes

	// Tarefa 1: Buscar Ações Nacionais
	go func() {
		defer wg.Done()
		acoes, errAcoes = GetAcoesNacionais(userID)
	}()

	// Tarefa 2: Buscar FIIs
	go func() {
		defer wg.Done()
		fiis, errFIIs = GetFIIsNacionais(userID)
	}()

	// Tarefa 3: Buscar Ativos Internacionais
	go func() {
		defer wg.Done()
		internacionais, cotacaoDolar, errInt = GetAtivosInternacionais(userID)
	}()

	wg.Wait() // Espera todas as tarefas terminarem

	// Log de erros, se houver
	if errAcoes != nil { log.Printf("ERRO na API de preços (Ações): %v", errAcoes) }
	if errFIIs != nil { log.Printf("ERRO na API de preços (FIIs): %v", errFIIs) }
	if errInt != nil { log.Printf("ERRO na API de preços (Internacionais): %v", errInt) }

	c.JSON(http.StatusOK, gin.H{
		"acoes":          acoes,
		"fiis":           fiis,
		"internacionais": internacionais,
		"cotacaoDolar":   cotacaoDolar,
	})
}


// --- Payloads para requisições (sem alterações) ---
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

// --- Handlers de Adição, Edição e Exclusão (sem alterações) ---
func AddAtivoNacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var payload AddNacionalPayload
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()}); return }
	payload.Ticker = strings.ToUpper(strings.TrimSpace(payload.Ticker))
	if payload.Ticker == "" || payload.Quantidade <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker e quantidade são obrigatórios e a quantidade deve ser positiva."}); return }
	db := database.GetDB()
	var query string
	if database.DriverName == "postgres" {
		query = `INSERT INTO investimentos_nacionais (user_id, ticker, tipo, quantidade) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, ticker) DO UPDATE SET quantidade = investimentos_nacionais.quantidade + EXCLUDED.quantidade;`
	} else {
		query = `INSERT OR REPLACE INTO investimentos_nacionais (user_id, ticker, tipo, quantidade) VALUES (?, ?, ?, ?);`
	}
	_, err := db.Exec(query, userID, payload.Ticker, payload.Tipo, payload.Quantidade)
	if err != nil { log.Printf("Erro ao adicionar/atualizar ativo nacional: %v", err); c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar o ativo no banco de dados."}); return }
	ClearNacionalCache()
	c.JSON(http.StatusOK, gin.H{"message": "Ativo adicionado/atualizado com sucesso!"})
}
func AddAtivoInternacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var payload AddInternacionalPayload
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()}); return }
	payload.Ticker = strings.ToUpper(strings.TrimSpace(payload.Ticker))
	if payload.Ticker == "" || payload.Descricao == "" || payload.Quantidade <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "Todos os campos são obrigatórios e a quantidade deve ser positiva."}); return }
	db := database.GetDB()
	var query string
	if database.DriverName == "postgres" {
		query = `INSERT INTO investimentos_internacionais (user_id, ticker, descricao, quantidade, moeda) VALUES ($1, $2, $3, $4, 'USD') ON CONFLICT (user_id, ticker) DO UPDATE SET quantidade = investimentos_internacionais.quantidade + EXCLUDED.quantidade, descricao = EXCLUDED.descricao;`
	} else {
		query = `INSERT OR REPLACE INTO investimentos_internacionais (user_id, ticker, descricao, quantidade, moeda) VALUES (?, ?, ?, ?, 'USD');`
	}
	_, err := db.Exec(query, userID, payload.Ticker, payload.Descricao, payload.Quantidade)
	if err != nil { log.Printf("Erro ao adicionar/atualizar ativo internacional: %v", err); c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar o ativo no banco de dados."}); return }
	ClearInternacionalCache()
	c.JSON(http.StatusOK, gin.H{"message": "Ativo adicionado/atualizado com sucesso!"})
}
func UpdateAtivoNacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	ticker := c.Param("ticker")
	var payload UpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()}); return }
	db := database.GetDB()
	query := database.Rebind("UPDATE investimentos_nacionais SET quantidade = ? WHERE user_id = ? AND ticker = ?")
	result, err := db.Exec(query, int(payload.Quantidade), userID, ticker)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar o ativo no banco de dados."}); return }
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Ativo não encontrado ou não pertence a este usuário."}); return }
	ClearNacionalCache()
	c.JSON(http.StatusOK, gin.H{"message": "Ativo atualizado com sucesso!"})
}
func DeleteAtivoNacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	ticker := c.Param("ticker")
	db := database.GetDB()
	query := database.Rebind("DELETE FROM investimentos_nacionais WHERE user_id = ? AND ticker = ?")
	result, err := db.Exec(query, userID, ticker)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir o ativo do banco de dados."}); return }
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Ativo não encontrado ou não pertence a este usuário."}); return }
	ClearNacionalCache()
	c.JSON(http.StatusOK, gin.H{"message": "Ativo excluído com sucesso!"})
}
func UpdateAtivoInternacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	ticker := c.Param("ticker")
	var payload UpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()}); return }
	db := database.GetDB()
	query := database.Rebind("UPDATE investimentos_internacionais SET quantidade = ? WHERE user_id = ? AND ticker = ?")
	result, err := db.Exec(query, payload.Quantidade, userID, ticker)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar o ativo no banco de dados."}); return }
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Ativo não encontrado ou não pertence a este usuário."}); return }
	ClearInternacionalCache()
	c.JSON(http.StatusOK, gin.H{"message": "Ativo atualizado com sucesso!"})
}
func DeleteAtivoInternacional(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	ticker := c.Param("ticker")
	db := database.GetDB()
	query := database.Rebind("DELETE FROM investimentos_internacionais WHERE user_id = ? AND ticker = ?")
	result, err := db.Exec(query, userID, ticker)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir o ativo do banco de dados."}); return }
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Ativo não encontrado ou não pertence a este usuário."}); return }
	ClearInternacionalCache()
	c.JSON(http.StatusOK, gin.H{"message": "Ativo excluído com sucesso!"})
}
