// static/js/transacoes.js

document.addEventListener('DOMContentLoaded', () => {
    const goData = window.minhasEconomiasData || {};
    const filterFormId = 'filterForm';
    const apiLinkId = 'apiLink';
    const filterForm = document.getElementById(filterFormId);
    const exportButtonId = 'export-csv-button';

    // --- Lógica do Formulário de Adicionar/Editar ---
    const addEditForm = document.getElementById('add-edit-form');
    const addEditFormTitle = document.getElementById('add-edit-form-title');
    const movementIdInput = document.getElementById('movement-id-input');
    const submitMovementButton = document.getElementById('submit-movement-button');
    const cancelEditButton = document.getElementById('cancel-edit-button');

    // Inputs
    const newDataOcorrenciaInput = document.getElementById('new_data_ocorrencia');
    const newDescricaoInput = document.getElementById('new_descricao');
    const newValorInput = document.getElementById('new_valor');
    const newCategoriaInput = document.getElementById('new_categoria');
    const newContaInput = document.getElementById('new_conta');
    const newConsolidadoCheckbox = document.getElementById('new_consolidado');
    const newContaOrigemInput = document.getElementById('new_conta_origem');
    const newContaDestinoInput = document.getElementById('new_conta_destino');

    // Seletores de Tipo
    const tipoMovimentacaoDisplay = document.getElementById('tipo-movimentacao-display');
    const tipoMovimentacaoOptions = document.getElementById('tipo-movimentacao-options');
    const tipoReceitaRadio = document.getElementById('tipo_receita');
    const tipoDespesaRadio = document.getElementById('tipo_despesa');
    const tipoTransferenciaRadio = document.getElementById('tipo_transferencia');

    // Grupos de Campos
    const groupCategoria = document.getElementById('group-categoria');
    const groupConta = document.getElementById('group-conta');
    const groupContaOrigem = document.getElementById('group-conta-origem');
    const groupContaDestino = document.getElementById('group-conta-destino');
    const groupConsolidado = document.getElementById('group-consolidado');

    function adjustValorSign() {
        if (!newValorInput) return;
        let rawValue = newValorInput.value.trim().replace(/\./g, '').replace(',', '.');
        if (rawValue === '' || isNaN(rawValue)) return;
        let currentValue = parseFloat(rawValue);

        if (tipoTransferenciaRadio.checked) {
            newValorInput.value = Math.abs(currentValue).toFixed(2).replace('.', ',');
            return;
        }

        let newValue = tipoDespesaRadio.checked ? -Math.abs(currentValue) : Math.abs(currentValue);
        newValorInput.value = newValue.toFixed(2).replace('.', ',');
    }

    function updateTipoMovimentacaoDisplay() {
        if (!tipoMovimentacaoDisplay) return;
        if (tipoTransferenciaRadio.checked) {
            tipoMovimentacaoDisplay.textContent = "Transferência";
        } else {
            tipoMovimentacaoDisplay.textContent = tipoDespesaRadio.checked ? "Despesa" : "Receita";
        }
    }

    function toggleFormFields() {
        const isTransfer = tipoTransferenciaRadio.checked;

        groupCategoria.classList.toggle('select-hide', isTransfer);
        groupConta.classList.toggle('select-hide', isTransfer);
        groupConsolidado.classList.toggle('select-hide', isTransfer);

        groupContaOrigem.classList.toggle('select-hide', !isTransfer);
        groupContaDestino.classList.toggle('select-hide', !isTransfer);

        // Atualiza os campos obrigatórios
        newContaInput.required = !isTransfer;
        newContaOrigemInput.required = isTransfer;
        newContaDestinoInput.required = isTransfer;
        newCategoriaInput.required = false; // Categoria nunca é obrigatória

        // Atualiza a action do formulário
        addEditForm.action = isTransfer ? '/movimentacoes/transferencia' : '/movimentacoes';
        adjustValorSign();
    }

    if (tipoMovimentacaoDisplay) {
        tipoMovimentacaoDisplay.addEventListener('click', e => {
            e.stopPropagation();
            tipoMovimentacaoOptions.classList.toggle('select-hide');
        });
        document.addEventListener('click', () => tipoMovimentacaoOptions.classList.add('select-hide'));

        [tipoReceitaRadio, tipoDespesaRadio, tipoTransferenciaRadio].forEach(radio => {
            radio.addEventListener('change', () => {
                updateTipoMovimentacaoDisplay();
                toggleFormFields();
                tipoMovimentacaoOptions.classList.add('select-hide');
            });
        });
    }

    if (newValorInput) newValorInput.addEventListener('change', adjustValorSign);

    // --- Lógica de preenchimento e validação ---
    const categoriesFromGo = goData.categories || [];
    const accountsFromGo = goData.accounts || [];
    function populateDatalist(datalistElement, dataArray) {
        if (!datalistElement || !dataArray) return;
        datalistElement.innerHTML = '';
        dataArray.forEach(item => {
            const option = document.createElement('option');
            option.value = item;
            datalistElement.appendChild(option);
        });
    }
    populateDatalist(document.getElementById('category-suggestions'), categoriesFromGo);
    populateDatalist(document.getElementById('account-suggestions'), accountsFromGo);

    if (newValorInput) {
        newValorInput.addEventListener('input', (event) => {
            let value = event.target.value;
            value = value.replace(/[^\d.,-]/g, '');
            if ((value.match(/-/g) || []).length > 1) value = '-' + value.replace(/-/g, '');
            if (value.lastIndexOf('-') > 0) value = value.replace(/-/g, '');
            event.target.value = value;
        });
    }

    // --- Reset e modo de edição ---
    function resetAddEditForm() {
        addEditForm.reset();
        addEditFormTitle.textContent = "Adicionar Nova Movimentação";
        submitMovementButton.textContent = "Adicionar Movimentação";
        movementIdInput.value = "";
        cancelEditButton.classList.add('select-hide');
        tipoDespesaRadio.checked = true;
        updateTipoMovimentacaoDisplay();
        toggleFormFields();
        if (newDataOcorrenciaInput) newDataOcorrenciaInput.value = goData.currentDate;
    }

    if (cancelEditButton) cancelEditButton.addEventListener('click', resetAddEditForm);

    document.querySelectorAll('.edit-button').forEach(button => {
        button.addEventListener('click', (event) => {
            const row = event.target.closest('tr');
            if (row.dataset.categoria === 'Transferência') {
                alert('Não é possível editar uma transferência diretamente. Por favor, exclua e crie novamente se necessário.');
                return;
            }
            movementIdInput.value = row.dataset.id;
            newDataOcorrenciaInput.value = row.dataset.data;
            newDescricaoInput.value = row.dataset.descricao;
            newValorInput.value = parseFloat(row.dataset.valor).toFixed(2).replace('.', ',');
            newCategoriaInput.value = row.dataset.categoria;
            newContaInput.value = row.dataset.conta;
            newConsolidadoCheckbox.checked = (row.dataset.consolidado === 'true');
            if (row.dataset.tipo === 'receita') tipoReceitaRadio.checked = true;
            else tipoDespesaRadio.checked = true;
            updateTipoMovimentacaoDisplay();
            toggleFormFields();
            addEditFormTitle.textContent = `Editar Movimentação (ID: ${row.dataset.id})`;
            submitMovementButton.textContent = "Salvar Alterações";
            addEditForm.action = `/movimentacoes/update/${row.dataset.id}`;
            cancelEditButton.classList.remove('select-hide');
            addEditForm.scrollIntoView({ behavior: 'smooth', block: 'start' });
        });
    });

    document.querySelectorAll('.delete-button').forEach(button => {
        button.addEventListener('click', async (event) => {
            const id = event.target.dataset.id;
            if (confirm(`Tem certeza que deseja excluir a movimentação ${id}?`)) {
                try {
                    const response = await fetch(`/movimentacoes/${id}`, { method: 'DELETE' });
                    if (response.ok) {
                        alert('Movimentação excluída!');
                        window.location.reload();
                    } else {
                        const err = await response.json();
                        alert(`Erro: ${err.error}`);
                    }
                } catch (error) {
                    alert('Erro de comunicação.');
                }
            }
        });
    });

    // --- Lógica dos Filtros (incluindo link de exportação) ---
    function updateExportLink() {
        const exportButton = document.getElementById(exportButtonId);
        if (!filterForm || !exportButton) return;
        const params = new URLSearchParams(new FormData(filterForm));
        const activeValueFilterButton = document.querySelector('.value-filter-button.active');
        if (activeValueFilterButton && activeValueFilterButton.dataset.valueFilter) {
            params.set('value_filter', activeValueFilterButton.dataset.valueFilter);
        } else {
            params.delete('value_filter');
        }
        const finalParams = new URLSearchParams();
        for (const [key, value] of params.entries()) {
            if (value) finalParams.append(key, value);
        }
        exportButton.href = `/export/csv?${finalParams.toString()}`;
    }

    if (filterForm) {
        const callback = () => {
            updateApiLink(filterFormId, apiLinkId);
            updateExportLink();
        };
        setupMultiSelectDropdown('category-select-display', 'category-select-options', 'custom-checkbox', callback);
        setupMultiSelectDropdown('account-select-display', 'account-select-options', 'custom-checkbox', callback);
        filterForm.querySelectorAll('input, select').forEach(input => {
            if (input.type !== 'checkbox') input.addEventListener('change', callback);
        });
        updateApiLink(filterFormId, apiLinkId);
        updateExportLink();
    }

    document.querySelectorAll('.value-filter-button').forEach(button => {
        button.addEventListener('click', () => {
            const valueFilter = button.dataset.valueFilter;
            const currentParams = new URLSearchParams(new FormData(filterForm));
            if (valueFilter) currentParams.set('value_filter', valueFilter);
            else currentParams.delete('value_filter');
            window.location.href = `/transacoes?${currentParams.toString()}`;
        });
    });

    // Inicializa o formulário no estado correto
    toggleFormFields();
});