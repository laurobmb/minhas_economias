package main

import (
	"log"
	"minhas_economias/auth"
	"minhas_economias/database"
	"minhas_economias/handlers"
	"minhas_economias/investimentos"

	"os"
	"path/filepath"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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
	if os.Getenv("SESSION_KEY") == "" {
		log.Fatal("A variável de ambiente SESSION_KEY não foi definida.")
	}
	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	defer database.CloseDB()

	r := gin.Default()
	r.HTMLRender = createMyRender()
	r.Static("/static", "./static")

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

		// API
		authorized.GET("/api/movimentacoes", handlers.GetTransacoesPage)
		authorized.POST("/api/user/settings", handlers.UpdateUserSettings)
		authorized.POST("/api/user/profile", handlers.UpdateUserProfile)
		authorized.POST("/api/user/password", handlers.ChangePassword)

		// Movimentações
		authorized.POST("/movimentacoes", handlers.AddMovimentacao)
		authorized.DELETE("/movimentacoes/:id", handlers.DeleteMovimentacao)
		authorized.POST("/movimentacoes/update/:id", handlers.UpdateMovimentacao)
		authorized.GET("/relatorio/transactions", handlers.GetTransactionsByCategory)
		authorized.POST("/relatorio/pdf", handlers.DownloadRelatorioPDF)
		authorized.GET("/export/csv", handlers.ExportTransactionsCSV)

		// --- ROTAS DE INVESTIMENTOS (ATUALIZADAS) ---
		authorized.POST("/investimentos/nacional", investimentos.AddAtivoNacional) // NOVA ROTA
		authorized.POST("/investimentos/nacional/:ticker", investimentos.UpdateAtivoNacional)
		authorized.DELETE("/investimentos/nacional/:ticker", investimentos.DeleteAtivoNacional)
		authorized.POST("/investimentos/internacional", investimentos.AddAtivoInternacional) // NOVA ROTA
		authorized.POST("/investimentos/internacional/:ticker", investimentos.UpdateAtivoInternacional)
		authorized.DELETE("/investimentos/internacional/:ticker", investimentos.DeleteAtivoInternacional)
	}

	log.Println("Servidor Gin iniciado na porta :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar o servidor Gin: %v", err)
	}
}
