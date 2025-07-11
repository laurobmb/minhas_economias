package investimentos

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2" // CORREÇÃO AQUI
)

// =================================================================================
// >> INÍCIO DA SEÇÃO DE DADOS INTERNACIONAIS (ESTRATÉGIA FINAL: XPATH) <<
// =================================================================================

// buildYahooURL foi simplificada para confiar no ticker do banco de dados.
func buildYahooURL(ticker string) string {
	return fmt.Sprintf("https://finance.yahoo.com/quote/%s", ticker)
}

// BuscarMuitosPrecosInternacionais faz o scraping no Yahoo Finance para uma lista de tickers.
// Esta versão final usa um seletor de XPath direto para capturar o preço.
func BuscarMuitosPrecosInternacionais(carteira map[string]string) (map[string]float64, error) {
	log.Println("[MarketData] Iniciando busca de cotações internacionais (método XPath)...")
	resultados := make(map[string]float64)
	var mu sync.Mutex

	c := colly.NewCollector(colly.Async(true))
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5, RandomDelay: 1 * time.Second})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		r.Headers.Set("Cookie", "CONSENT=YES+cb.20210418-17-p0.v1.cs2; A3=d=AQABBOMR-GEC&S=AQAAAmG9LpA-sB79c_MDFw")
		log.Printf("[MarketData] Visitando: %s", r.URL.String())
	})

	// **NOVA ESTRATÉGIA**: Usando o seletor XPath fornecido por você.
	priceSelector := `//*[@id="nimbus-app"]/section/section/section/article/section[1]/div[2]/div[1]/section/div/section/div[1]/div[1]/span`

	// Trocamos para OnXML para usar o seletor XPath.
	c.OnXML(priceSelector, func(e *colly.XMLElement) {
		ticker := e.Request.Ctx.Get("ticker")

		mu.Lock()
		_, found := resultados[ticker]
		mu.Unlock()
		if found {
			return
		}

		// **CORREÇÃO FINAL**: Remove espaços em branco antes e depois do texto do preço.
		// Isso resolve o erro de "invalid syntax" no ParseFloat.
		precoStr := strings.TrimSpace(e.Text)
		precoStr = strings.ReplaceAll(precoStr, ",", "")

		preco, err := strconv.ParseFloat(precoStr, 64)
		if err != nil {
			log.Printf("!!! ERRO: Falha ao converter o texto do preço '%s' para o ticker '%s': %v", precoStr, ticker, err)
			return
		}

		if preco > 0 {
			mu.Lock()
			if _, found := resultados[ticker]; !found {
				log.Printf(">>> SUCESSO: Preço encontrado para %s: %.2f", ticker, preco)
				resultados[ticker] = preco
			}
			mu.Unlock()
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		ticker := r.Request.Ctx.Get("ticker")
		log.Printf("!!! ERRO no scraping para o ticker '%s' na URL %s: %v (Status: %d)", ticker, r.Request.URL, err, r.StatusCode)
	})

	for ticker := range carteira {
		url := buildYahooURL(ticker)
		ctx := colly.NewContext()
		ctx.Put("ticker", ticker)
		c.Request("GET", url, nil, ctx, nil)
	}

	c.Wait()

	log.Printf("[MarketData] Busca internacional finalizada. Encontradas cotações para %d de %d ativos.", len(resultados), len(carteira))
	return resultados, nil
}


// =================================================================================
// >> RESTANTE DO ARQUIVO (SEM ALTERAÇÕES) <<
// =================================================================================

// RasparDadosFundamentus agora é exportada (letra maiúscula)
func RasparDadosFundamentus(url string, tipoAtivo string) (map[string][]string, error) {
	log.Printf("[MarketData] Iniciando scraping da tabela de %s...", tipoAtivo)
	dadosMapa := make(map[string][]string)
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
		r.Headers.Set("Referer", "https://www.fundamentus.com.br")
	})

	var seletorTabela string
	if tipoAtivo == "fii" {
		seletorTabela = "#tabelaResultado > tbody > tr"
	} else {
		seletorTabela = "#resultado > tbody > tr"
	}
	c.OnHTML(seletorTabela, func(e *colly.HTMLElement) {
		cols := e.ChildTexts("td")
		if len(cols) > 1 {
			tickerLimpo := strings.TrimSpace(cols[0])
			dadosMapa[tickerLimpo] = cols
		}
	})
	err := c.Visit(url)
	if err != nil {
		return nil, fmt.Errorf("falha ao visitar o site de %s: %w", tipoAtivo, err)
	}
	c.Wait()
	log.Printf("[MarketData] Scraping de %s finalizado. %d ativos encontrados.", tipoAtivo, len(dadosMapa))
	return dadosMapa, nil
}

// BuscarLPAeVPA agora é exportada (letra maiúscula)
func BuscarLPAeVPA(ticker string) (float64, float64, error) {
	var lpa, vpa float64
	var scrapingErr error
	url := "https://statusinvest.com.br/acoes/" + strings.ToLower(ticker)
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	})
	lpaXPath := `//*[@id="indicators-section"]/div[2]/div/div[1]/div/div[11]/div/div/strong`
	vpaXPath := `//*[@id="indicators-section"]/div[2]/div/div[1]/div/div[9]/div/div/strong`
	c.OnXML(lpaXPath, func(e *colly.XMLElement) {
		textValue := strings.ReplaceAll(e.Text, ",", ".")
		lpa, _ = strconv.ParseFloat(textValue, 64)
	})
	c.OnXML(vpaXPath, func(e *colly.XMLElement) {
		textValue := strings.ReplaceAll(e.Text, ",", ".")
		vpa, _ = strconv.ParseFloat(textValue, 64)
	})
	c.OnError(func(r *colly.Response, e error) {
		scrapingErr = fmt.Errorf("erro no scraping para o ticker %s: %v", ticker, e)
		log.Printf("Erro no scraping: %v", scrapingErr)
	})
	err := c.Visit(url)
	if err != nil {
		return 0, 0, err
	}
	time.Sleep(1 * time.Second)
	c.Wait()
	if scrapingErr != nil {
		return 0, 0, scrapingErr
	}
	if lpa == 0 && vpa == 0 {
		log.Printf("AVISO: LPA ou VPA não encontrados para %s com XPath. O cálculo de Graham será 0.", ticker)
		return 0, 0, nil
	}
	log.Printf("Dados encontrados para %s -> LPA: %.2f, VPA: %.2f", ticker, lpa, vpa)
	return lpa, vpa, nil
}

// BuscarCotacaoDolarBRL agora é exportada (letra maiúscula)
func BuscarCotacaoDolarBRL() (float64, error) {
	resp, err := http.Get("https://api.frankfurter.app/latest?from=USD&to=BRL")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	taxa, ok := result.Rates["BRL"]
	if !ok {
		return 0, fmt.Errorf("taxa BRL não encontrada na resposta da API")
	}
	return taxa, nil
}

// --- Funções Auxiliares ---
// ParsePtBrFloat agora é exportada (letra maiúscula)
func ParsePtBrFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "%", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

// CalcularValorGraham agora é exportada (letra maiúscula)
func CalcularValorGraham(lpa, vpa float64) float64 {
	if lpa <= 0 || vpa <= 0 {
		return 0.0
	}
	valor := 22.5 * lpa * vpa
	return math.Sqrt(valor)
}
