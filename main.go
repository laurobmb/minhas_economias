package main

import (
	"log"
	"minhas_economias/auth"
	"minhas_economias/database"
	"minhas_economias/handlers"
	"os"
	"path/filepath"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// createMyRender carrega os templates de forma que a herança funcione corretamente.
// Ela agora diferencia páginas que usam o layout de páginas standalone.
func createMyRender() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	
	// Lista de páginas que NÃO usam o layout principal.
	standalonePages := map[string]bool{
		"login.html":    true,
		"register.html": true,
	}

	// Carrega o layout principal
	layouts, err := filepath.Glob("templates/_layout.html")
	if err != nil {
		panic(err.Error())
	}

	// Carrega todas as páginas
	pages, err := filepath.Glob("templates/*.html")
	if err != nil {
		panic(err.Error())
	}

	for _, page := range pages {
		pageName := filepath.Base(page)

		// Se a página for standalone, carrega somente ela.
		if _, ok := standalonePages[pageName]; ok {
			log.Printf("Carregando template standalone: %s", pageName)
			r.AddFromFiles(pageName, page)
			continue
		}
		
		// Se a página usa o layout, carrega o layout junto com ela.
		if pageName != "_layout.html" {
			log.Printf("Carregando template com layout: %s", pageName)
			r.AddFromFiles(pageName, append(layouts, page)...)
		}
	}
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

	// Use o renderizador customizado
	r.HTMLRender = createMyRender()
	log.Println("Templates HTML carregados com o renderizador customizado.")

	r.Static("/static", "./static")
	log.Println("Servindo arquivos estáticos de '/static' para './static'.")

	// --- ROTAS (sem alterações) ---
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
		authorized.GET("/configuracoes", handlers.GetConfiguracoesPage) // <-- NOVA ROTA

		authorized.POST("/logout", auth.PostLogout)
		authorized.GET("/api/movimentacoes", handlers.GetTransacoesPage)
		authorized.POST("/api/user/settings", handlers.UpdateUserSettings)
		authorized.POST("/movimentacoes", handlers.AddMovimentacao)
		authorized.DELETE("/movimentacoes/:id", handlers.DeleteMovimentacao)
		authorized.POST("/movimentacoes/update/:id", handlers.UpdateMovimentacao)
		authorized.GET("/relatorio/transactions", handlers.GetTransactionsByCategory)
		authorized.POST("/relatorio/pdf", handlers.DownloadRelatorioPDF)
	}

	log.Println("Servidor Gin iniciado na porta :8080")
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("Erro ao iniciar o servidor Gin: %v", err)
	}
}