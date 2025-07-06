# Minhas Economias: Gestão Financeira Pessoal (Web e CLI)

Este projeto oferece um sistema para gestão de movimentações financeiras pessoais, permitindo a importação e exportação de dados via linha de comando, e a visualização, filtragem e gerenciamento de registros através de uma interface web moderna e modular.

![Minhas Economias Dashboard](https://i.imgur.com/your-new-screenshot-here.png)
*(Sugestão: Substitua o link acima por um novo screenshot da sua página de Saldos)*

## Descrição

Minhas Economias é uma ferramenta desenvolvida em Go, utilizando o framework Gin para a aplicação web e SQLite como banco de dados. O objetivo é fornecer uma maneira completa, privada e organizada de acompanhar suas finanças, com foco em usabilidade e robustez. A aplicação foi reestruturada para separar a visualização de saldos de alto nível da área de gerenciamento de transações.

## Funcionalidades

### Aplicação Web

A interface web é dividida em páginas dedicadas para uma melhor experiência do usuário.

* **Dashboard de Saldos (`/`)**: Exibe o saldo atual de cada conta em um layout de cartões responsivo, com indicadores visuais e links diretos para as transações.
* **Página de Transações (`/transacoes`)**: Área principal para visualizar, adicionar, editar, excluir e filtrar detalhadamente todas as movimentações.
* **Página de Relatório (`/relatorio`)**: Gera um gráfico de pizza com a distribuição de despesas e permite a exportação de um relatório em PDF.
* **Página Sobre (`/sobre`)**: Apresenta informações sobre o projeto, autor e tecnologias utilizadas.
* **Validação e Sanitização de Entradas**: Limites de caracteres e validação de formato para os campos de entrada, garantindo a integridade dos dados.

### Ferramentas de Linha de Comando (CLI)

* **`data_manager.go`**: Ferramenta para importação (`-import`) e exportação (`-export`) de dados em massa do banco de dados.
* **`xls_to_csv.go`**: Utilitário para converter extratos antigos do formato `.xls` para `.csv`.
* **`popular_saldos.sh`**: Script de shell para configurar facilmente os saldos iniciais de todas as contas no banco de dados.

## Estrutura do Projeto

```

minhas\_economias/
├── main.go                       \# Ponto de entrada da aplicação web.
├── extratos.db                   \# Banco de dados SQLite.
├── Containerfile                 \# Define como construir a imagem do contêiner.
├── .dockerignore                 \# Exclui arquivos desnecessários do build do contêiner.
│
├── templates/                    \# Templates HTML.
│   ├── index.html                \# Dashboard de Saldos.
│   ├── transacoes.html           \# Página de gerenciamento de transações.
│   ├── relatorio.html            \# Página de relatórios.
│   └── sobre.html                \# Página "Sobre".
│
├── static/                       \# Arquivos estáticos.
│   ├── css/style.css             \# Folha de estilos principal.
│   ├── js/                       \# Arquivos JavaScript.
│   │   ├── common.js             \# Funções JS compartilhadas.
│   │   ├── transacoes.js         \# Lógica da página de transações.
│   │   └── relatorio.js          \# Lógica da página de relatórios.
│   └── ...                       \# Ícones e imagens.
│
├── handlers/                     \# Controladores que lidam com as requisições HTTP.
│   ├── movimentacoes.go          \# Lógica para as páginas de saldo e transações.
│   ├── sobre.go                  \# Lógica para a página "Sobre".
│   └── movimentacoes\_test.go     \# Testes para os handlers.
│
├── test/                         \# Testes End-to-End.
│   └── test\_app\_playbook.yml     \# Playbook Ansible para testes automatizados.
│
├── data\_manager.go               \# Ferramenta CLI para importação/exportação.
├── xls\_to\_csv.go                 \# Ferramenta CLI para conversão de XLS.
└── popular\_saldos.sh             \# Script para configurar saldos iniciais.

````

## Tecnologias Utilizadas

* **Backend**: Go, Gin Web Framework, SQLite
* **Frontend**: HTML5, CSS3, JavaScript, Tailwind CSS (via CDN)
* **Testes**: Testes Nativos do Go, Ansible
* **Contêineres**: Docker / Podman

## Como Começar (Executando Localmente)

### 1. Clonar o Repositório e Preparar Dependências
```bash
git clone [https://github.com/laurobmb/minhas_economias.git](https://github.com/laurobmb/minhas_economias.git)
cd minhas_economias
go mod tidy
````

### 2\. Criar Tabelas no Banco de Dados

Este é o primeiro passo essencial. Crie o esquema do banco de dados executando:

```bash
go run data_manager.go -import
```

### 3\. Configurar Saldos Iniciais

Edite o script `popular_saldos.sh` para definir os saldos iniciais de todas as suas contas. Depois, execute-o:

```bash
# Edite o arquivo primeiro!
./popular_saldos.sh
```

### 4\. Executar a Aplicação Web

Para iniciar o servidor, você pode definir as variáveis de ambiente para a página "Sobre":

```bash
# Exemplo para Linux/macOS
AUTHOR_NAME="Seu Nome" GITHUB_URL="[https://github.com/seu_usuario](https://github.com/seu_usuario)" go run main.go

# Exemplo para Windows (PowerShell)
# $env:AUTHOR_NAME="Seu Nome"; $env:GITHUB_URL="[https://github.com/seu_usuario](https://github.com/seu_usuario)"; go run main.go
```

Após a execução, o servidor estará ativo em `http://localhost:8080`.

## Executando com Contêineres (Docker/Podman)

Para uma implantação mais fácil e consistente, o projeto pode ser executado dentro de um contêiner.

### Pré-requisitos

  * [Docker](https://www.docker.com/get-started) ou uma ferramenta compatível (como [Podman](https://podman.io/)) instalado.

### Passos para Execução

1.  **Prepare o Banco de Dados (Passo Essencial):** O contêiner irá copiar o arquivo `extratos.db` que existe localmente. Portanto, você deve primeiro criar e popular seu banco de dados na sua máquina, seguindo os passos 2 e 3 da seção "Como Começar (Executando Localmente)".

2.  **Construa a Imagem:** No diretório raiz do projeto, execute o comando para construir a imagem Docker. Isso lerá o `Containerfile` e empacotará a aplicação.

    ```bash
    podman build -t minhas-economias .
    ```

3.  **Execute o Contêiner:** Após a imagem ser construída, inicie um contêiner a partir dela.

    ```bash
    # Execute o comando abaixo, substituindo os valores das variáveis de ambiente
    podman run \
      --rm \
      -p 8080:8080 \
      -e AUTHOR_NAME="Seu Nome" \
      -e GITHUB_URL="[https://github.com/seu_usuario](https://github.com/seu_usuario)" \
      -e LINKEDIN_URL="[https://linkedin.com/in/seu-perfil](https://linkedin.com/in/seu-perfil)" \
      --name minhas-economias-app \
      minhas-economias
    ```

      * `-p 8080:8080`: Mapeia a porta do seu computador para a porta do contêiner.
      * `--rm`: Remove o contêiner automaticamente quando ele é parado (ótimo para desenvolvimento).
      * `-e`: Define as variáveis de ambiente para a página "Sobre".
      * `--name`: Dá um nome ao seu contêiner para facilitar o gerenciamento.

4.  **Acesse a Aplicação**: Agora, a aplicação estará rodando e acessível em `http://localhost:8080` no seu navegador.

## Licença

Este projeto é de código aberto e está disponível sob a [Licença MIT](https://opensource.org/licenses/MIT).
