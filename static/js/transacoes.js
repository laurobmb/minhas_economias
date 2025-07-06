// Este arquivo contém toda a lógica da página /transacoes

document.addEventListener('DOMContentLoaded', () => {
    // Pega os dados que foram passados pelo template Go
    const goData = window.minhasEconomiasData || {};

    const filterForm = document.getElementById('filterForm');

    function updateApiLink() {
        const apiLink = document.getElementById('apiLink');
        if (!apiLink || !filterForm) return;
        const formData = new FormData(filterForm);
        const params = new URLSearchParams();
        for (const [key, value] of formData.entries()) {
            if (value) {
                params.append(key, value);
            }
        }
        const activeValueFilterButton = document.querySelector('.value-filter-button.active');
        if (activeValueFilterButton && activeValueFilterButton.dataset.valueFilter) {
            params.set('value_filter', activeValueFilterButton.dataset.valueFilter);
        } else {
            params.delete('value_filter');
        }
        const finalParams = new URLSearchParams();
        for (const [key, value] of params.entries()) {
            if (value) {
               finalParams.append(key, value)
            }
        }
        apiLink.href = '/api/movimentacoes?' + finalParams.toString();
    }

    function setupMultiSelectDropdown(displayId, optionsId, checkboxClass) {
        const selectDisplay = document.getElementById(displayId);
        const selectOptions = document.getElementById(optionsId);
        if (!selectDisplay || !selectOptions) return;
        const checkboxes = selectOptions.querySelectorAll('.' + checkboxClass);
        function updateDisplay() {
            const currentSelectedValues = Array.from(checkboxes).filter(cb => cb.checked).map(cb => cb.value);
            if (currentSelectedValues.length === 0) {
                selectDisplay.textContent = "Todas as " + (checkboxClass.includes("category") ? "Categorias" : "Contas");
            } else if (currentSelectedValues.length === 1) {
                selectDisplay.textContent = currentSelectedValues[0];
            } else {
                selectDisplay.textContent = `${currentSelectedValues.length} selecionadas`;
            }
            updateApiLink();
        }
        updateDisplay();
        selectDisplay.addEventListener('click', e => { e.stopPropagation(); selectOptions.classList.toggle('select-hide'); });
        document.addEventListener('click', () => selectOptions.classList.add('select-hide'));
        checkboxes.forEach(checkbox => checkbox.addEventListener('change', updateDisplay));
    }
    
    if (filterForm) {
        setupMultiSelectDropdown('category-select-display', 'category-select-options', 'custom-checkbox');
        setupMultiSelectDropdown('account-select-display', 'account-select-options', 'custom-checkbox');
        filterForm.querySelectorAll('input, select').forEach(input => {
            if (input.type !== 'checkbox') {
                 input.addEventListener('change', updateApiLink);
            }
        });
        updateApiLink();
    }

    document.querySelectorAll('.value-filter-button').forEach(button => {
        button.addEventListener('click', () => {
            const valueFilter = button.dataset.valueFilter;
            const currentParams = new URLSearchParams(new FormData(filterForm));
            if (valueFilter) {
                currentParams.set('value_filter', valueFilter);
            } else {
                currentParams.delete('value_filter');
            }
            window.location.href = `/transacoes?${currentParams.toString()}`;
        });
    });

    const addEditForm = document.getElementById('add-edit-form');
    const addEditFormTitle = document.getElementById('add-edit-form-title');
    const movementIdInput = document.getElementById('movement-id-input');
    const newValorInput = document.getElementById('new_valor');
    const newDataOcorrenciaInput = document.getElementById('new_data_ocorrencia');
    const newDescricaoInput = document.getElementById('new_descricao');
    const newCategoriaInput = document.getElementById('new_categoria');
    const newContaInput = document.getElementById('new_conta');
    const newConsolidadoCheckbox = document.getElementById('new_consolidado');
    const submitMovementButton = document.getElementById('submit-movement-button');
    const cancelEditButton = document.getElementById('cancel-edit-button');
    const tipoReceitaRadio = document.getElementById('tipo_receita');
    const tipoDespesaRadio = document.getElementById('tipo_despesa');
    const tipoMovimentacaoDisplay = document.getElementById('tipo-movimentacao-display');
    const tipoMovimentacaoOptions = document.getElementById('tipo-movimentacao-options');

    function adjustValorSign() {
        let rawValue = newValorInput.value.trim().replace(/\./g, '').replace(',', '.');
        if (rawValue === '' || isNaN(rawValue)) return;
        let currentValue = parseFloat(rawValue);
        let newValue = tipoDespesaRadio.checked ? -Math.abs(currentValue) : Math.abs(currentValue);
        newValorInput.value = newValue.toFixed(2).replace('.', ',');
    }
    
    function updateTipoMovimentacaoDisplay() {
        tipoMovimentacaoDisplay.textContent = tipoDespesaRadio.checked ? "Despesa" : "Receita";
    }

    if (tipoMovimentacaoDisplay) {
        tipoMovimentacaoDisplay.addEventListener('click', e => { e.stopPropagation(); tipoMovimentacaoOptions.classList.toggle('select-hide'); });
        document.addEventListener('click', () => tipoMovimentacaoOptions.classList.add('select-hide'));
        [tipoReceitaRadio, tipoDespesaRadio].forEach(radio => radio.addEventListener('change', () => {
            adjustValorSign();
            updateTipoMovimentacaoDisplay();
            tipoMovimentacaoOptions.classList.add('select-hide');
        }));
    }

    if(newValorInput) newValorInput.addEventListener('change', adjustValorSign);
    if(tipoDespesaRadio) { tipoDespesaRadio.checked = true; updateTipoMovimentacaoDisplay(); }

    const categoriesFromGo = goData.categories || [];
    const accountsFromGo = goData.accounts || [];
    function populateDatalist(datalistElement, dataArray) {
        if(!datalistElement || !dataArray) return;
        datalistElement.innerHTML = '';
        dataArray.forEach(item => {
            const option = document.createElement('option');
            option.value = item;
            datalistElement.appendChild(option);
        });
    }
    populateDatalist(document.getElementById('category-suggestions'), categoriesFromGo);
    populateDatalist(document.getElementById('account-suggestions'), accountsFromGo);

    function resetAddEditForm() {
        addEditForm.reset();
        addEditFormTitle.textContent = "Adicionar Nova Movimentação";
        submitMovementButton.textContent = "Adicionar Movimentação";
        addEditForm.action = "/movimentacoes";
        movementIdInput.value = "";
        cancelEditButton.classList.add('select-hide');
        tipoDespesaRadio.checked = true;
        updateTipoMovimentacaoDisplay();
        if (newDataOcorrenciaInput) {
            newDataOcorrenciaInput.value = goData.currentDate;
        }
    }

    if(cancelEditButton) cancelEditButton.addEventListener('click', resetAddEditForm);

    document.querySelectorAll('.delete-button').forEach(button => {
        button.addEventListener('click', async (event) => {
            const id = event.target.dataset.id;
            if (confirm(`Tem certeza que deseja excluir a movimentação ${id}?`)) {
                try {
                    const response = await fetch(`/movimentacoes/${id}`, { method: 'DELETE' });
                    if (response.ok) { alert('Movimentação excluída!'); window.location.reload(); }
                    else { const err = await response.json(); alert(`Erro: ${err.error}`); }
                } catch (error) { alert('Erro de comunicação.'); }
            }
        });
    });

    document.querySelectorAll('.edit-button').forEach(button => {
        button.addEventListener('click', (event) => {
            const row = event.target.closest('tr');
            movementIdInput.value = row.dataset.id;
            newDataOcorrenciaInput.value = row.dataset.data;
            newDescricaoInput.value = row.dataset.descricao;
            newValorInput.value = parseFloat(row.dataset.valor).toFixed(2).replace('.',',');
            newCategoriaInput.value = row.dataset.categoria;
            newContaInput.value = row.dataset.conta;
            newConsolidadoCheckbox.checked = (row.dataset.consolidado === 'true');
            if(row.dataset.tipo === 'receita') tipoReceitaRadio.checked = true; else tipoDespesaRadio.checked = true;
            updateTipoMovimentacaoDisplay();
            addEditFormTitle.textContent = `Editar Movimentação (ID: ${row.dataset.id})`;
            submitMovementButton.textContent = "Salvar Alterações";
            addEditForm.action = `/movimentacoes/update/${row.dataset.id}`;
            cancelEditButton.classList.remove('select-hide');
            addEditForm.scrollIntoView({ behavior: 'smooth', block: 'start' });
        });
    });
});