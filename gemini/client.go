package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var genaiClient *genai.GenerativeModel

// DateExtractionResponse é a estrutura para o JSON que esperamos do Gemini.
type DateExtractionResponse struct {
	StartDate string `json:"start_date"` // Formato YYYY-MM-DD
	EndDate   string `json:"end_date"`   // Formato YYYY-MM-DD
}

// InitClient inicializa o cliente da API do Gemini de forma robusta.
func InitClient() error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("a variável de ambiente GEMINI_API_KEY não foi definida")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("erro ao criar cliente do GenAI: %w", err)
	}

	desiredModelName := os.Getenv("GEMINI_MODEL")
	if desiredModelName == "" {
		desiredModelName = "gemini-2.5-flash"
		log.Println("Variável de ambiente GEMINI_MODEL não definida. Usando modelo padrão: gemini-pro")
	}

	availableModels := make(map[string]bool)
	var availableModelNamesForLog []string

	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("erro ao listar modelos disponíveis: %w", err)
		}
		modelShortName := strings.TrimPrefix(m.Name, "models/")
		availableModels[modelShortName] = true
		availableModelNamesForLog = append(availableModelNamesForLog, modelShortName)
	}

	if _, ok := availableModels[desiredModelName]; !ok {
		errorMsg := fmt.Sprintf(
			"o modelo solicitado '%s' não foi encontrado ou não está disponível para sua chave de API.\n"+
				"============================== MODELOS DISPONÍVEIS ==============================\n"+
				"%s\n"+
				"===================================================================================\n"+
				"Verifique se a API 'Generative Language API' está ativada no seu projeto Google Cloud.",
			desiredModelName, strings.Join(availableModelNamesForLog, "\n"),
		)
		return fmt.Errorf(errorMsg)
	}

	genaiClient = client.GenerativeModel(desiredModelName)
	log.Printf("Cliente do Google Gemini AI inicializado com o modelo: %s", desiredModelName)

	return nil
}

// ExtractDatesFromQuestion usa a IA para interpretar um pedido de período em linguagem natural.
func ExtractDatesFromQuestion(userQuestion string) (startDate, endDate string, err error) {
	if genaiClient == nil {
		if err := InitClient(); err != nil {
			return "", "", err
		}
	}

	ctx := context.Background()
	today := time.Now().Format("2006-01-02")

	prompt := fmt.Sprintf(`
		Analise a pergunta do usuário e extraia um período de tempo. A data de hoje é %s.
		Responda APENAS com um objeto JSON contendo "start_date" e "end_date" no formato "YYYY-MM-DD".
		Se nenhum período for mencionado, retorne um JSON com valores vazios: {"start_date": "", "end_date": ""}.

		Exemplos:
		- Pergunta: "analise os últimos 3 meses" -> Calcule os últimos 3 meses completos a partir de hoje.
		- Pergunta: "como foram minhas finanças em agosto?" -> Retorne o primeiro e último dia de agosto do ano corrente.
		- Pergunta: "me dê um resumo" -> Retorne valores vazios.
		- Pergunta: "qual foi meu maior gasto?" -> Retorne valores vazios.

		Pergunta do Usuário: "%s"
	`, today, userQuestion)

	resp, err := genaiClient.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", "", fmt.Errorf("erro na pré-análise do Gemini para extrair datas: %w", err)
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			jsonString := strings.TrimSpace(string(text))
			jsonString = strings.TrimPrefix(jsonString, "```json")
			jsonString = strings.TrimSuffix(jsonString, "```")

			var dateResp DateExtractionResponse
			if err := json.Unmarshal([]byte(jsonString), &dateResp); err != nil {
				log.Printf("AVISO: Gemini não retornou um JSON de data válido. Resposta: %s. Erro: %v", jsonString, err)
				return "", "", nil
			}
			return dateResp.StartDate, dateResp.EndDate, nil
		}
	}

	return "", "", nil
}

// GenerateAnalysis envia os dados financeiros e a pergunta do usuário para o Gemini e retorna a análise.
func GenerateAnalysis(financialData string, userQuestion string) (string, error) {
    if genaiClient == nil {
        if err := InitClient(); err != nil {
            return "", err
        }
    }

    ctx := context.Background()
    prompt := fmt.Sprintf(`
		**Persona:** Você é um analista financeiro pessoal chamado "Minhas Economias AI". Sua comunicação deve ser clara, objetiva, construtiva e encorajadora.

		**Tarefa:** Analise os seguintes dados de transações financeiras. Os valores negativos são despesas e os positivos são receitas.  
		1. Forneça um **resumo mensal em formato de tabela Markdown**, incluindo receitas totais, despesas totais e saldo.  
		2. Identifique **tendências entre os meses** (ex.: aumento/diminuição de despesas, sazonalidade).  
		3. Destaque os **principais pontos de atenção** (ex.: maiores despesas, concentração de gastos).  
		4. Sugira **melhorias práticas** e realistas para a gestão financeira.  
		5. Sempre que possível, calcule métricas como: percentual de poupança, proporção das maiores despesas e receita.

		**Restrições:**  
		1. Baseie sua análise **APENAS** nos dados fornecidos abaixo. Não invente informações.  
		2. Se não houver dados, informe que a análise não pode ser feita.  
		3. Responda à pergunta do usuário ("%s") dentro do contexto da análise financeira. Se a pergunta fugir do tema, responda educadamente que você só pode discutir os dados financeiros apresentados.  
		4. Use Markdown para formatar a resposta, com títulos, listas e tabelas para melhor legibilidade.  

		**Dados Financeiros:**  
        ---
        %s
        ---
    `, userQuestion, financialData)

    resp, err := genaiClient.GenerateContent(ctx, genai.Text(prompt))
    if err != nil {
        return "", fmt.Errorf("erro ao gerar conteúdo do Gemini: %w", err)
    }

    if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
        if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
            return string(text), nil
        }
    }

    return "", fmt.Errorf("resposta da API do Gemini vazia ou em formato inesperado")
}