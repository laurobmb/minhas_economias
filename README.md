# Minhas Economias

**Minhas Economias** é uma aplicação web completa para gestão de finanças pessoais, desenvolvida em Go. O sistema permite que os utilizadores centralizem o seu histórico de transações, analisem os seus padrões de gastos através de relatórios visuais e personalizem a sua experiência na aplicação. O projeto foi construído com um backend robusto em Go (usando o framework Gin) e um frontend interativo com HTML, Tailwind CSS e JavaScript puro.

## ✨ Funcionalidades

* **Autenticação de Utilizador:** Sistema seguro de registo e login para garantir a privacidade dos dados financeiros.
* **Gestão de Transações (CRUD):** Interface completa para adicionar, visualizar, editar e apagar movimentações financeiras (receitas e despesas).
* **Importação de Dados:** Scripts utilitários para converter extratos em formato `.xls` para `.csv` e, em seguida, importar para a base de dados, permitindo a centralização do histórico financeiro.
* **Dashboard de Saldos:** Uma página inicial clara e objetiva que exibe o saldo consolidado de cada conta do utilizador.
* **Filtragem Avançada:** Ferramentas poderosas nas páginas de transações e relatórios para filtrar dados por descrição, período, categoria, conta e estado (consolidado/não consolidado).
* **Relatórios Visuais:** Geração de gráficos de pizza interativos que mostram a distribuição de despesas por categoria, permitindo uma análise visual dos gastos.
* **Exportação para PDF:** Funcionalidade para descarregar relatórios financeiros detalhados, incluindo gráficos e tabelas de transações, em formato PDF.
* **Página de Configurações Completa:**
    * **Gestão de Perfil:** O utilizador pode adicionar e atualizar as suas informações pessoais (data de nascimento, localização, etc.).
    * **Alteração de Senha:** Interface segura para alterar a senha da conta.
    * **Modo Escuro (Dark Mode):** Um *toggle* para alternar entre os temas claro e escuro, com a preferência a ser guardada no perfil do utilizador.
* **API RESTful:** Endpoints para interagir com os dados de forma programática.
* **Suporte a Múltiplos Bancos de Dados:** Arquitetura preparada para funcionar com PostgreSQL e SQLite.

---

## 🚀 Tecnologias Utilizadas

* **Backend:** Go, Gin Web Framework
* **Frontend:** HTML5, Tailwind CSS, JavaScript
* **Base de Dados:** PostgreSQL, SQLite
* **Geração de PDF:** Gofpdf
* **Testes de Backend:** Testes unitários/integração nativos do Go
* **Testes End-to-End (E2E):** Python com Selenium (para UI), Ansible (para API)

---

## ⚙️ Pré-requisitos

* Go (versão 1.20 ou superior)
* Podman (ou Docker) para a opção de base de dados em container, ou uma instalação local de PostgreSQL/SQLite3.
* Python (versão 3.8 ou superior, para os testes de frontend)
* Ansible (para os testes de API)
* Um navegador web (ex: Chrome) e o respetivo ChromeDriver para os testes com Selenium.

---

## 🏁 Como Começar

Siga estes passos para configurar e executar o projeto localmente.

### 1. Clonar o Repositório

```bash
git clone [https://github.com/seu-usuario/minhas_economias.git](https://github.com/seu-usuario/minhas_economias.git)
cd minhas_economias
````

### 2\. Configurar a Base de Dados (Escolha uma opção)

#### 2.a. Opção com PostgreSQL (usando Podman)

Uma forma rápida de configurar um banco de dados PostgreSQL para desenvolvimento é usando um container. O comando abaixo irá criar um container chamado `postgres`, configurar as credenciais e a base de dados, e persistir os dados no diretório `/tmp/database` do seu sistema.

```bash
# Limpa o diretório de dados antigo e cria um novo
sudo rm -rf /tmp/database
mkdir /tmp/database

# Executa o container do PostgreSQL com Podman
podman run \
    -it \
    --rm \
    --name postgres \
    -e POSTGRES_USER=me \
    -e POSTGRES_PASSWORD=1q2w3e \
    -e POSTGRES_DB=minhas_economias \
    -p 5432:5432 \
    -v /tmp/database:/var/lib/postgresql/data:Z \
    postgres:latest
```

#### 2.b. Opção Manual

Se preferir, pode configurar uma instância de PostgreSQL ou SQLite manualmente no seu sistema.

### 3\. Configurar Variáveis de Ambiente

Crie um ficheiro chamado `.env` na raiz do projeto ou exporte as seguintes variáveis de ambiente no seu terminal. **As credenciais devem corresponder às que configurou no passo anterior.**

```bash
# Chave secreta para a sessão de utilizador. Use um gerador de strings aleatórias.
export SESSION_KEY="uma-chave-secreta-muito-longa-e-segura"

# Configuração do Banco de Dados (exemplo para PostgreSQL com Podman)
export DB_TYPE="postgres"
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="me"
export DB_PASS="1q2w3e"
export DB_NAME="minhas_economias"
```

### 4\. Inicializar as Tabelas

Com o banco de dados em execução, execute o `data_manager` para criar todas as tabelas necessárias.

```bash
go run ./data_manager.go
```

### 5\. Criar um Utilizador

Use o script `create_user` para criar a sua conta. Anote o ID do utilizador que será gerado, pois precisará dele para importar os dados.

```bash
# Exemplo de criação de utilizador
go run ./create_user.go -email="seu-email@exemplo.com" -password="sua-senha-forte"
```

### 6\. Importar Transações Históricas (Opcional)

Se você possui extratos bancários em formato `.xls`, pode importá-los para a aplicação.

#### Passo 1: Converter XLS para CSV

Primeiro, coloque os seus ficheiros `.xls` dentro do diretório `xls/` na raiz do projeto. Em seguida, execute o script de conversão:

```bash
go run ./xls_to_csv.go
```

Este comando irá ler todos os ficheiros em `xls/` e criar os ficheiros `.csv` correspondentes no diretório `csv/`.

#### Passo 2: Importar CSV para a Base de Dados

Agora, use o `data_manager` para importar os dados dos ficheiros CSV para a sua conta. Substitua `SEU_USER_ID` pelo ID do utilizador que criou no passo anterior.

```bash
go run ./data_manager.go -import -user-id=SEU_USER_ID
```

### 7\. Executar a Aplicação

Finalmente, inicie o servidor web.

```bash
go run ./main.go
```

A aplicação estará disponível em `http://localhost:8080`.

-----

## 🧪 Como Executar os Testes

O projeto inclui um conjunto completo de testes.

### 1\. Testes de Backend (Go)

Estes testes validam a lógica dos handlers e as interações com a base de dados. Certifique-se de que as variáveis de ambiente do seu banco de dados de teste estão configuradas.

```bash
# Executar todos os testes de um pacote específico (ex: handlers)
go test -v ./handlers/...
```

### 2\. Testes de Frontend (Selenium)

Estes testes simulam a interação de um utilizador real com a interface.

```bash
# 1. Navegue para a pasta de testes
cd tests

# 2. Instale as dependências do Python (se necessário)
# pip install -r requirements.txt

# 3. Execute o script de teste
python ./test_app_frontend.py
```

### 3\. Testes de API (Ansible)

Estes testes verificam os endpoints da API.

```bash
# 1. Navegue para a pasta de testes
cd tests

# 2. Execute o playbook do Ansible
ansible-playbook ./test_app_api.yml
```

-----

## 📄 Licença

Este projeto está licenciado sob a Licença MIT. Veja o ficheiro `LICENSE` para mais detalhes.

