# Exemplos de Comandos Curl para a API

Aqui estão exemplos de como você pode usar `curl` para interagir com os endpoints da sua API.

### **1. Criar (POST) uma Nova Movimentação**

Este comando `POST` envia os dados de um novo registro para o endpoint `/movimentacoes`.

```

curl -X POST  
http://localhost:8080/movimentacoes  
\-H 'Content-Type: application/x-www-form-urlencoded'  
\--data-urlencode 'data\_ocorrencia=2025-06-29'  
\--data-urlencode 'descricao=Compra de Alimentos'  
\--data-urlencode 'valor=120.50'  
\--data-urlencode 'categoria=Alimentacao'  
\--data-urlencode 'conta=Cartao Credito'  
\--data-urlencode 'consolidado=on'

```

* `-X POST`: Especifica o método HTTP como POST.

* `http://localhost:8080/movimentacoes`: O URL do endpoint para adicionar movimentações.

* `-H 'Content-Type: application/x-www-form-urlencoded'`: Informa ao servidor que o corpo da requisição está no formato URL-encoded (padrão de formulários HTML).

* `--data-urlencode 'campo=valor'`: Envia os pares chave-valor do formulário. Note que para `consolidado`, `on` significa `true` para o backend, e se omitido, será `false`.

### **2. Atualizar (POST) uma Movimentação Existente**

Para atualizar uma movimentação, você enviará um `POST` para a rota de atualização, incluindo o `ID` da movimentação na URL. Você deve enviar todos os campos da movimentação, mesmo os que não foram alterados, pois a rota `updateMovimentacao` espera todos eles.

**Primeiro, descubra o ID de uma movimentação para editar.** Você pode fazer isso acessando `http://localhost:8080` ou `http://localhost:8080/api/movimentacoes` no seu navegador. Vamos assumir que você quer editar a movimentação com `ID = 1`.

```

curl -X POST  
http://localhost:8080/movimentacoes/update/1  
\-H 'Content-Type: application/x-www-form-urlencoded'  
\--data-urlencode 'data\_ocorrencia=2025-06-28'  
\--data-urlencode 'descricao=Aluguel Mensal Atualizado'  
\--data-urlencode 'valor=-1600.00'  
\--data-urlencode 'categoria=Moradia'  
\--data-urlencode 'conta=Conta Corrente'  
\--data-urlencode 'consolidado=on'

```

* `http://localhost:8080/movimentacoes/update/1`: O URL do endpoint de atualização, com o ID da movimentação (neste caso, `1`).

* Os parâmetros `--data-urlencode` contêm os novos valores para a movimentação.

### **3. Excluir (DELETE) uma Movimentação**

Para excluir uma movimentação, você envia uma requisição `DELETE` para o endpoint `/movimentacoes/:id`, especificando o `ID` da movimentação a ser excluída.

**Novamente, descubra o ID de uma movimentação para excluir.** Vamos assumir que você quer excluir a movimentação com `ID = 2`.

```

curl -X DELETE  
http://localhost:8080/movimentacoes/2

```

* `-X DELETE`: Especifica o método HTTP como DELETE.

* `http://localhost:8080/movimentacoes/2`: O URL do endpoint de exclusão, com o ID da movimentação (neste caso, `2`).

**Após executar qualquer um desses comandos, você pode recarregar a página `http://localhost:8080` no seu navegador para ver as alterações refletidas na tabela.**
