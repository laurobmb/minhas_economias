package main

import (
	"html/template"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"           // Import do driver PostgreSQL
	_ "github.com/mattn/go-sqlite3" // Import do driver SQLite
	"minhas_economias/database"
	"minhas_economias/handlers"
)

func main() {
	// A inicialização do banco de dados agora lê a variável DB_TYPE para decidir qual banco usar.
	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	defer database.CloseDB()

	r := gin.Default()

	r.SetHTMLTemplate(template.Must(template.ParseGlob("templates/*")))
	log.Println("Templates HTML carregados de 'templates/'.")

	// Garante que os diretórios existem
	if _, err = os.Stat("templates"); os.IsNotExist(err) {
		os.Mkdir("templates", 0755)
	}
	if _, err = os.Stat("static"); os.IsNotExist(err) {
		os.Mkdir("static", 0755)
	}
	if _, err = os.Stat("static/css"); os.IsNotExist(err) {
		os.Mkdir("static/css", 0755)
	}

	r.Static("/static", "./static")
	log.Println("Servindo arquivos estáticos de '/static' para './static'.")

	// Rotas da aplicação
	r.GET("/", handlers.GetIndexPage)
	r.GET("/transacoes", handlers.GetTransacoesPage)
	r.GET("/relatorio", handlers.GetRelatorio)
	r.GET("/sobre", handlers.GetSobrePage)

	// Rotas para a API e ações de formulário
	r.GET("/api/movimentacoes", handlers.GetTransacoesPage)
	r.POST("/movimentacoes", handlers.AddMovimentacao)
	r.DELETE("/movimentacoes/:id", handlers.DeleteMovimentacao)
	r.POST("/movimentacoes/update/:id", handlers.UpdateMovimentacao)
	r.GET("/relatorio/transactions", handlers.GetTransactionsByCategory)
	r.POST("/relatorio/pdf", handlers.DownloadRelatorioPDF)

	log.Println("Servidor Gin iniciado na porta :8080")
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("Erro ao iniciar o servidor Gin: %v", err)
	}
}