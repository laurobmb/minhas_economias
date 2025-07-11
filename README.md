# Minhas Economias - Controle Financeiro Pessoal

## 🎯 Sobre o Projeto

**Minhas Economias** é um sistema de controle financeiro pessoal, de código aberto, construído em Go. Ele foi projetado para ser uma solução centralizada e privada, permitindo que o usuário tenha total controle sobre seus dados financeiros. A aplicação oferece uma interface web moderna e responsiva para gerenciar transações, visualizar saldos, acompanhar investimentos e gerar relatórios detalhados.

Com suporte para múltiplos bancos de dados (PostgreSQL e SQLite) e uma arquitetura robusta, o projeto é ideal tanto para uso pessoal quanto como um estudo de caso de desenvolvimento de software em Go.

-----

## ✨ Funcionalidades Principais

  - **Dashboard de Saldos:** Visualização rápida e clara do saldo atual de todas as contas cadastradas.
  - **Gerenciamento de Transações:** Interface completa para adicionar, editar, excluir e filtrar todas as movimentações financeiras.
  - **Acompanhamento de Investimentos:**
      - Monitoramento de Ações Nacionais, Fundos Imobiliários (FIIs) e Ativos Internacionais.
      - Atualização de preços em tempo real através de scraping e APIs externas.
      - Cálculo de indicadores importantes como P/VP, Dividend Yield e Valor de Graham.
      - CRUD completo para gerenciar a carteira de investimentos diretamente na interface.
  - **Relatórios Visuais:** Gráficos interativos que ajudam a entender os padrões de gastos, com opção de exportar relatórios detalhados em PDF.
  - **Importação e Exportação de Dados:**
      - Ferramenta de linha de comando (`data_manager.go`) para importar extratos bancários e carteiras de investimentos a partir de arquivos CSV.
      - Funcionalidade para exportar transações filtradas para um arquivo CSV.
  - **Autenticação e Personalização:** Sistema de registro e login de usuários, com opções de personalização como o Modo Escuro.
  - **Arquitetura Assíncrona:** A página de investimentos carrega os dados de preço em background, proporcionando uma experiência de usuário mais rápida e fluida.

-----

## 🚀 Instalação e Configuração

Siga os passos abaixo para configurar e executar o projeto em seu ambiente local.

### 1\. Pré-requisitos

  - **Go:** Versão 1.18 ou superior.
  - **Banco de Dados:**
      - **PostgreSQL:** (Recomendado para produção) ou
      - **SQLite:** (Padrão para desenvolvimento, não requer instalação adicional).
  - **Git:** Para clonar o repositório.

### 2\. Clonar o Repositório

```bash
git clone https://github.com/seu-usuario/minhas_economias.git
cd minhas_economias
```

### 3\. Instalar Dependências

O Go Modules cuidará da maior parte do trabalho. Execute o comando abaixo para baixar as dependências necessárias.

```bash
go mod tidy
```

### 4\. Configurar Variáveis de Ambiente

A aplicação utiliza variáveis de ambiente para configurar a conexão com o banco de dados e a chave de sessão. A forma mais fácil é criar um arquivo `.env` na raiz do projeto.

**Exemplo de arquivo `.env` para PostgreSQL:**

```env
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=sua_senha_segura
DB_NAME=minhas_economias
SESSION_KEY=uma-chave-secreta-muito-longa-e-aleatoria
```

**Exemplo de arquivo `.env` para SQLite:**

```env
DB_TYPE=sqlite3
DB_NAME=minhas_economias.db
SESSION_KEY=uma-chave-secreta-muito-longa-e-aleatoria
```

**Importante:** Para carregar essas variáveis automaticamente, você pode usar um pacote como o `godotenv` ou simplesmente exportá-las no seu terminal antes de rodar a aplicação.

### 5\. Preparar o Banco de Dados

O script `data_manager.go` é responsável por criar todas as tabelas necessárias no banco de dados.

Execute o comando abaixo (com uma flag qualquer, como `-h` para ajuda) para forçar a verificação e criação das tabelas.

```bash
go run data_manager.go -h
```

### 6\. Criar Usuários e Popular Dados (Opcional)

#### a) Criar um Usuário Administrador

É recomendado ter um usuário administrador (ID 1) para testes e gerenciamento.

```bash
go run create_user.go -id 1 -email "admin@localnet.com" -password "senha_admin" -admin=true
```

#### b) Criar um Usuário Comum e Importar Dados

Crie um usuário comum para uso diário.

```bash
go run create_user.go -email "usuario@exemplo.com" -password "senha_usuario"
```

*Este comando irá retornar o ID do usuário criado (ex: ID 2).*

Com o ID do usuário em mãos, você pode importar os dados de exemplo dos arquivos CSV.

```bash
# Importar movimentações financeiras
go run data_manager.go -import -user-id 2

# Importar carteira de investimentos nacionais e internacionais
go run data_manager.go -import-nacionais -import-internacionais -user-id 2
```

### 7\. Iniciar a Aplicação

Com tudo configurado, inicie o servidor web:

```bash
go run main.go
```

A aplicação estará disponível em `http://localhost:8080`.

-----

## 🧪 Testes

O projeto possui uma suíte de testes completa, dividida em três camadas.

### 1\. Testes de Backend (Go)

Estes testes verificam a lógica dos handlers e a interação com o banco de dados. Eles são executados em um banco de dados SQLite em memória para não afetar seus dados.

Para rodar todos os testes de backend:

```bash
go test ./... -v
```

### 2\. Testes de API (Ansible)

O playbook do Ansible testa os endpoints da API de forma independente, verificando status, respostas e contratos.

**Pré-requisito:** Instalar o Ansible e as coleções necessárias.

```bash
pip install -r tests/requirements.txt
ansible-galaxy collection install ansible.builtin
```

**Executar o Playbook:**

```bash
ansible-playbook tests/test_app_api.yml
```

### 3\. Testes de Frontend (Selenium)

Estes testes simulam a interação de um usuário real com a interface da aplicação, cobrindo fluxos completos de CRUD e outras funcionalidades.

**Pré-requisito:** Instalar as dependências do Python e o `chromedriver`.

```bash
pip install -r tests/requirements.txt
# Certifique-se de que o chromedriver está no seu PATH ou especifique o caminho.
```

**Executar os Testes de UI:**

```bash
python3 tests/test_app_frontend.py
```

-----

## 🏛️ Arquitetura e Tecnologias

  - **Backend:** Go
  - **Framework Web:** Gin
  - **Banco de Dados:** PostgreSQL, SQLite
  - **Frontend:** HTML5, Tailwind CSS, JavaScript
  - **Geração de PDF:** Gofpdf
  - **Web Scraping:** Colly
  - **Testes:** Go Testing, Ansible, Python + Selenium

-----

## 🤝 Como Contribuir

Contribuições são muito bem-vindas\! Se você encontrar um bug ou tiver uma ideia para uma nova funcionalidade, sinta-se à vontade para:

1.  Fazer um Fork do projeto.
2.  Criar uma nova Branch (`git checkout -b feature/sua-feature`).
3.  Fazer o Commit de suas mudanças (`git commit -m 'Adiciona sua-feature'`).
4.  Fazer o Push para a Branch (`git push origin feature/sua-feature`).
5.  Abrir um Pull Request.

-----

## 📜 Licença

Este projeto é distribuído sob a Licença MIT. Veja o arquivo `LICENSE` para mais detalhes.