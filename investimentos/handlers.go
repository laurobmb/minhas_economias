package investimentos

import (
	"log"
	"minhas_economias/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetInvestimentosPage busca os dados do BD, enriquece e renderiza a página.
func GetInvestimentosPage(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	// Busca Ações Nacionais
	acoes, errAcoes := GetAcoesNacionais(user.ID)
	if errAcoes != nil {
		log.Printf("ERRO ao carregar ações nacionais: %v", errAcoes)
	}

	// Busca Fundos Imobiliários Nacionais
	fiis, errFIIs := GetFIIsNacionais(user.ID)
	if errFIIs != nil {
		log.Printf("ERRO ao carregar FIIs: %v", errFIIs)
	}

	// ======================== CORREÇÃO AQUI ========================
	// Chama a função com o argumento correto (apenas user.ID)
	// e recebe os 3 valores de retorno que a função agora provê.
	internacionais, cotacaoDolar, errInt := GetAtivosInternacionais(user.ID)
	// ===============================================================
	
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
