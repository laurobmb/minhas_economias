document.addEventListener('DOMContentLoaded', () => {
    // --- SEÇÃO DE EDIÇÃO DE ATIVOS (EXISTENTE) ---
    const editSection = document.getElementById('edit-investment-section');
    const editForm = document.getElementById('edit-investment-form');
    const editFormTitle = document.getElementById('edit-form-title');
    const editTickerOriginalInput = document.getElementById('edit-ticker-original');
    const editTickerDisplayInput = document.getElementById('edit-ticker-display');
    const editQuantityInput = document.getElementById('edit-quantity');
    const editAssetTypeInput = document.getElementById('edit-asset-type');
    const cancelEditButton = document.getElementById('cancel-edit-button');

    function showEditForm(ticker, quantity, assetType) {
        editTickerOriginalInput.value = ticker;
        editTickerDisplayInput.value = ticker;
        editQuantityInput.value = quantity;
        editAssetTypeInput.value = assetType;
        editFormTitle.textContent = `Editar Ativo: ${ticker}`;
        editSection.classList.remove('select-hide');
        editSection.scrollIntoView({ behavior: 'smooth', block: 'start' });
        editQuantityInput.focus();
    }

    function hideEditForm() {
        editSection.classList.add('select-hide');
        editForm.reset();
    }

    document.querySelectorAll('.edit-button').forEach(button => {
        button.addEventListener('click', (event) => {
            const ticker = event.target.dataset.ticker;
            const quantity = event.target.dataset.quantity;
            const assetType = event.target.dataset.type;
            showEditForm(ticker, quantity, assetType);
        });
    });

    if (cancelEditButton) {
        cancelEditButton.addEventListener('click', hideEditForm);
    }

    if (editForm) {
        editForm.addEventListener('submit', async (event) => {
            event.preventDefault();
            const ticker = editTickerOriginalInput.value;
            const assetType = editAssetTypeInput.value;
            const newQuantity = editQuantityInput.value;

            if (!ticker || !assetType || newQuantity === '') {
                alert('Erro: Informações incompletas para a atualização.');
                return;
            }

            const apiUrl = `/investimentos/${assetType}/${ticker}`;

            try {
                const response = await fetch(apiUrl, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ quantidade: parseFloat(newQuantity.replace(',', '.')) })
                });
                const result = await response.json();
                if (response.ok) {
                    alert(result.message);
                    window.location.reload();
                } else {
                    alert(`Erro: ${result.error}`);
                }
            } catch (error) {
                alert('Erro de comunicação ao tentar atualizar o ativo.');
            }
        });
    }

    // --- SEÇÃO DE EXCLUSÃO DE ATIVOS (EXISTENTE) ---
    document.querySelectorAll('.delete-button').forEach(button => {
        button.addEventListener('click', async (event) => {
            const ticker = event.target.dataset.ticker;
            const assetType = event.target.dataset.type;

            if (!confirm(`Tem certeza que deseja excluir o ativo ${ticker}? Esta ação não pode ser desfeita.`)) {
                return;
            }

            const apiUrl = `/investimentos/${assetType}/${ticker}`;

            try {
                const response = await fetch(apiUrl, { method: 'DELETE' });
                const result = await response.json();
                if (response.ok) {
                    alert(result.message);
                    window.location.reload();
                } else {
                    alert(`Erro: ${result.error}`);
                }
            } catch (error) {
                alert('Erro de comunicação ao tentar excluir o ativo.');
            }
        });
    });

    // --- NOVA SEÇÃO: ADIÇÃO DE ATIVOS NACIONAIS ---
    const addNacionalForm = document.getElementById('add-nacional-form');
    if (addNacionalForm) {
        addNacionalForm.addEventListener('submit', async (event) => {
            event.preventDefault();
            const ticker = document.getElementById('add-nacional-ticker').value.toUpperCase();
            const tipo = document.getElementById('add-nacional-tipo').value;
            const quantidade = document.getElementById('add-nacional-quantidade').value;

            if (!ticker || !tipo || !quantidade) {
                alert('Por favor, preencha todos os campos.');
                return;
            }

            try {
                const response = await fetch('/investimentos/nacional', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        ticker: ticker,
                        tipo: tipo,
                        quantidade: parseInt(quantidade, 10)
                    })
                });
                const result = await response.json();
                if (response.ok) {
                    alert(result.message);
                    window.location.reload();
                } else {
                    alert(`Erro: ${result.error}`);
                }
            } catch (error) {
                alert('Erro de comunicação ao tentar adicionar o ativo.');
            }
        });
    }

    // --- NOVA SEÇÃO: ADIÇÃO DE ATIVOS INTERNACIONAIS ---
    const addInternacionalForm = document.getElementById('add-internacional-form');
    if (addInternacionalForm) {
        addInternacionalForm.addEventListener('submit', async (event) => {
            event.preventDefault();
            const ticker = document.getElementById('add-internacional-ticker').value.toUpperCase();
            const descricao = document.getElementById('add-internacional-descricao').value;
            const quantidade = document.getElementById('add-internacional-quantidade').value;

            if (!ticker || !descricao || !quantidade) {
                alert('Por favor, preencha todos os campos.');
                return;
            }

            try {
                const response = await fetch('/investimentos/internacional', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        ticker: ticker,
                        descricao: descricao,
                        quantidade: parseFloat(quantidade.replace(',', '.'))
                    })
                });
                const result = await response.json();
                if (response.ok) {
                    alert(result.message);
                    window.location.reload();
                } else {
                    alert(`Erro: ${result.error}`);
                }
            } catch (error) {
                alert('Erro de comunicação ao tentar adicionar o ativo.');
            }
        });
    }
});
