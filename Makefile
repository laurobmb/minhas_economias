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

.PHONY: all build build-cli build-converter run test clean help

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
	# CGO pode ser 0 aqui se não usar sqlite no conversor, mas deixamos 1 por segurança
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

# Converte XLS para CSV
# Uso: make cli-convert
cli-convert: build-converter
	@echo "📊 Convertendo arquivos XLS em 'xls/' para CSV em 'csv/'..."
	./$(CONVERTER_NAME)$(EXT) -input xls -output csv

# Inicializa Tabelas
cli-init: build-cli
	./$(CLI_NAME)$(EXT) -init-db

# Cria Admin
cli-create-admin: build-cli
	./$(CLI_NAME)$(EXT) -create-user -email "admin@localnet.com" -password "1q2w3e" -admin=true -user-id 1

# Cria Usuário Comum
cli-create-user: build-cli
	./$(CLI_NAME)$(EXT) -create-user -email "lauro@localnet.com" -password "1q2w3e" -admin=false -user-id 2
	./$(CLI_NAME)$(EXT) -create-user -email "liz@localnet.com" -password "1q2w3e" -admin=false -user-id 3
	./$(CLI_NAME)$(EXT) -create-user -email "camila@localnet.com" -password "1q2w3e" -admin=false -user-id 4
	./$(CLI_NAME)$(EXT) -create-user -email "pguel@localnet.com" -password "1q2w3e" -admin=false -user-id 5

# Importa Dados (Após conversão)
cli-import: build-cli
	@if [ -z "$(USER_ID)" ]; then \
		echo "❌ Erro: Defina o ID. Ex: make cli-import USER_ID=2"; \
	else \
		./$(CLI_NAME)$(EXT) -import -import-nacionais -import-internacionais -user-id $(USER_ID); \
	fi

# Setup completo de desenvolvimento (Converte -> Cria Banco -> Cria Users -> Importa)
dev-setup: cli-convert cli-init cli-create-admin cli-create-user
	@echo "⚡ Setup inicial..."
	$(MAKE) cli-import USER_ID=2
	@echo "✅ Ambiente pronto!"