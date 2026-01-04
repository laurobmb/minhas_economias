# ==========================================
# 🛠️ VARIÁVEIS DO PROJETO
# ==========================================
BINARY_NAME=minhas-economias
CLI_NAME=admin-cli
CONVERTER_NAME=xls-converter
BUILD_DIR=bin
VERSION=v1.1.0
CONTAINER_TOOL=podman

# Detecta o Sistema Operacional
ifeq ($(OS),Windows_NT)
    EXT=.exe
else
    EXT=
endif

# ==========================================
# 💻 DESENVOLVIMENTO LOCAL
# ==========================================

.PHONY: all build build-cli build-converter run test clean help up down clean-data logs setup-prod

all: clean build build-cli build-converter

# 1. Compila a API (Servidor Web)
build:
	@echo "🔨 Compilando API..."
	CGO_ENABLED=1 go build -o $(BINARY_NAME)$(EXT) ./cmd/api

# 2. Compila a CLI de Administração
build-cli:
	@echo "🔨 Compilando Admin CLI..."
	CGO_ENABLED=1 go build -o $(CLI_NAME)$(EXT) ./cmd/admin

# 3. Compila o Conversor XLS -> CSV
build-converter:
	@echo "🔨 Compilando Conversor XLS..."
	CGO_ENABLED=1 go build -o $(CONVERTER_NAME)$(EXT) ./cmd/converter

# Roda a API localmente
run: build
	@echo "🚀 Rodando API localmente..."
	./$(BINARY_NAME)$(EXT)

test:
	@echo "🧪 Executando testes..."
	go test -v ./...

clean:
	@echo "🧹 Limpando binários..."
	rm -f $(BINARY_NAME)$(EXT) $(CLI_NAME)$(EXT) $(CONVERTER_NAME)$(EXT)
	rm -rf $(BUILD_DIR)

# ==========================================
# 📦 BUILD E RELEASE (CROSS-PLATFORM)
# ==========================================

build-linux:
	@echo "🐧 Compilando Release Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)/linux
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/linux/$(BINARY_NAME) ./cmd/api
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/linux/$(CLI_NAME) ./cmd/admin
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/linux/$(CONVERTER_NAME) ./cmd/converter
	@echo "📄 Copiando assets..."
	cp -r static templates fonts csv xls $(BUILD_DIR)/linux/ 2>/dev/null || :
	@echo "📦 Empacotando Linux..."
	tar -czvf $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64.tar.gz -C $(BUILD_DIR)/linux .

build-windows:
	@echo "🪟 Compilando Release Windows (amd64)..."
	@mkdir -p $(BUILD_DIR)/windows
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags="-s -w" -o $(BUILD_DIR)/windows/$(BINARY_NAME).exe ./cmd/api
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags="-s -w" -o $(BUILD_DIR)/windows/$(CLI_NAME).exe ./cmd/admin
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags="-s -w" -o $(BUILD_DIR)/windows/$(CONVERTER_NAME).exe ./cmd/converter
	@echo "📄 Copiando assets..."
	cp -r static templates fonts csv xls $(BUILD_DIR)/windows/ 2>/dev/null || :
	@echo "📦 Empacotando Windows..."
	cd $(BUILD_DIR)/windows && zip -r ../$(BINARY_NAME)-windows-amd64.zip .

# ==========================================
# 🛠️ COMANDOS DE UTILIDADE (ADMINISTRADOR)
# ==========================================

cli-convert: build-converter
	@echo "📊 Convertendo arquivos XLS em 'xls/' para CSV em 'csv/'..."
	./$(CONVERTER_NAME)$(EXT) -input xls -output csv

cli-init: build-cli
	./$(CLI_NAME)$(EXT) -init-db

cli-create-admin: build-cli
	./$(CLI_NAME)$(EXT) -create-user -email "admin@localnet.com" -password "1q2w3e" -admin=true -user-id 1

cli-create-user: build-cli
	./$(CLI_NAME)$(EXT) -create-user -email "lauro@localnet.com" -password "1q2w3e" -admin=false -user-id 2

cli-import: build-cli
	@if [ -z "$(USER_ID)" ]; then \
		echo "❌ Erro: Defina o ID. Ex: make cli-import USER_ID=2"; \
	else \
		./$(CLI_NAME)$(EXT) -import -import-nacionais -import-internacionais -user-id $(USER_ID); \
	fi

dev-setup: cli-convert cli-init cli-create-admin cli-create-user
	@echo "⚡ Setup inicial..."
	$(MAKE) cli-import USER_ID=2
	@echo "✅ Ambiente pronto!"
	@echo "✅ https://app.minhaseconomias.com.br:8443"

# ==========================================
# 🐙 PODMAN / DOCKER & PRODUÇÃO
# ==========================================

up:
	@echo "🐙 Subindo stack com $(CONTAINER_TOOL) compose..."
	$(CONTAINER_TOOL) compose up -d --build
	@echo "✅ Stack rodando! Bancos e métricas ativos."
	@echo "✅ https://app.minhaseconomias.com.br:8443"

down:
	@echo "🛑 Parando containers..."
	$(CONTAINER_TOOL) compose down

clean-data:
	@echo "🔥 Apagando tudo (Volumes)..."
	$(CONTAINER_TOOL) compose down -v

logs:
	$(CONTAINER_TOOL) compose logs -f app

# ==========================================
# ⚙️ SETUP AUTOMATIZADO (DENTRO DO CONTAINER)
# ==========================================

setup-prod:
	@echo "⏳ Aguardando containers estabilizarem (5s)..."
	@sleep 5
	
	@echo "🛠️  [1/3] Inicializando Schema no Postgres..."
	$(CONTAINER_TOOL) compose exec app ./admin-cli -init-db
	
	@echo "👤 [2/3] Criando Usuário Admin..."
	$(CONTAINER_TOOL) compose exec app ./admin-cli -create-user -email="admin@localnet.com" -password="1q2w3e" -admin=true -user-id=1
	
	@echo "👤 [3/3] Criando Usuário Principal e Importando Dados..."
	$(CONTAINER_TOOL) compose exec app ./admin-cli -create-user -email="lauro@localnet.com" -password="1q2w3e" -admin=false -user-id=2

	@echo "👤 [3/3] Exportando dados dos XLS para CSV..."
	$(CONTAINER_TOOL) compose exec app ./xls-converter

	@echo "👤 [3/3] Importando Dados do usuario lauro..."
	$(CONTAINER_TOOL) compose exec app ./admin-cli -import -import-nacionais -import-internacionais -user-id=2
	
	@echo "✅ Setup em container concluído com sucesso!"
	@echo "✅ https://app.minhaseconomias.com.br:8443"

help:
	@echo "========================================================================"
	@echo "                   SISTEMA MINHAS ECONOMIAS - MAKEFILE                  "
	@echo "========================================================================"
	@echo "Uso: make [comando]"
	@echo ""
	@echo "🌐 DESENVOLVIMENTO LOCAL"
	@echo "  build            - Compila o binário da API"
	@echo "  build-cli        - Compila a CLI de administração"
	@echo "  build-converter  - Compila o conversor de arquivos XLS"
	@echo "  run              - Compila e executa a API localmente"
	@echo "  test             - Executa a suíte de testes unitários"
	@echo "  clean            - Remove binários e diretórios de build"
	@echo ""
	@echo "📦 RELEASE (CROSS-PLATFORM)"
	@echo "  build-linux      - Gera release tar.gz para Linux (amd64)"
	@echo "  build-windows    - Gera release .zip para Windows (amd64)"
	@echo ""
	@echo "🛠️  ADMINISTRAÇÃO (CLI LOCAL)"
	@echo "  cli-convert      - Converte planilhas XLS em 'xls/' para CSV em 'csv/'"
	@echo "  cli-init         - Inicializa as tabelas do banco de dados (Schema)"
	@echo "  cli-create-admin - Cria o usuário administrador padrão (ID 1)"
	@echo "  cli-create-user  - Cria o usuário comum (Lauro - ID 2)"
	@echo "  cli-import       - Importa CSVs para o DB (Uso: make cli-import USER_ID=X)"
	@echo "  dev-setup        - Fluxo completo: Converte, Inicia DB, Cria Users e Importa"
	@echo ""
	@echo "🐳 OPERAÇÃO VIA CONTAINER ($(CONTAINER_TOOL))"
	@echo "  up               - Sobe a stack completa (API, DB, Observabilidade) em background"
	@echo "  down             - Para e remove os containers da stack"
	@echo "  logs             - Segue os logs do container da aplicação"
	@echo "  clean-data       - Para a stack e APAGA todos os volumes/dados (CUIDADO)"
	@echo "  setup-prod       - Executa migrações e popula dados iniciais DENTRO do container"
	@echo ""
	@echo "========================================================================"