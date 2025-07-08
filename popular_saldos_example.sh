#!/bin/bash

# ==============================================================================
# --- CONFIGURE AQUI OS SALDOS INICIAIS DE SUAS CONTAS ---
#
# INSTRUÇÕES:
# 1. Mantenha os nomes das variáveis com underscore (_) para contas com espaços.
#    O script irá usar o nome da variável (sem o sufixo _valor) como nome da conta.
#    Ex: 'vale_refeicao_valor' resultará na conta 'vale_refeicao'.
# 2. Use ponto (.) como separador decimal para os valores.
#
# ==============================================================================

bradesco="0.00"
c6="0.00"
itau="0.00"
xp="0.00"
bb="0.00"

# --- FIM DA CONFIGURAÇÃO ---
# ==============================================================================
# ==============================================================================

# --- Lê as variáveis de ambiente do banco de dados (com valores padrão) ---
DB_TYPE="${DB_TYPE:-postgres}"
DB_USER="${DB_USER:-postgres}"
DB_PASS="${DB_PASS:-postgres}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-minhas_economias}"

# --- Verificações Iniciais ---
if [[ "$DB_TYPE" != "postgres" ]]; then
    echo "AVISO: Este script foi feito para ser executado com DB_TYPE=postgres."
    echo "O DB_TYPE atual é '$DB_TYPE'. Saindo."
    exit 0
fi

if ! command -v psql &> /dev/null; then
    echo "Erro: O comando 'psql' não foi encontrado."
    echo "Por favor, instale o cliente do PostgreSQL para executar este script."
    exit 1
fi

# Exporta a senha para que o psql não a peça interativamente.
# Esta é a forma padrão e segura de passar senhas para o psql em scripts.
export PGPASSWORD="$DB_PASS"

# Testa a conexão
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\q" &> /dev/null
if [ $? -ne 0 ]; then
    echo "Erro: Não foi possível conectar ao banco de dados PostgreSQL."
    echo "Verifique as variáveis de ambiente DB_USER, DB_PASS, DB_HOST, DB_PORT e DB_NAME."
    exit 1
fi

echo "Iniciando a atualização de saldos na tabela 'contas' do PostgreSQL..."
echo "Conectando em: postgresql://$DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
echo "------------------------------------------------------------"

# --- Lógica Principal ---
# Encontra todas as variáveis que terminam com "_valor"
for var_name in $(compgen -v | grep '_valor$'); do
    # Pega o valor da variável (ex: "8112.55")
    saldo="${!var_name}"

    # Extrai o nome da conta, removendo o sufixo "_valor"
    conta_nome="${var_name%_valor}"
    
    # Valida se o saldo é um número válido
    if ! [[ $saldo =~ ^-?[0-9]+(\.?[0-9]+)?$ ]]; then
        echo "AVISO: Valor '$saldo' para a conta '$conta_nome' não é um número válido. Pulando."
        continue
    fi

    echo "Processando: Conta = '$conta_nome', Saldo Inicial = $saldo"

    # Constrói a query SQL de "UPSERT" para o PostgreSQL
    # A sintaxe ON CONFLICT é a mesma do SQLite moderno, o que facilita a migração.
    SQL_QUERY="INSERT INTO contas (nome, saldo_inicial) VALUES ('$conta_nome', $saldo) ON CONFLICT(nome) DO UPDATE SET saldo_inicial = EXCLUDED.saldo_inicial;"

    # Executa o comando no PostgreSQL
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$SQL_QUERY"
done

# Limpa a variável de senha do ambiente
unset PGPASSWORD

echo "------------------------------------------------------------"
echo "✅ Atualização da tabela 'contas' concluída com sucesso!"