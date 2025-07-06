package main

import (
	"html/template"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"minhas_economias/database"
	"minhas_economias/handlers"
)

func main() {
	var err error
	_, err = database.InitDB("extratos.db")
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	defer database.CloseDB()

	r := gin.Default()

	// A configuração do template foi simplificada, sem Funcs() pois o jsonify foi removido.
	// O template do Go lida com a conversão para JSON automaticamente dentro de tags <script>.
	r.SetHTMLTemplate(template.Must(template.ParseGlob("templates/*")))
	log.Println("Templates HTML carregados de 'templates/'.")

	// Garante que os diretórios existem (bom para primeiro run)
	if _, err = os.Stat("templates"); os.IsNotExist(err) {
		err = os.Mkdir("templates", 0755)
		if err != nil {
			log.Fatalf("Erro ao criar diretório 'templates': %v", err)
		}
	} else if err != nil {
		log.Fatalf("Erro ao verificar diretório 'templates': %v", err)
	}
	log.Println("Diretório 'templates' verificado/criado com sucesso.")

	if _, err = os.Stat("static"); os.IsNotExist(err) {
		err = os.Mkdir("static", 0755)
		if err != nil {
			log.Fatalf("Erro ao criar diretório 'static': %v", err)
		}
	} else if err != nil {
		log.Fatalf("Erro ao verificar diretório 'static': %v", err)
	}
	log.Println("Diretório 'static' verificado/criado com sucesso.")

	if _, err = os.Stat("static/css"); os.IsNotExist(err) {
		err = os.Mkdir("static/css", 0755)
		if err != nil {
			log.Fatalf("Erro ao criar diretório 'static/css': %v", err)
		}
	} else if err != nil {
		log.Fatalf("Erro ao verificar diretório 'static/css': %v", err)
	}
	log.Println("Diretório 'static/css' verificado/criado com sucesso.")

	log.Println("Esperando que 'templates/index.html' e 'static/css/style.css' existam.")

	// Configura o serviço de arquivos estáticos
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