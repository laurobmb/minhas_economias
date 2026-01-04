document.addEventListener('DOMContentLoaded', () => {
    const goData = window.reportPageData || {};
    const reportFilterForm = document.getElementById('reportFilterForm');
    const apiLinkId = 'apiLink';
    const ctx = document.getElementById('expensesPieChart');
    let expensesPieChartInstance;
    const reportData = goData.reportData || [];

    // --- Lógica para atualizar o link da API ---
    if (reportFilterForm) {
        const callback = () => updateApiLink('reportFilterForm', apiLinkId);
        setupMultiSelectDropdown('category-select-display', 'category-select-options', 'custom-checkbox', callback);
        setupMultiSelectDropdown('account-select-display', 'account-select-options', 'custom-checkbox', callback);
        
        reportFilterForm.querySelectorAll('input, select').forEach(input => {
            if (input.type !== 'checkbox') {
                input.addEventListener('change', callback);
            }
        });
        
        updateApiLink('reportFilterForm', apiLinkId); // Chamada inicial
    }

    // --- LÓGICA DO GRÁFICO ATUALIZADA PARA DARK MODE ---
    if (ctx && reportData && reportData.length > 0) {
        const labels = reportData.map(item => item.categoria);
        const data = reportData.map(item => Math.abs(item.total));
        const backgroundColors = data.map((_, index) => `hsl(${(index * 137.508) % 360}, 70%, 60%)`);

        // Verifica se o modo escuro está ativo no HTML
        const isDarkMode = document.documentElement.classList.contains('dark');
        const FONT_COLOR = isDarkMode ? '#e2e8f0' : '#475569'; // Cinza claro para dark, cinza escuro para light
        const GRID_COLOR = isDarkMode ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)';

        expensesPieChartInstance = new Chart(ctx, {
            type: 'pie',
            data: { 
                labels, 
                datasets: [{ 
                    data, 
                    backgroundColor: backgroundColors, 
                    borderColor: isDarkMode ? '#334155' : '#ffffff', // Borda mais escura no dark mode
                    borderWidth: 2 
                }] 
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { 
                        position: 'top',
                        labels: {
                            color: FONT_COLOR, // Cor da legenda
                            font: { size: 12 }
                        }
                    },
                    title: { 
                        display: true, 
                        text: 'Despesas por Categoria',
                        color: FONT_COLOR, // Cor do título
                        font: { size: 18 } 
                    },
                    tooltip: {
                        bodyColor: '#e2e8f0',
                        titleColor: '#ffffff',
                        backgroundColor: '#1e293b',
                        callbacks: { 
                            label: c => `${c.label || ''}: R$ ${c.parsed.toFixed(2).replace('.', ',')}` 
                        }
                    }
                },
                onClick: (event, elements) => {
                    if (elements.length > 0) {
                        const categoryName = labels[elements[0].index];
                        fetchTransactionsForCategory(categoryName);
                    }
                }
            }
        });
    }
    
    // --- Lógica de clique e PDF (sem alterações, mas incluída para completude) ---
    const categoryTransactionsSection = document.getElementById('category-transactions-section');
    const selectedCategoryNameSpan = document.getElementById('selected-category-name');
    const categoryTransactionsTbody = document.getElementById('category-transactions-tbody');
    const noTransactionsMessage = document.getElementById('no-transactions-message');
    const savePdfButton = document.getElementById('save-pdf-button');

    async function fetchTransactionsForCategory(categoryName) {
        if (!categoryTransactionsSection || !selectedCategoryNameSpan || !categoryTransactionsTbody || !noTransactionsMessage) return;

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
                noTransactionsMessage.classList.add('select-hide');
                transactions.forEach(tx => {
                    const row = document.createElement('tr');
                    const valorFormatado = tx.valor.toFixed(2).replace('.', ',');
                    const consolidadoTexto = tx.consolidado ? 'Sim' : 'Não';
                    row.innerHTML = `<td>${tx.id}</td><td>${tx.data_ocorrencia}</td><td>${tx.descricao}</td><td class="text-right">R$ ${valorFormatado}</td><td>${tx.conta}</td><td>${consolidadoTexto}</td>`;
                    categoryTransactionsTbody.appendChild(row);
                });
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

    if (savePdfButton) {
        savePdfButton.addEventListener('click', async () => {
            if (!expensesPieChartInstance) { alert("Não há gráfico para salvar."); return; }
            savePdfButton.textContent = 'Gerando...';
            savePdfButton.disabled = true;
            try {
                const payload = {
                    search_descricao: reportFilterForm.search_descricao.value,
                    start_date: reportFilterForm.start_date.value,
                    end_date: reportFilterForm.end_date.value,

                    categories: Array.from(reportFilterForm.querySelectorAll('input[name="category"]:checked')).map(cb => cb.value).filter(val => val.trim() !== ""),

                    accounts: Array.from(reportFilterForm.querySelectorAll('input[name="account"]:checked')).map(cb => cb.value),
                    consolidated_filter: reportFilterForm.consolidated_filter.value,
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