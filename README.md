# Minhas Economias: Gestão Financeira Pessoal (Web e CLI)

Este projeto oferece um sistema para gestão de movimentações financeiras pessoais, permitindo a importação e exportação de dados via linha de comando, e a visualização, filtragem e gerenciamento de registros através de uma interface web moderna e modular.

![Minhas Economias 1](photos/me1.png)

![Minhas Economias 2](photos/me2.png)

![Minhas Economias 3](photos/me3.png)

## Descrição

Minhas Economias é uma ferramenta desenvolvida em Go, utilizando o framework Gin para a aplicação web e SQLite como banco de dados. O objetivo é fornecer uma maneira completa, privada e organizada de acompanhar suas finanças, com foco em usabilidade e robustez. A aplicação foi reestruturada para separar a visualização de saldos de alto nível da área de gerenciamento de transações.

## Funcionalidades

### Aplicação Web

A interface web é dividida em páginas dedicadas para uma melhor experiência do usuário.

#### 1. Dashboard de Saldos (`/`)
* **Visão Geral Moderna:** Exibe o saldo atual de cada conta em um layout de cartões responsivo.
* **Indicador Visual:** Cada cartão possui uma borda colorida (verde para positivo, vermelho para negativo) para uma rápida identificação do estado da conta.
* **Navegação Direta:** Cada cartão contém um link direto para a página de transações, já filtrada para aquela conta específica.

#### 2. Página de Transações (`/transacoes`)
* **Gerenciamento Completo:** A área principal para visualizar, adicionar, editar e excluir movimentações.
* **Tabela Detalhada:** Exibe todas as transações filtradas com colunas para Data, Descrição, Valor, Categoria, Conta e status de Consolidação.
* **Filtros Avançados**:
    * **Período**: Filtra por um intervalo de datas. Por padrão, exibe o mês corrente.
    * **Categorias e Contas Múltiplas**: Selecione uma ou mais opções em dropdowns personalizados.
    * **Busca por Descrição**: Campo de texto para buscar movimentações específicas.
    * **Entradas/Saídas/Consolidado**: Filtros rápidos para ver apenas receitas, despesas ou status de consolidação.
* **Resumo Financeiro Dinâmico**: Exibe os totais de Entradas, Saídas e o Saldo Total para o período e filtros selecionados.
* **Adição e Edição**: Formulário integrado para adicionar novas transações ou editar existentes, com o formulário sendo preenchido automaticamente ao clicar em "Editar".
* **Exclusão Segura**: Botão para excluir transações com uma janela de confirmação para evitar acidentes.

#### 3. Página de Relatório (`/relatorio`)
* **Análise Visual:** Gera um gráfico de pizza com a distribuição de despesas por categoria para o período filtrado.
* **Exportação para PDF:** Permite baixar um relatório completo em PDF, contendo os filtros aplicados, o gráfico e uma lista detalhada de todas as transações.

#### 4. Página Sobre (`/sobre`)
* **Informações do Projeto:** Apresenta um resumo do projeto, suas funcionalidades e tecnologias.
* **Dados do Autor:** Exibe informações de contato (configuráveis via variáveis de ambiente).
* **Licença de Uso:** Detalha a licença do software.

#### 5. Validação e Sanitização de Entradas
* **Campo de Valor**:
    * **Frontend**: Limita a digitação a caracteres numéricos e a um máximo de 13 caracteres.
    * **Backend**: Valida o formato monetário (até 2 casas decimais) e rejeita valores absolutos acima de 100 milhões.
* **Campo de Descrição**:
    * **Frontend**: Limita a entrada a um máximo de 60 caracteres.
    * **Backend**: Garante que nenhuma descrição com mais de 60 caracteres seja salva no banco de dados.

### Ferramentas de Linha de Comando (CLI)

#### 1. `data_manager.go`
* **Importação (`-import`)**: Lê arquivos CSV e popula o banco de dados. É a ferramenta usada para criar o esquema do banco pela primeira vez.
* **Exportação (`-export`)**: Extrai todos os dados do banco para um único arquivo CSV, ideal para backups.

#### 2. `xls_to_csv.go`
* **Conversor de Legado**: Uma ferramenta auxiliar para converter arquivos de extrato antigos no formato `.xls` para o formato `.csv` compatível com o `data_manager`.

#### 3. `popular_saldos.sh`
* **Configuração de Contas**: Um script de shell para popular a tabela `contas` com os saldos iniciais. É um passo crucial na configuração inicial para garantir que os saldos calculados pela aplicação estejam corretos.

## Estrutura do Projeto

O projeto é modularizado para facilitar a manutenção e o desenvolvimento:

```

minhas\_economias/
├── main.go                       \# Ponto de entrada da aplicação web.
├── extratos.db                   \# Banco de dados SQLite.
├── go.mod & go.sum               \# Gerenciamento de dependências.
│
├── templates/                    \# Templates HTML.
│   ├── index.html                \# Dashboard de Saldos.
│   ├── transacoes.html           \# Página de gerenciamento de transações.
│   ├── relatorio.html            \# Página de relatórios.
│   └── sobre.html                \# Nova página "Sobre".
│
├── static/                       \# Arquivos estáticos.
│   ├── css/style.css             \# Folha de estilos principal.
│   ├── js/                       \# Arquivos JavaScript.
│   │   ├── transacoes.js         \# Lógica da página de transações.
│   │   └── relatorio.js          \# Lógica da página de relatórios.
│   └── ...                       \# Ícones e imagens.
│
├── handlers/                     \# Controladores que lidam com as requisições HTTP.
│   ├── movimentacoes.go          \# Lógica para as páginas de saldo e transações.
│   ├── sobre.go                  \# Lógica para a página "Sobre".
│   └── movimentacoes\_test.go     \# Testes unitários/integração para os handlers.
│
├── database/ & models/           \# Pacotes para acesso ao banco e modelos de dados.
│
├── pdfgenerator/                 \# Lógica para a geração de relatórios em PDF.
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
* **Ferramentas**: Diversas bibliotecas Go para manipulação de XLS, PDF e banco de dados.

## Como Começar

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

*(Isso criará as tabelas `movimentacoes` e `contas`, mesmo que não haja CSVs para importar).*

### 3\. Configurar Saldos Iniciais

Edite o script `popular_saldos.sh` para definir os saldos iniciais de todas as suas contas. Depois, execute-o:

```bash
# Edite o arquivo primeiro!
./popular_saldos.sh
```

### 4\. Importar Dados (Opcional)

Se você tiver arquivos CSV (ou XLS convertidos), coloque-os nos diretórios apropriados e rode as ferramentas CLI para importar os dados.

### 5\. Executar a Aplicação Web

Para iniciar o servidor, você pode definir as variáveis de ambiente para a página "Sobre":

```bash
# Exemplo para Linux/macOS
AUTHOR_NAME="Seu Nome" GITHUB_URL="[https://github.com/seu_usuario](https://github.com/seu_usuario)" go run main.go

# Exemplo para Windows (PowerShell)
# $env:AUTHOR_NAME="Seu Nome"; $env:GITHUB_URL="[https://github.com/seu_usuario](https://github.com/seu_usuario)"; go run main.go
```

Após a execução, o servidor estará ativo em `http://localhost:8080`.

## Melhorias Futuras

  * **Autenticação de Usuários**: Implementar um sistema de login.
  * **Gerenciamento de Contas/Categorias**: Permitir criar, editar e excluir contas e categorias pela interface web.
  * **Testes de Frontend**: Adicionar testes de interface com ferramentas como Cypress ou Playwright.
  * **Migrações de Banco de Dados**: Adotar uma ferramenta para gerenciar alterações de esquema (ex: `golang-migrate`).

## Licença

Este projeto é de código aberto e está disponível sob a [Licença MIT](https://opensource.org/licenses/MIT).
