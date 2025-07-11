package investimentos

// AcaoNacional representa uma Ação com dados do BD e dados dinâmicos.
type AcaoNacional struct {
	Ticker               string  `json:"ticker"`
	Tipo                 string  `json:"tipo"`
	Quantidade           int     `json:"quantidade"`
	Cotacao              float64 `json:"cotacao"`
	ValorTotal           float64 `json:"valor_total"`
	PVP                  float64 `json:"pvp"`
	DivYield             float64 `json:"div_yield"`
	ValorGraham          float64 `json:"valor_graham"`
	DivYieldPercent      float64 `json:"div_yield_percent"`
	IsGrahamAdvantageous bool    `json:"is_graham_advantageous"`
}

// FundoImobiliario representa um FII com dados do BD e dados dinâmicos.
type FundoImobiliario struct {
	Ticker       string  `json:"ticker"`
	Tipo         string  `json:"tipo"`
	Quantidade   int     `json:"quantidade"`
	Cotacao      float64 `json:"cotacao"`
	ValorTotal   float64 `json:"valor_total"`
	Segmento     string  `json:"segmento"`
	PVP          float64 `json:"pvp"`
	DivYield     float64 `json:"div_yield"`
	Vacancia     float64 `json:"vacancia"`
	NumImoveis   int     `json:"num_imoveis"`
	DivYieldPercent float64 `json:"div_yield_percent"`
}

// AtivoInternacional representa um ativo no exterior.
type AtivoInternacional struct {
	Ticker           string  `json:"ticker"`
	Descricao        string  `json:"descricao"`
	Quantidade       float64 `json:"quantidade"`
	Moeda            string  `json:"moeda"`
	PrecoUnitarioUSD float64 `json:"preco_unitario_usd"`
	ValorTotalUSD    float64 `json:"valor_total_usd"`
	ValorTotalBRL    float64 `json:"valor_total_brl"`
}
