// main.go
package main

import (
	"html/template"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	// O nome do módulo que você usou em 'go mod init'
	"minhas_economias/database" // Importe o pacote database
	"minhas_economias/handlers" // Importe o pacote handlers
	"encoding/json"              // Mantenha para a função jsonify
)

// jsonify é uma função auxiliar para converter um valor para JSON.
// Mantida aqui por ser uma função auxiliar para templates, não específica de um handler.
func jsonify(data interface{}) (template.JS, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return template.JS(string(b)), nil
}

func main() {
	var err error

	// Inicializa o banco de dados
	// Certifique-se de que "extratos.db" existe ou será criado pelo seu script de importação
	_, err = database.InitDB("extratos.db") 
	if err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	defer database.CloseDB() // Garante que a conexão seja fechada

	r := gin.Default()

	// Criar um novo FuncMap e adicionar a função jsonify
	funcMap := template.FuncMap{
		"jsonify": jsonify,
	}
	// SetHTMLTemplate agora usa o FuncMap
	r.SetHTMLTemplate(template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*")))
	log.Println("Templates HTML carregados de 'templates/' com funções customizadas.")

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

	// Rotas HTTP
	r.GET("/", handlers.GetMovimentacoes)
	r.GET("/api/movimentacoes", handlers.GetMovimentacoes)
	r.POST("/movimentacoes", handlers.AddMovimentacao)
	r.DELETE("/movimentacoes/:id", handlers.DeleteMovimentacao)
	r.POST("/movimentacoes/update/:id", handlers.UpdateMovimentacao) 

	log.Println("Servidor Gin iniciado na porta :8080")
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("Erro ao iniciar o servidor Gin: %v", err)
	}
}
