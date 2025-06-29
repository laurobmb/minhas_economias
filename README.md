# Minhas Economias: Gestão Financeira Pessoal (Web e CLI)

Este projeto oferece um sistema para gestão de movimentações financeiras pessoais, permitindo a importação e exportação de dados via linha de comando, e a visualização, filtragem, adição, edição e exclusão de registros através de uma interface web moderna.

## Descrição

Minhas Economias é uma ferramenta desenvolvida em Go, utilizando o framework Gin para a aplicação web e SQLite como banco de dados. O objetivo é fornecer uma maneira eficiente e organizada de acompanhar suas finanças, com foco em simplicidade e usabilidade.

![Minhas Economias 1](photos/view_app_1.png)

## Funcionalidades

### Aplicação Web

* **Visualização de Movimentações**: Exibe uma tabela com todas as movimentações financeiras.

* **Filtros Avançados**:

    * **Período**: Filtra por um intervalo de datas (Data Início e Data Fim). Por padrão, exibe o mês corrente.

    * **Categorias Múltiplas**: Selecione uma ou mais categorias via checkboxes.

    * **Contas Múltiplas**: Selecione uma ou mais contas via checkboxes.

    * **Consolidado**: Filtra por movimentações consolidadas (Sim/Não/Todos).

* **Resumo Financeiro**:

    * **Total Consolidado**: Mostra a soma total dos valores filtrados.

    * **Entradas**: Exibe a soma de todas as movimentações positivas (receitas) dentro do filtro.

    * **Saídas**: Exibe a soma de todas as movimentações negativas (despesas) dentro do filtro.

* **Adição de Novas Movimentações**:

    * Formulário intuitivo para inserir novos registros.

    * **Data Padrão**: A data atual é sugerida automaticamente.

    * **Tipo (Receita/Despesa)**: Selecione se é uma receita (valor positivo) ou despesa (valor negativo), e o sistema ajusta o sinal do valor automaticamente. A despesa é o padrão.

    * **Autopreenchimento de Categoria e Conta**: Sugestões baseadas em registros existentes enquanto você digita.

    * **Conta Obrigatória**: Garante que uma conta seja sempre informada.

    * **Categoria Padrão**: Se a categoria não for preenchida, é definida como "Sem Categoria".

    * **Valor Padrão**: Se o valor não for preenchido, é definido como 0.

    * **Consolidado Padrão**: O campo "Consolidado" vem desmarcado por padrão.

* **Edição de Movimentações**:

    * Botão "Editar" em cada linha da tabela.

    * Preenche o formulário de adição com os dados da movimentação selecionada.

    * O botão de envio muda para "Salvar Alterações".

    * Um botão "Cancelar Edição" permite reverter o formulário para o modo de adição.

* **Exclusão de Movimentações**:

    * Botão "Excluir" em cada linha da tabela.

    * Solicita confirmação antes de deletar um registro.

* **Visualização JSON da API**: Um botão permite ver os dados filtrados em formato JSON através da API.

### Ferramenta CLI (data_manager.go)

Uma ferramenta de linha de comando para gerenciar a importação e exportação em massa de dados.

* **Importação (`-import`)**: Lê arquivos CSV de um diretório especificado e insere os dados no banco de dados SQLite.

    * **Formato CSV esperado**: `Data Ocorrência;Descrição;Valor;Categoria;Conta;Consolidado` (com ponto e vírgula como delimitador).

* **Exportação (`-export`)**: Extrai todas as movimentações do banco de dados SQLite e gera um único arquivo CSV no formato de importação.

    * **Caminho de Saída (`-output-path`)**: Permite especificar o diretório e o nome do arquivo CSV de saída (ex: `-output-path backup/meu_extrato.csv`).

## Estrutura do Projeto

O projeto é modularizado para facilitar a manutenção e o desenvolvimento:

```

minhas\_economias/
├── main.go                       \# Ponto de entrada da aplicação web, inicializa o servidor Gin e rotas.
├── extratos.db                   \# Banco de dados SQLite (gerado e gerenciado por data\_manager.go e a webapp).
├── templates/                    \# Contém os templates HTML.
│   └── index.html                \# A interface principal da aplicação web.
├── static/                       \# Contém arquivos estáticos (CSS, ícones).
│   ├── css/
│   │   └── style.css             \# Estilos CSS da aplicação.
│   └── minhas\_economias.ico      \# Ícone da aplicação.
├── models/                       \# Definições de modelos de dados.
│   └── movimentacao.go           \# Estrutura (struct) para Movimentação.
├── database/                     \# Lógica de conexão e interação com o banco de dados.
│   └── sqlite.go                 \# Funções para inicializar e gerenciar a conexão SQLite.
└── handlers/                     \# Funções que lidam com as requisições HTTP (controladores).
└── movimentacoes.go          \# Lógica para GET, POST, PUT, DELETE de movimentações.

````

**Observação**: O arquivo `data_manager.go` (que lida com importação/exportação) fica na raiz do projeto, separado da estrutura de pacotes da aplicação web, por ser uma ferramenta utilitária.

## Tecnologias Utilizadas

* **Backend**:

    * [Go](https://golang.org/)

    * [Gin Web Framework](https://gin-gonic.com/)

    * [SQLite](https://www.sqlite.org/index.html) (via `github.com/mattn/go-sqlite3`)

* **Frontend**:

    * HTML5

    * CSS3

    * JavaScript

    * [Tailwind CSS (CDN)](https://tailwindcss.com/docs/installation/play-cdn) (para prototipagem rápida de estilos)

## Como Começar

Siga estas instruções para configurar e executar o projeto em sua máquina local.

### Pré-requisitos

Certifique-se de ter o Go instalado em sua máquina. Você pode baixá-lo em [golang.org/dl](https://golang.org/dl/).

### 1. Clonar o Repositório

```bash
git clone [https://github.com/laurobmb/minhas_economias.git](https://github.com/laurobmb/minhas_economias.git)
cd minhas_economias
````

### 2\. Inicializar o Módulo Go e Baixar Dependências

No diretório raiz do projeto (`minhas_economias/`), execute:

```bash
go mod init minhas_economias # Se você não usou este nome, ajuste para o que usou
go mod tidy
```

### 3\. Configurar Estrutura de Arquivos

Certifique-se de que os diretórios `templates/`, `static/css/` e `csv/` existam, e que os arquivos estejam em seus devidos lugares.

  * `templates/index.html`

  * `static/css/style.css`

  * `static/minhas_economias.ico`

  * Coloque seus arquivos CSV para importação no diretório `csv/`.

### 4\. Usar a Ferramenta CLI (Importar/Exportar)

O arquivo `data_manager.go` é responsável por gerenciar a importação e exportação de dados.

  * **Para importar dados de CSVs para o `extratos.db`**:

    ```bash
    go run data_manager.go -import
    ```

    (Certifique-se de que a lista `csvFiles` em `data_manager.go` aponte para seus arquivos CSV corretamente.)

  * **Para exportar dados do `extratos.db` para um CSV**:

    ```bash
    go run data_manager.go -export
    # Ou para um caminho específico:
    go run data_manager.go -export -output-path backup/meu_extrato_completo.csv
    ```

### 5\. Executar a Aplicação Web

No diretório raiz do projeto (`minhas_economias/`), execute:

```bash
go run main.go
```

Após a execução, o servidor estará ativo em `http://localhost:8080`.

## Uso

### Aplicação Web (`http://localhost:8080`)

  * **Navegação**: Acesse a URL no seu navegador.

  * **Filtros**: Use os campos "Categoria", "Conta", "Data Início", "Data Fim" e "Consolidado" para refinar a visualização dos dados.

      * Para Categoria e Conta, clique no campo para abrir as opções com checkboxes e selecione múltiplas.

      * A primeira carga da página e as limpezas de filtro exibirão os dados do mês corrente por padrão.

  * **Adicionar Movimentação**: Preencha o formulário "Adicionar Nova Movimentação".

      * Escolha "Receita" ou "Despesa" (Despesa é o padrão). O valor será automaticamente ajustado para positivo ou negativo.

      * Use o autopreenchimento para Categoria e Conta (ou digite um novo valor).

      * O campo "Conta" é obrigatório.

      * A data atual é pré-preenchida.

  * **Editar Movimentação**: Clique no botão "Editar" na linha desejada. O formulário de adição será preenchido. Faça suas alterações e clique em "Salvar Alterações". Use "Cancelar Edição" para voltar ao modo de adição.

  * **Excluir Movimentação**: Clique no botão "Excluir" na linha desejada e confirme a exclusão.

### Ferramenta CLI (data\_manager.go)

Execute a ferramenta no terminal usando as flags:

  * `go run data_manager.go -import`

  * `go run data_manager.go -export`

  * `go run data_manager.go -export -output-path caminho/do/arquivo.csv`

## Melhorias Futuras

  * **Autenticação de Usuários**: Implementar um sistema de login para múltiplos usuários.

  * **Relatórios Financeiros**: Gerar gráficos e relatórios mais detalhados.

  * **Categorias e Contas Dinâmicas**: Gerenciar categorias e contas diretamente pela interface web.

  * **Validação de Formulário no Frontend**: Adicionar validações JavaScript mais robustas para feedback instantâneo ao usuário.

  * **Otimização de Querys**: Para grandes volumes de dados, considerar índices no SQLite.

  * **Migrações de Banco de Dados**: Ferramentas para gerenciar alterações de esquema (ex: `golang-migrate`).

## Licença

Este projeto é de código aberto e está disponível sob a licença [MIT](https://opensource.org/licenses/MIT) (exemplo).

