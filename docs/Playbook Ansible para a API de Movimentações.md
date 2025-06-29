# Playbook Ansible para a API de Movimentações

Este playbook Ansible demonstra como interagir com a API de Gestão de Movimentações Financeiras (seu backend Go) para criar, atualizar e excluir registros. Ele utiliza o módulo `ansible.builtin.uri` para realizar as requisições HTTP.

Certifique-se de que seu servidor Go (`main.go`) esteja em execução (`http://localhost:8080`) antes de executar este playbook.

### **`movimentacoes_api.yml`**

Crie um arquivo chamado `movimentacoes_api.yml` e adicione o seguinte conteúdo:

```yaml
---
- name: Gerenciar Movimentações Financeiras via API Go
  hosts: localhost # Ou o IP/nome de host do seu servidor Go, se não for local
  connection: local # Para executar curl na própria máquina Ansible
  gather_facts: false

  vars:
    api_base_url: "http://localhost:8080"
    # Para exemplos de Update/Delete, você precisará de um ID de movimentação existente.
    # Obtenha um ID válido acessando http://localhost:8080 ou http://localhost:8080/api/movimentacoes
    movimentacao_id_para_operacoes: 1 # <-- ALtere este ID para uma movimentação real no seu banco de dados para testar update/delete

  tasks:
    - name: 1. Criar uma Nova Movimentação (POST)
      ansible.builtin.uri:
        url: "{{ api_base_url }}/movimentacoes"
        method: POST
        body_format: form-urlencoded
        body:
          data_ocorrencia: "2025-07-01"
          descricao: "Compra de Livro Ansible"
          valor: "-85.00"
          categoria: "Educacao"
          conta: "Cartao Credito"
          consolidado: "on" # Ou remova a linha para 'false'
        status_code: 302 # Espera um redirecionamento (Status Found)
      register: create_result
      ignore_errors: yes # Para permitir que o playbook continue mesmo se a criação falhar (ex: erro de validação)
      tags: create

    - name: Verificar resultado da criação (se houver redirecionamento, não haverá JSON)
      debug:
        msg: "Redirecionamento após criação (esperado). Status: {{ create_result.status }}. Location: {{ create_result.location }}"
      when: create_result.status == 302
      tags: create

    - name: Verificar falha na criação (se status não for 302)
      debug:
        msg: "Falha na criação. Status: {{ create_result.status }}. Resposta: {{ create_result.content | default('N/A') }}"
      when: create_result.status != 302 and create_result.failed
      tags: create

    - name: 2. Atualizar uma Movimentação Existente (POST para rota de update)
      # Nota: A API Go usa POST para a rota de update.
      # Certifique-se de que 'movimentacao_id_para_operacoes' seja um ID válido no seu DB.
      ansible.builtin.uri:
        url: "{{ api_base_url }}/movimentacoes/update/{{ movimentacao_id_para_operacoes }}"
        method: POST
        body_format: form-urlencoded
        body:
          data_ocorrencia: "2025-07-02" # Data atualizada
          descricao: "Descricao Editada via Ansible" # Descrição atualizada
          valor: "-250.75" # Valor atualizado (certifique-se de que o sinal esteja correto para Despesa/Receita)
          categoria: "Servicos" # Categoria atualizada
          conta: "Conta Corrente" # Conta atualizada
          consolidado: "off" # Consolidado atualizado
        status_code: 302 # Espera um redirecionamento (Status Found)
      register: update_result
      ignore_errors: yes # Para permitir que o playbook continue mesmo se a atualização falhar
      tags: update

    - name: Verificar resultado da atualização (se houver redirecionamento)
      debug:
        msg: "Redirecionamento após atualização (esperado). Status: {{ update_result.status }}. Location: {{ update_result.location }}"
      when: update_result.status == 302
      tags: update

    - name: Verificar falha na atualização (se status não for 302)
      debug:
        msg: "Falha na atualização. Status: {{ update_result.status }}. Resposta: {{ update_result.content | default('N/A') }}"
      when: update_result.status != 302 and update_result.failed
      tags: update

    - name: 3. Excluir uma Movimentação (DELETE)
      # Certifique-se de que 'movimentacao_id_para_operacoes' seja um ID válido no seu DB.
      # CUIDADO: Esta operação é irreversível.
      ansible.builtin.uri:
        url: "{{ api_base_url }}/movimentacoes/{{ movimentacao_id_para_operacoes }}"
        method: DELETE
        status_code: 200 # Espera um status OK para sucesso
      register: delete_result
      ignore_errors: yes # Para permitir que o playbook continue
      tags: delete

    - name: Verificar resultado da exclusão
      debug:
        msg: "Exclusão da movimentação {{ movimentacao_id_para_operacoes }} Status: {{ delete_result.status }}. Resposta: {{ delete_result.json | default(delete_result.content) }}"
      tags: delete
````

### Como Executar o Playbook Ansible

1.  **Salve o Playbook**: Copie o conteúdo acima e salve-o como `movimentacoes_api.yml` no seu diretório de trabalho Ansible.

2.  **Verifique/Instale Ansible**: Se você não tem Ansible instalado, siga as instruções em [docs.ansible.com/ansible/latest/installation\_guide/index.html](https://docs.ansible.com/ansible/latest/installation_guide/index.html).

3.  **Inicie seu Servidor Go**: Certifique-se de que sua aplicação Go esteja rodando em `http://localhost:8080`. Se você mudou a porta, ajuste a variável `api_base_url` no playbook.

4.  **Encontre um ID Existente (para Update/Delete)**:

      * Acesse `http://localhost:8080` no seu navegador para ver a tabela de movimentações e obter o `ID` de uma que você deseja atualizar ou excluir.

      * **ATENÇÃO**: Altere a variável `movimentacao_id_para_operacoes` no playbook para um ID real e válido antes de executar as tarefas de `update` ou `delete`. **Cuidado ao executar a tarefa de exclusão, pois é irreversível.**

5.  **Execute o Playbook**:

      * **Para executar todas as tarefas:**

        ```bash
        ansible-playbook movimentacoes_api.yml
        ```

      * **Para executar apenas a tarefa de criação:**

        ```bash
        ansible-playbook movimentacoes_api.yml --tags create
        ```

      * **Para executar apenas a tarefa de atualização:**

        ```bash
        ansible-playbook movimentacoes_api.yml --tags update
        ```

      * **Para executar apenas a tarefa de exclusão:**

        ```bash
        ansible-playbook movimentacoes_api.yml --tags delete
        ```

Ao executar, você verá a saída do Ansible indicando o status de cada requisição. O módulo `debug` ajudará a inspecionar as respostas da API.

