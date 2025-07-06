// static/js/common.js

/**
 * Funções compartilhadas entre as páginas de Transações e Relatório
 */

/**
 * Atualiza dinamicamente o link do botão da API com base nos filtros de um formulário.
 * @param {string} formId - O ID do formulário de filtro.
 * @param {string} apiLinkId - O ID do link da API a ser atualizado.
 */
function updateApiLink(formId, apiLinkId) {
    const filterForm = document.getElementById(formId);
    const apiLink = document.getElementById(apiLinkId);
    if (!apiLink || !filterForm) return;

    const formData = new FormData(filterForm);
    const params = new URLSearchParams();

    for (const [key, value] of formData.entries()) {
        if (value) {
            params.append(key, value);
        }
    }

    // A página de transações tem esse filtro, a de relatório não.
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

    // A API de movimentações é o destino padrão
    apiLink.href = '/api/movimentacoes?' + finalParams.toString();
}

/**
 * Configura um dropdown com checkboxes múltiplos.
 * @param {string} displayId - O ID do elemento que exibe a seleção.
 * @param {string} optionsId - O ID do contêiner das opções.
 * @param {string} checkboxClass - A classe dos checkboxes.
 * @param {function} onChangeCallback - Uma função a ser chamada quando uma seleção muda.
 */
function setupMultiSelectDropdown(displayId, optionsId, checkboxClass, onChangeCallback) {
    const selectDisplay = document.getElementById(displayId);
    const selectOptions = document.getElementById(optionsId);
    if (!selectDisplay || !selectOptions) return;

    const checkboxes = selectOptions.querySelectorAll('.' + checkboxClass);

    function updateDisplay() {
        const currentSelectedValues = Array.from(checkboxes)
            .filter(cb => cb.checked)
            .map(cb => cb.value);
            
        if (currentSelectedValues.length === 0) {
            selectDisplay.textContent = "Todas as " + (checkboxClass.includes("category") ? "Categorias" : "Contas");
        } else if (currentSelectedValues.length === 1) {
            selectDisplay.textContent = currentSelectedValues[0];
        } else {
            selectDisplay.textContent = `${currentSelectedValues.length} selecionadas`;
        }
        
        if (onChangeCallback) {
            onChangeCallback();
        }
    }

    updateDisplay();
    selectDisplay.addEventListener('click', e => { 
        e.stopPropagation(); 
        selectOptions.classList.toggle('select-hide'); 
    });

    document.addEventListener('click', () => selectOptions.classList.add('select-hide'));
    
    checkboxes.forEach(checkbox => checkbox.addEventListener('change', updateDisplay));
}