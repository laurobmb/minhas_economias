// Este arquivo contém toda a lógica da página /relatorio

document.addEventListener('DOMContentLoaded', () => {
    // Pega os dados que foram passados pelo template Go
    const goData = window.reportPageData || {};

    const reportFilterForm = document.getElementById('reportFilterForm');
    const categoryTransactionsSection = document.getElementById('category-transactions-section');
    const selectedCategoryNameSpan = document.getElementById('selected-category-name');
    const categoryTransactionsTbody = document.getElementById('category-transactions-tbody');
    const noTransactionsMessage = document.getElementById('no-transactions-message');
    const savePdfButton = document.getElementById('save-pdf-button');
    const ctx = document.getElementById('expensesPieChart');
    
    const searchDescricaoInput = document.getElementById('search_descricao');
    
    const categoryCheckboxes = document.querySelectorAll('#category-select-options .custom-checkbox');
    const accountCheckboxes = document.querySelectorAll('#account-select-options .custom-checkbox');
    const startDateInput = document.getElementById('start_date');
    const endDateInput = document.getElementById('end_date');
    const consolidatedFilterSelect = document.getElementById('consolidated_filter');

    let expensesPieChartInstance;

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
        }
        updateDisplay();
        selectDisplay.addEventListener('click', e => { e.stopPropagation(); selectOptions.classList.toggle('select-hide'); });
        document.addEventListener('click', () => selectOptions.classList.add('select-hide'));
        checkboxes.forEach(checkbox => checkbox.addEventListener('change', updateDisplay));
    }

    setupMultiSelectDropdown('category-select-display', 'category-select-options', 'custom-checkbox');
    setupMultiSelectDropdown('account-select-display', 'account-select-options', 'custom-checkbox');

    const reportData = goData.reportData || [];

    if (ctx && reportData && reportData.length > 0) {
        const labels = reportData.map(item => item.categoria);
        const data = reportData.map(item => Math.abs(item.total));
        const backgroundColors = data.map((_, index) => `hsl(${(index * 137.508) % 360}, 70%, 60%)`);

        expensesPieChartInstance = new Chart(ctx, {
            type: 'pie',
            data: { labels, datasets: [{ data, backgroundColor: backgroundColors, borderWidth: 1 }] },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { position: 'top' },
                    title: { display: true, text: 'Despesas por Categoria', font: { size: 18 } },
                    tooltip: { callbacks: { label: c => `${c.label || ''}: R$ ${c.parsed.toFixed(2).replace('.', ',')}` } }
                },
                onClick: async (event, elements) => {
                    if (elements.length > 0) {
                        const categoryName = labels[elements[0].index];
                        selectedCategoryNameSpan.textContent = categoryName;

                        const params = new URLSearchParams(new FormData(reportFilterForm));
                        params.set('category', categoryName); 
                        
                        const apiUrl = `/relatorio/transactions?${params.toString()}`;

                        try {
                            const response = await fetch(apiUrl);
                            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
                            const transactions = await response.json();
                            
                            categoryTransactionsTbody.innerHTML = '';
                            if (transactions.length > 0) {
                                transactions.forEach(tx => {
                                    const row = document.createElement('tr');
                                    const valorFormatado = tx.valor.toFixed(2).replace('.', ',');
                                    const consolidadoTexto = tx.consolidado ? 'Sim' : 'Não';
                                    row.innerHTML = `<td>${tx.id}</td><td>${tx.data_ocorrencia}</td><td>${tx.descricao}</td><td class="text-right">R$ ${valorFormatado}</td><td>${tx.conta}</td><td>${consolidadoTexto}</td>`;
                                    categoryTransactionsTbody.appendChild(row);
                                });
                                noTransactionsMessage.classList.add('select-hide');
                            } else {
                                noTransactionsMessage.textContent = 'Nenhuma transação encontrada para esta categoria com os filtros aplicados.';
                                noTransactionsMessage.classList.remove('select-hide');
                            }
                            categoryTransactionsSection.classList.remove('select-hide');
                        } catch (error) {
                            noTransactionsMessage.textContent = 'Erro ao carregar transações.';
                            noTransactionsMessage.classList.remove('select-hide');
                        }
                    }
                }
            }
        });
    }
    
    if (savePdfButton) {
        savePdfButton.addEventListener('click', async () => {
            if (!expensesPieChartInstance) { alert("Não há gráfico para salvar."); return; }
            savePdfButton.textContent = 'Gerando...';
            savePdfButton.disabled = true;

            try {
                const payload = {
                    search_descricao: searchDescricaoInput.value,
                    start_date: startDateInput.value,
                    end_date: endDateInput.value,
                    categories: Array.from(categoryCheckboxes).filter(cb => cb.checked).map(cb => cb.value),
                    accounts: Array.from(accountCheckboxes).filter(cb => cb.checked).map(cb => cb.value),
                    consolidated_filter: consolidatedFilterSelect.value,
                    chartImageBase64: expensesPieChartInstance.toBase64Image()
                };

                const response = await fetch('/relatorio/pdf', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload),
                });

                if (response.ok) {
                    const blob = await response.blob();
                    const url = window.URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `Relatorio-MinhasEconomias-${new Date().toISOString().slice(0,10)}.pdf`;
                    document.body.appendChild(a);
                    a.click();
                    window.URL.revokeObjectURL(url);
                    a.remove();
                } else {
                    const errorData = await response.json();
                    alert(`Erro ao gerar PDF: ${errorData.error}`);
                }
            } catch (error) {
                alert('Erro de comunicação ao gerar o PDF.');
            } finally {
                savePdfButton.textContent = 'Salvar em PDF';
                savePdfButton.disabled = false;
            }
        });
    }
});