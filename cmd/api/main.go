package main

import (
	"log"
	"minhas_economias/auth"
	"minhas_economias/database"
	"minhas_economias/handlers"
	"minhas_economias/investimentos"
	"minhas_economias/gemini"
	"minhas_economias/middleware"

	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus/promhttp" // <-- IMPORTANTE
)

func createMyRender() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	pages, err := filepath.Glob("templates/*.html")
	if err != nil {
		panic(err.Error())
	}
	layout := "templates/_layout.html"
	for _, page := range pages {
		pageName := filepath.Base(page)
		if pageName == "_layout.html" || pageName == "login.html" || pageName == "register.html" {
			continue
		}
		r.AddFromFiles(pageName, layout, page)
	}
	r.AddFromFiles("login.html", "templates/login.html")
	r.AddFromFiles("register.html", "templates/register.html")
	return r
}

func main() {
	errEnv := godotenv.Load()
	if errEnv != nil {
		// Logamos apenas como aviso, pois em produção (Docker/K8s) o arquivo pode não existir.
		log.Println("Aviso: Arquivo .env não encontrado, usando variáveis de ambiente do sistema.")
	} else {
		log.Println("Arquivo .env carregado com sucesso.")
	}

	
	if os.Getenv("SESSION_KEY") == "" {
		log.Fatal("A variável de ambiente SESSION_KEY não foi definida.")
	}

	auth.InitSessionStore()
	
	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	defer database.CloseDB()

	if err := gemini.InitClient(); err != nil {
        log.Printf("AVISO: Não foi possível inicializar o cliente do Gemini AI. A funcionalidade de análise estará indisponível. Erro: %v", err)
    }

	r := gin.Default()
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.AuditLogger())
	r.HTMLRender = createMyRender()
	r.Static("/static", "./static")

	r.GET("/healthz", handlers.LivenessProbe)
	r.GET("/readyz", handlers.ReadinessProbe)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/login", auth.GetLoginPage)
	r.POST("/login", auth.PostLogin)
	r.GET("/register", auth.GetRegisterPage)
	r.POST("/register", auth.PostRegister)

	authorized := r.Group("/")
	authorized.Use(auth.AuthRequired())
	{
		authorized.GET("/", handlers.GetIndexPage)
		authorized.GET("/transacoes", handlers.GetTransacoesPage)
		authorized.GET("/relatorio", handlers.GetRelatorio)
		authorized.GET("/sobre", handlers.GetSobrePage)
		authorized.GET("/configuracoes", handlers.GetConfiguracoesPage)
		authorized.GET("/investimentos", investimentos.GetInvestimentosPage)
		authorized.POST("/logout", auth.PostLogout)

		authorized.GET("/analise", handlers.GetAnalisePage)
		authorized.POST("/api/analise/chat", handlers.PostAnaliseChat)

		// API
		authorized.GET("/api/movimentacoes", handlers.GetTransacoesPage)
		authorized.POST("/api/user/settings", handlers.UpdateUserSettings)
		authorized.POST("/api/user/profile", handlers.UpdateUserProfile)
		authorized.POST("/api/user/password", handlers.ChangePassword)
		authorized.GET("/api/investimentos/precos", investimentos.GetPrecosInvestimentosAPI)
		authorized.GET("/api/saldos", handlers.GetSaldosAPI) // <-- NOVA ROTA

		// Movimentações
		authorized.POST("/movimentacoes", handlers.AddMovimentacao)
		authorized.POST("/movimentacoes/transferencia", handlers.AddTransferencia) // <-- NOVA ROTA
		authorized.DELETE("/movimentacoes/:id", handlers.DeleteMovimentacao)
		authorized.POST("/movimentacoes/update/:id", handlers.UpdateMovimentacao)
		authorized.GET("/relatorio/transactions", handlers.GetTransactionsByCategory)
		authorized.POST("/relatorio/pdf", handlers.DownloadRelatorioPDF)
		authorized.GET("/export/csv", handlers.ExportTransactionsCSV)

		// Investimentos
		authorized.POST("/investimentos/nacional", investimentos.AddAtivoNacional)
		authorized.POST("/investimentos/nacional/:ticker", investimentos.UpdateAtivoNacional)
		authorized.DELETE("/investimentos/nacional/:ticker", investimentos.DeleteAtivoNacional)
		authorized.POST("/investimentos/internacional", investimentos.AddAtivoInternacional)
		authorized.POST("/investimentos/internacional/:ticker", investimentos.UpdateAtivoInternacional)
		authorized.DELETE("/investimentos/internacional/:ticker", investimentos.DeleteAtivoInternacional)
	}

	log.Println("Servidor Gin iniciado na porta :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar o servidor Gin: %v", err)
	}
}