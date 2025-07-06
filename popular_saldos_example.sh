#!/bin/bash

# ==============================================================================
# --- CONFIGURE AQUI OS SALDOS INICIAIS DE SUAS CONTAS ---
#
# INSTRUÇÕES:
# 1. Substitua o valor "0.00" de cada conta abaixo pelo saldo real que ela
#    possuía no dia ANTERIOR à sua primeira transação (ex: 31/12/2012).
# 2. IMPORTANTE: Contas com espaços no nome devem usar underscore (_).
#    Ex: "vale refeição" vira "vale_refeicao_valor"
# 3. Use ponto (.) como separador decimal para os valores (ex: "1500.50").
#
# ==============================================================================

Banco_do_Brasil_valor="2350.75"
Itau_Corretora_valor="10000.00"
Cartao_XP_valor="-850.40"
Carteira_Fisica_valor="150.00"
Caixinha_NuBank_valor="0"

# --- FIM DA CONFIGURAÇÃO ---
# (Não altere o código abaixo desta linha)
# ==============================================================================

# Define o nome do arquivo do banco de dados
DB_FILE="extratos.db"

# --- Verificações Iniciais ---
if ! command -v sqlite3 &> /dev/null; then
    echo "Erro: O comando 'sqlite3' não foi encontrado."
    echo "Por favor, instale o sqlite3 para executar este script (ex: sudo apt-get install sqlite3)."
    exit 1
fi

if [ ! -f "$DB_FILE" ]; then
    echo "Erro: O arquivo de banco de dados '$DB_FILE' não foi encontrado."
    exit 1
fi

echo "Iniciando a atualização de saldos na tabela 'contas'..."
echo "As contas serão criadas se não existirem, ou atualizadas se já existirem."
echo "------------------------------------------------------------"

# --- Lógica Principal ---
# Encontra todas as variáveis que terminam com "_valor"
for var_name in $(compgen -v | grep '_valor$'); do
    # Pega o valor da variável (ex: "2350.75")
    saldo="${!var_name}"

    # Extrai o nome da conta:
    # 1. Remove o sufixo "_valor" do nome da variável
    # 2. Substitui todos os underscores de volta para espaços
    conta_nome_raw="${var_name%_valor}"
    conta_nome="${conta_nome_raw//_/ }"

    # Valida se o saldo é um número válido (pode ser negativo ou decimal)
    if ! [[ $saldo =~ ^-?[0-9]+(\.?[0-9]+)?$ ]]; then
        echo "AVISO: Valor '$saldo' para a conta '$conta_nome' não é um número válido. Pulando."
        continue
    fi

    echo "Processando: Conta = '$conta_nome', Saldo Inicial = $saldo"

    # Executa o comando "UPSERT":
    # - INSERT: Insere um novo registro se a conta (nome) não existir.
    # - ON CONFLICT...DO UPDATE: Se a conta já existir, ele apenas atualiza o saldo_inicial.
    sqlite3 "$DB_FILE" "INSERT INTO contas (nome, saldo_inicial) VALUES ('$conta_nome', $saldo) ON CONFLICT(nome) DO UPDATE SET saldo_inicial = excluded.saldo_inicial;"

done

echo "------------------------------------------------------------"
echo "✅ Atualização da tabela 'contas' concluída com sucesso!"