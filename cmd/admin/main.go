package main

import (
	"flag"
	"log"
	"minhas_economias/database"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Constantes compartilhadas pelo pacote main (admin)
const (
	tableName    = "movimentacoes"
	csvDelimiter = ';'
)

// MovimentacaoAux é usado pelo backup.go
type MovimentacaoAux struct {
	DataOcorrencia string
	Descricao      string
	Valor          float64
	Categoria      string
	Conta          string
	Consolidado    bool
}

func main() {
	// --- Definição das Flags ---
	importMovimentacoes := flag.Bool("import", false, "Importar dados de movimentações.")
	exportMovimentacoes := flag.Bool("export", false, "Exportar dados de movimentações.")
	importNacionais := flag.Bool("import-nacionais", false, "Importar investimentos nacionais.")
	importInternacionais := flag.Bool("import-internacionais", false, "Importar investimentos internacionais.")
	
	// Flags de Usuário e Configuração
	createUser := flag.Bool("create-user", false, "Criar um novo usuário.")
	initSchema := flag.Bool("init-db", false, "Criar tabelas do banco de dados.")
	
	// Parâmetros
	userIdParam := flag.Int64("user-id", 0, "ID do usuário (obrigatório para import/export).")
	userEmail := flag.String("email", "", "E-mail para criação de usuário.")
	userPass := flag.String("password", "", "Senha para criação de usuário.")
	userAdmin := flag.Bool("admin", false, "Define se o usuário criado é admin.")
	outputPathParam := flag.String("output-path", "backup/extrato_exportado.csv", "Caminho para exportação.")

	flag.Parse()

	// Conexão com o Banco
	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("Erro ao inicializar DB: %v", err)
	}
	db := database.GetDB()
	defer database.CloseDB()

	// 1. Inicialização de Schema
	if *initSchema {
		createTables(db)
		// Se for apenas init-db, não precisamos sair, podemos continuar se houver outras flags
	}

	// 2. Criação de Usuário
	if *createUser {
		if *userEmail == "" || *userPass == "" {
			log.Fatal("Para criar usuário, as flags -email e -password são obrigatórias.")
		}
		createNewUser(db, *userEmail, *userPass, *userAdmin, *userIdParam)
		return
	}

	// 3. Operações de Importação/Exportação
	hasDataOp := *importMovimentacoes || *exportMovimentacoes || *importNacionais || *importInternacionais

	if hasDataOp {
		if *userIdParam == 0 {
			log.Fatal("ERRO: A flag -user-id é obrigatória para operações de dados.")
		}
		if *userIdParam == 1 {
			log.Println("AVISO: Executando operações de dados no ID 1 (Admin).")
		}

		if *importNacionais {
			runImportInvestimentosNacionais(db, *userIdParam)
		}
		// AQUI ESTAVA O PROBLEMA RELATADO: Chamada corrigida via import.go
		if *importInternacionais {
			runImportInvestimentosInternacionais(db, *userIdParam)
		}
		if *importMovimentacoes {
			runImportMovimentacoes(db, *userIdParam)
		}
		if *exportMovimentacoes {
			runExport(db, *outputPathParam, *userIdParam)
		}
	} else if !*initSchema && !*createUser {
		flag.PrintDefaults()
	}
}