package handlers

import (
	// "database/sql" // <-- REMOVIDO DAQUI
	"fmt"
	"log"
	"minhas_economias/database"
	"minhas_economias/gemini"
	"minhas_economias/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GetAnalisePage (sem alterações)
func GetAnalisePage(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	chatHistory, err := GetChatHistoryByUserID(user.ID)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao carregar o histórico da análise.", err)
		return
	}
	c.HTML(http.StatusOK, "analise.html", gin.H{
		"Titulo":      "Análise com IA",
		"User":        user,
		"ChatHistory": chatHistory,
	})
}

// PostAnaliseChat (sem alterações na lógica, apenas garantindo que esteja correto)
func PostAnaliseChat(c *gin.Context) {
	userID := c.MustGet("userID").(int64)

	var requestBody struct {
		Question string `json:"question"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requisição inválida"})
		return
	}

	userMessage := models.ChatMessage{UserID: userID, Role: "user", Content: requestBody.Question}
	if err := saveChatMessage(userMessage); err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao salvar sua pergunta.", err)
		return
	}

	startDate, endDate, err := gemini.ExtractDatesFromQuestion(requestBody.Question)
	if err != nil {
		log.Printf("AVISO: Falha ao extrair datas da pergunta: %v", err)
	}

	financialData, err := fetchFinancialDataForPeriod(userID, startDate, endDate)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao buscar dados financeiros.", err)
		return
	}

	analysis, err := gemini.GenerateAnalysis(financialData, requestBody.Question)
	if err != nil {
		renderErrorPage(c, http.StatusInternalServerError, "Erro ao comunicar com a IA.", err)
		return
	}

	aiMessage := models.ChatMessage{UserID: userID, Role: "ai", Content: analysis}
	if err := saveChatMessage(aiMessage); err != nil {
		c.JSON(http.StatusOK, gin.H{"analysis": analysis, "warning": "Não foi possível salvar esta resposta no histórico."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analysis": analysis})
}

// GetChatHistoryByUserID (sem alterações)
func GetChatHistoryByUserID(userID int64) ([]models.ChatMessage, error) {
	query := database.Rebind(`
		SELECT id, user_id, role, content, created_at 
		FROM chat_history 
		WHERE user_id = ? 
		ORDER BY created_at ASC 
		LIMIT 50`)

	rows, err := database.GetDB().Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.ChatMessage
	for rows.Next() {
		var msg models.ChatMessage
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, msg)
	}
	return history, nil
}

// saveChatMessage (sem alterações)
func saveChatMessage(msg models.ChatMessage) error {
	query := database.Rebind(`
		INSERT INTO chat_history (user_id, role, content) 
		VALUES (?, ?, ?)`)

	_, err := database.GetDB().Exec(query, msg.UserID, msg.Role, msg.Content)
	return err
}

// fetchFinancialDataForPeriod (sem alterações)
func fetchFinancialDataForPeriod(userID int64, startDate, endDate string) (string, error) {
	if startDate == "" || endDate == "" {
		now := time.Now()
		endOfLastMonth := time.Date(now.Year(), now.Month(), 0, 23, 59, 59, 0, now.Location())
		startOfLastMonth := time.Date(endOfLastMonth.Year(), endOfLastMonth.Month(), 1, 0, 0, 0, 0, now.Location())
		startDate = startOfLastMonth.Format("2006-01-02")
		endDate = endOfLastMonth.Format("2006-01-02")
		fmt.Printf("Nenhum período informado. Usando período padrão (último mês): %s a %s\n", startDate, endDate)
	}

	var args []interface{}
	args = append(args, userID)

	query := fmt.Sprintf(`
		SELECT data_ocorrencia, descricao, valor, categoria, conta
		FROM %s
		WHERE user_id = ?`, database.TableName)

	query += " AND data_ocorrencia BETWEEN ? AND ?"
	args = append(args, startDate, endDate)

	query += " ORDER BY data_ocorrencia ASC"

	reboundQuery := database.Rebind(query)
	rows, err := database.GetDB().Query(reboundQuery, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var dataBuilder strings.Builder
	dataBuilder.WriteString(fmt.Sprintf("Período da Análise: %s a %s\n", startDate, endDate))
	dataBuilder.WriteString("Data;Descricao;Valor;Categoria;Conta\n")

	rowCount := 0
	for rows.Next() {
		var mov models.Movimentacao
		var rawDate interface{}
		if err := rows.Scan(&rawDate, &mov.Descricao, &mov.Valor, &mov.Categoria, &mov.Conta); err != nil {
			continue
		}
		mov.DataOcorrencia = scanDate(rawDate)
		line := fmt.Sprintf("%s;%s;%.2f;%s;%s\n", mov.DataOcorrencia, mov.Descricao, mov.Valor, mov.Categoria, mov.Conta)
		dataBuilder.WriteString(line)
		rowCount++
	}

	if rowCount == 0 {
		return fmt.Sprintf("Nenhuma transação encontrada para o período de %s a %s.", startDate, endDate), nil
	}

	return dataBuilder.String(), nil
}