# Estágio 1: Builder
FROM golang:1.23-alpine AS builder

# Instala dependências necessárias para compilar CGO (necessário para sqlite3)
RUN apk add --no-cache \
    build-base \
    gcc \
    musl-dev

WORKDIR /src

# Cache de dependências do Go
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte
COPY . .

# Compilação da API Principal
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/minhaseconomias ./cmd/api/main.go

# Compilação da CLI de Administração (admin-cli)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/admin-cli ./cmd/admin/

# Compilação do Conversor XLS (Opcional, mas útil ter no container)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/xls-converter ./cmd/converter/main.go

# Estágio 2: Final (Runtime)
FROM alpine:latest

# Instala CA-Certificates para chamadas externas de API (Gemini/Scraping)
# e fuso horário para consistência nos logs e transações
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Cria um usuário não-root (boa prática para OpenShift/Kubernetes)
RUN adduser -D -u 1001 appuser && \
    mkdir -p /app/data && \
    chown -R appuser:appuser /app

# 1. Copia os BINÁRIOS (API, CLI e o CONVERSOR)
COPY --from=builder --chown=appuser:appuser /app/minhaseconomias .
COPY --from=builder --chown=appuser:appuser /app/admin-cli .
COPY --from=builder --chown=appuser:appuser /app/xls-converter .

# 2. Copia os ASSETS de interface (Públicos)
COPY --from=builder --chown=appuser:appuser /src/templates ./templates
COPY --from=builder --chown=appuser:appuser /src/static ./static
COPY --from=builder --chown=appuser:appuser /src/fonts ./fonts

# Variáveis de ambiente padrão (podem ser sobrescritas no Deployment)
ENV GIN_MODE=release \
    PORT=8080 \
    TZ=America/Sao_Paulo \
    DB_TYPE=sqlite3 \
    DB_NAME=/app/data/extratos.db

# Define o usuário para execução
USER 1001

EXPOSE 8080

# O binário agora é o entrypoint
ENTRYPOINT ["./minhaseconomias"]