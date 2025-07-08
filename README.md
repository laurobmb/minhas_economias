# Minhas Economias: Gestão Financeira Pessoal (Web e CLI)

Este projeto oferece um sistema para gestão de movimentações financeiras pessoais, com suporte a bancos de dados PostgreSQL e SQLite. Ele permite a importação e exportação de dados via linha de comando, e a visualização, filtragem e gerenciamento de registros através de uma interface web moderna e modular.

![Minhas Economias Dashboard](https://i.imgur.com/GjD3F9q.png)
*(Screenshot da aplicação em execução)*

## Descrição

O Minhas Economias é uma ferramenta desenvolvida em Go, utilizando o framework Gin para a aplicação web. O objetivo é fornecer uma maneira completa, privada e organizada de acompanhar suas finanças. Após uma significativa refatoração, o projeto agora suporta **PostgreSQL** como banco de dados principal (recomendado), mantendo a opção de usar **SQLite** para simplicidade e portabilidade. Toda a configuração do banco de dados é feita de forma flexível através de variáveis de ambiente.

## Funcionalidades

* **Backend Flexível**: Suporte nativo para PostgreSQL e SQLite, configurável via variáveis de ambiente.
* **Dashboard de Saldos (`/`)**: Exibe o saldo atual de cada conta em um layout de cartões, com indicadores visuais e links diretos para as transações.
* **Página de Transações (`/transacoes`)**: Área principal para visualizar, adicionar, editar, excluir e filtrar detalhadamente todas as movimentações.
* **Página de Relatório (`/relatorio`)**: Gera um gráfico de pizza com a distribuição de despesas e permite a exportação de um relatório completo em PDF.
* **API Robusta**: Endpoints JSON para consultar dados e realizar ações, facilitando testes automatizados e integrações futuras.
* **Validação de Dados**: Validação no backend para garantir a integridade dos dados inseridos através dos formulários.

## Tecnologias Utilizadas

* **Backend**: Go, Gin Web Framework, PostgreSQL / SQLite
* **Frontend**: HTML5, CSS3, JavaScript, Tailwind CSS (via CDN), Chart.js
* **Testes**: Testes Nativos do Go, Ansible (API), Selenium com Python (E2E)
* **Contêineres**: Docker / Podman, Docker Compose

## Estrutura do Projeto

```

minhas\_economias/
├── main.go                       \# Ponto de entrada da aplicação web.
├── go.mod / go.sum               \# Gerenciamento de dependências do Go.
│
├── database/
│   └── database.go               \# Lógica de conexão para PostgreSQL e SQLite.
│
├── handlers/                     \# Controladores que lidam com as requisições HTTP.
│   ├── handlers.go               \# Funções auxiliares de handlers.
│   ├── movimentacoes.go          \# Lógica para as páginas principais.
│   └── ...
│
├── templates/                    \# Templates HTML.
├── static/                       \# Arquivos estáticos (CSS, JS, Imagens).
│
├── test/
│   ├── ansible\_playbook.yml      \# Playbook Ansible para testes de API.
│   └── selenium\_test.py          \# Script Python para testes End-to-End com Selenium.
│
├── data\_manager.go               \# Ferramenta CLI para importação/exportação de dados.
├── xls\_to\_csv.go                 \# Ferramenta CLI para conversão de XLS.
├── popular\_saldos.sh             \# Script para configurar saldos iniciais.
│
├── Containerfile                 \# Define como construir a imagem da aplicação.
├── docker-compose.yml            \# Orquestra os serviços da aplicação e do banco de dados.
└── .env.example                  \# Arquivo de exemplo para variáveis de ambiente.

````

## Como Começar (Executando Localmente)

### 1. Pré-requisitos
* Go (versão 1.18 ou superior).
* Se for usar SQLite: O pacote `sqlite3` instalado na sua máquina.
* **Se for usar PostgreSQL (Recomendado):** É pré-requisito ter um servidor PostgreSQL acessível. A instalação, configuração, tunning e práticas de segurança do PostgreSQL estão fora do escopo deste README. O cliente `psql` também deve estar instalado para rodar os scripts de ajuda.

<details>
<summary><b>➡️ Exemplo: Subindo um PostgreSQL rapidamente com Podman/Docker</b></summary>

Se você tem Podman ou Docker, pode subir um banco de dados PostgreSQL para desenvolvimento com o seguinte comando:

```bash
podman run \
    -it \
    --rm \
    --name postgres-dev \
    -e POSTGRES_USER=me \
    -e POSTGRES_PASSWORD=1q2w3e \
    -e POSTGRES_DB=minhas_economias \
    -p 5432:5432 \
    -v /tmp/database:/var/lib/postgresql/data:Z \
    postgres:latest
````

**Análise do comando:**

  - `-e`: Define as variáveis de ambiente para criar o usuário, senha e banco de dados iniciais.
  - `-p 5432:5432`: Expõe a porta do PostgreSQL para que sua aplicação Go possa se conectar a ela em `localhost:5432`.
  - `-v /tmp/database...`: Cria um volume para persistir os dados do banco na sua máquina local (no diretório `/tmp/database`), para que você não perca tudo ao reiniciar o contêiner. A flag `:Z` ajusta o contexto de segurança do SELinux, se aplicável.

\</details\>

### 2\. Clonar e Configurar o Projeto

```bash
git clone [https://github.com/laurobmb/minhas_economias.git](https://github.com/laurobmb/minhas_economias.git)
cd minhas_economias

# Instalar as dependências do Go
go mod tidy
```

### 3\. Configurar as Variáveis de Ambiente

A aplicação e os scripts são configurados via variáveis de ambiente. A maneira mais fácil é criar um arquivo `.env`.

```bash
# Copie o arquivo de exemplo
cp .env.example .env
```

Agora, **edite o arquivo `.env`** com as configurações do seu banco de dados. Elas devem corresponder ao banco que você configurou no passo 1.

**Exemplo de `.env` para PostgreSQL:**

```env
# TIPO DO BANCO: "postgres" ou "sqlite3"
DB_TYPE=postgres

# CONFIGURAÇÕES DO POSTGRESQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=me
DB_PASS=1q2w3e
DB_NAME=minhas_economias

# Variáveis da página "Sobre"
AUTHOR_NAME="Seu Nome"
GITHUB_URL="[https://github.com/seu_usuario](https://github.com/seu_usuario)"
LINKEDIN_URL="[https://linkedin.com/in/seu-perfil](https://linkedin.com/in/seu-perfil)"
```

### 4\. Executar a Aplicação

Para carregar as variáveis do arquivo `.env` automaticamente, você pode usar uma biblioteca como a `godotenv`. Assumindo que você a adicionou ao projeto (`go get github.com/joho/godotenv`), você pode rodar a aplicação.

```bash
# Inicia a aplicação web
go run main.go
```

O servidor estará ativo em `http://localhost:8080`.

## Scripts de Gerenciamento de Dados

Execute estes scripts no terminal após configurar seu arquivo `.env`. Eles lerão as variáveis de ambiente para se conectar ao banco de dados correto.

  * **Criar as tabelas do banco de dados:**

    ```bash
    go run data_manager.go -import
    ```

    *(Nota: A flag `-import` é usada para garantir que o programa execute a lógica de criação de tabelas, mesmo que nenhum arquivo CSV seja processado.)*

  * **Popular os saldos iniciais:**

    1.  Edite os valores no script `popular_saldos.sh`.
    2.  Execute o script:
        ```bash
        # Certifique-se de que o arquivo .env está configurado
        ./popular_saldos.sh
        ```

  * **Converter XLS para CSV:**

    ```bash
    go run xls_to_csv.go
    ```

## Executando com Contêineres (Docker Compose)

Esta é a forma mais robusta para um ambiente de desenvolvimento, pois orquestra a aplicação e o banco de dados juntos.

1.  **Pré-requisitos**: Docker e Docker Compose instalados.

2.  **Configuração**: Copie o arquivo de exemplo `.env.example` para `.env` e preencha com suas configurações, principalmente `DB_PASS`.

    ```bash
    cp .env.example .env
    ```

3.  **Suba os Serviços**: No diretório raiz do projeto, execute:

    ```bash
    docker-compose up --build
    ```

    Este comando irá:

      * Construir a imagem da sua aplicação Go (`app`).
      * Baixar e iniciar um contêiner do PostgreSQL (`db`).
      * Configurar uma rede para que os dois conversem.
      * Criar um volume para persistir os dados do PostgreSQL.

4.  **Acesse a Aplicação**: A aplicação estará disponível em `http://localhost:8080`.

5.  **Para executar os scripts de gerenciamento**, use o `docker-compose exec`:

    ```bash
    # Para criar as tabelas
    docker-compose exec app go run data_manager.go -import

    # Para popular os saldos
    docker-compose exec app ./popular_saldos.sh
    ```

## Licença

Este projeto é de código aberto e está disponível sob a [Licença MIT](https://opensource.org/licenses/MIT).
