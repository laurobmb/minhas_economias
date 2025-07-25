package investimentos

import (
	"log"
	"minhas_economias/database"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Estrutura do Cache (sem alterações) ---
type CacheItem struct {
	Data      interface{}
	Timestamp time.Time
}

var (
	cache      = make(map[string]CacheItem)
	cacheMutex = &sync.RWMutex{}
	cacheTTL   = 15 * time.Minute
)

func getFromCache(key string) (interface{}, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	item, found := cache[key]
	if !found || time.Since(item.Timestamp) > cacheTTL {
		return nil, false
	}
	return item.Data, true
}

func setToCache(key string, data interface{}) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	cache[key] = CacheItem{
		Data:      data,
		Timestamp: time.Now(),
	}
}

// --- NOVAS FUNÇÕES DE GERENCIAMENTO DE CACHE ---

// ClearNacionalCache remove os dados de Ações e FIIs do cache.
func ClearNacionalCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	delete(cache, "market_data_acoes")
	delete(cache, "market_data_fiis")
	log.Println("[Cache] Cache de ativos nacionais (Ações e FIIs) limpo.")
}

// ClearInternacionalCache remove os dados de ativos internacionais do cache.
func ClearInternacionalCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	delete(cache, "mapa_precos_internacionais")
	log.Println("[Cache] Cache de ativos internacionais limpo.")
}


// --- Funções de Serviço (sem alterações na lógica principal) ---

func GetAcoesNacionais(userID int64) ([]AcaoNacional, error) {
	log.Println("[InvestimentosService] Iniciando busca de Ações Nacionais...")
	dadosMercado, err := getDadosMercadoAcoes()
	if err != nil {
		return nil, err
	}
	db := database.GetDB()
	query := database.Rebind("SELECT ticker, tipo, quantidade FROM investimentos_nacionais WHERE user_id = ? AND (tipo = 'ACAO' OR tipo = 'Acao') ORDER BY ticker")
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var acoes []AcaoNacional
	for rows.Next() {
		var acao AcaoNacional
		if err := rows.Scan(&acao.Ticker, &acao.Tipo, &acao.Quantidade); err != nil {
			log.Printf("AVISO: Falha ao escanear linha de ação do banco: %v", err)
			continue
		}
		tickerLimpo := strings.TrimSpace(acao.Ticker)
		if dados, ok := dadosMercado[tickerLimpo]; ok {
			acao.Cotacao = ParsePtBrFloat(dados[1])
			acao.PVP = ParsePtBrFloat(dados[3])
			acao.DivYield = ParsePtBrFloat(dados[5]) / 100.0
			acao.DivYieldPercent = acao.DivYield * 100.0
			acao.ValorTotal = acao.Cotacao * float64(acao.Quantidade)
		} else {
			log.Printf("AVISO (Ações): Ticker '%s' da sua carteira não foi encontrado.", tickerLimpo)
		}
		acoes = append(acoes, acao)
	}
	var wg sync.WaitGroup
	for i := range acoes {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ticker := acoes[index].Ticker
			lpa, vpa, err := BuscarLPAeVPA(ticker)
			if err == nil {
				acoes[index].ValorGraham = CalcularValorGraham(lpa, vpa)
				if acoes[index].ValorGraham > 0 && acoes[index].Cotacao > 0 && acoes[index].ValorGraham > acoes[index].Cotacao {
					acoes[index].IsGrahamAdvantageous = true
				}
			} else {
				log.Printf("AVISO (Graham): Falha ao buscar LPA/VPA para %s: %v", ticker, err)
			}
		}(i)
	}
	wg.Wait()
	return acoes, nil
}

func GetFIIsNacionais(userID int64) ([]FundoImobiliario, error) {
	log.Println("[InvestimentosService] Iniciando busca de FIIs...")
	dadosMercado, err := getDadosMercadoFIIs()
	if err != nil {
		return nil, err
	}
	db := database.GetDB()
	query := database.Rebind("SELECT ticker, tipo, quantidade FROM investimentos_nacionais WHERE user_id = ? AND tipo = 'FII' ORDER BY ticker")
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fiis []FundoImobiliario
	for rows.Next() {
		var fii FundoImobiliario
		if err := rows.Scan(&fii.Ticker, &fii.Tipo, &fii.Quantidade); err != nil {
			log.Printf("AVISO: Falha ao escanear linha de FII do banco: %v", err)
			continue
		}
		tickerLimpo := strings.TrimSpace(fii.Ticker)
		if dados, ok := dadosMercado[tickerLimpo]; ok {
			fii.Segmento = dados[1]
			fii.Cotacao = ParsePtBrFloat(dados[2])
			fii.DivYield = ParsePtBrFloat(dados[4]) / 100.0
			fii.DivYieldPercent = fii.DivYield * 100.0
			fii.PVP = ParsePtBrFloat(dados[5])
			fii.Vacancia = ParsePtBrFloat(dados[12])
			fii.NumImoveis, _ = strconv.Atoi(dados[8])
			fii.ValorTotal = fii.Cotacao * float64(fii.Quantidade)
		} else {
			log.Printf("AVISO (FIIs): Ticker '%s' da sua carteira não foi encontrado.", tickerLimpo)
		}
		fiis = append(fiis, fii)
	}
	return fiis, nil
}

func GetAtivosInternacionais(userID int64) ([]AtivoInternacional, float64, error) {
	log.Println("[InvestimentosService] Iniciando busca de Ativos Internacionais...")
	cotacaoDolar, err := getCotacaoDolar()
	if err != nil {
		log.Printf("AVISO: Falha ao buscar cotação do dólar. Usando valor 0. Erro: %v", err)
		cotacaoDolar = 0
	}
	db := database.GetDB()
	query := database.Rebind("SELECT ticker, descricao, quantidade, moeda FROM investimentos_internacionais WHERE user_id = ? ORDER BY ticker")
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var ativos []AtivoInternacional
	carteiraParaBusca := make(map[string]string)
	for rows.Next() {
		var ativo AtivoInternacional
		if err := rows.Scan(&ativo.Ticker, &ativo.Descricao, &ativo.Quantidade, &ativo.Moeda); err != nil {
			log.Printf("AVISO: Falha ao escanear linha de ativo internacional: %v", err)
			continue
		}
		ativos = append(ativos, ativo)
		carteiraParaBusca[ativo.Ticker] = ativo.Moeda
	}
	if len(carteiraParaBusca) == 0 {
		return ativos, cotacaoDolar, nil
	}
	mapaDePrecos, err := getMapaDePrecosInternacionais(carteiraParaBusca)
	if err != nil {
		log.Printf("ERRO ao buscar mapa de preços internacionais: %v. Os preços não serão preenchidos.", err)
	}
	for i := range ativos {
		tickerLimpo := strings.TrimSpace(ativos[i].Ticker)
		if precoUSD, ok := mapaDePrecos[tickerLimpo]; ok {
			ativos[i].PrecoUnitarioUSD = precoUSD
			ativos[i].ValorTotalUSD = ativos[i].PrecoUnitarioUSD * ativos[i].Quantidade
			ativos[i].ValorTotalBRL = ativos[i].ValorTotalUSD * cotacaoDolar
		} else {
			log.Printf("AVISO (Internacional): Ticker '%s' da sua carteira não foi encontrado nos dados de mercado do Yahoo Finance.", tickerLimpo)
		}
	}
	return ativos, cotacaoDolar, nil
}

func getDadosMercadoAcoes() (map[string][]string, error) {
	const cacheKey = "market_data_acoes"
	if data, found := getFromCache(cacheKey); found {
		log.Println("[Cache] Usando dados de AÇÕES do cache.")
		return data.(map[string][]string), nil
	}
	data, err := RasparDadosFundamentus("https://www.fundamentus.com.br/resultado.php", "acoes")
	if err != nil {
		return nil, err
	}
	setToCache(cacheKey, data)
	return data, nil
}

func getDadosMercadoFIIs() (map[string][]string, error) {
	const cacheKey = "market_data_fiis"
	if data, found := getFromCache(cacheKey); found {
		log.Println("[Cache] Usando dados de FIIs do cache.")
		return data.(map[string][]string), nil
	}
	data, err := RasparDadosFundamentus("https://www.fundamentus.com.br/fii_resultado.php", "fii")
	if err != nil {
		return nil, err
	}
	setToCache(cacheKey, data)
	return data, nil
}

func getCotacaoDolar() (float64, error) {
	const cacheKey = "cotacao_dolar"
	if data, found := getFromCache(cacheKey); found {
		log.Println("[Cache] Usando cotação do dólar do cache.")
		return data.(float64), nil
	}
	data, err := BuscarCotacaoDolarBRL()
	if err != nil {
		return 0, err
	}
	setToCache(cacheKey, data)
	return data, nil
}

func getMapaDePrecosInternacionais(carteira map[string]string) (map[string]float64, error) {
	const cacheKey = "mapa_precos_internacionais"
	if data, found := getFromCache(cacheKey); found {
		log.Println("[Cache] Usando mapa de preços internacionais do cache.")
		return data.(map[string]float64), nil
	}
	data, err := BuscarMuitosPrecosInternacionais(carteira)
	if err != nil {
		return nil, err
	}
	setToCache(cacheKey, data)
	return data, nil
}
