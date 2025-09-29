document.addEventListener('DOMContentLoaded', () => {
    const chatForm = document.getElementById('chat-form');
    const chatInput = document.getElementById('chat-input');
    const sendButton = document.getElementById('send-button');
    const chatBox = document.getElementById('chat-box');
    const isDarkMode = document.documentElement.classList.contains('dark');

    // Função para adicionar uma mensagem à caixa de chat
    function addMessage(text, sender) {
        const messageDiv = document.createElement('div');
        messageDiv.classList.add('chat-message', `${sender}-message`);
        
        if (sender === 'ai') {
            messageDiv.innerHTML = marked.parse(text);
            if (isDarkMode) {
                messageDiv.classList.add('dark');
            }
        } else {
            messageDiv.textContent = text;
        }
        
        chatBox.appendChild(messageDiv);
        chatBox.scrollTop = chatBox.scrollHeight;
    }

    // Função para mostrar o feedback "pensando..."
    function showLoadingIndicator() {
        const loadingDiv = document.createElement('div');
        loadingDiv.id = 'loading-indicator';
        loadingDiv.classList.add('chat-message', 'ai-message');
        if (isDarkMode) loadingDiv.classList.add('dark');
        loadingDiv.innerHTML = '<span class="italic text-gray-500 dark:text-gray-400">Analisando seus dados...</span>';
        chatBox.appendChild(loadingDiv);
        chatBox.scrollTop = chatBox.scrollHeight;
    }

    // Função para remover o "pensando..."
    function hideLoadingIndicator() {
        const indicator = document.getElementById('loading-indicator');
        if (indicator) {
            indicator.remove();
        }
    }

    // Função para enviar a mensagem para o backend
    async function sendMessage(question) {
        addMessage(question, 'user');
        chatInput.value = '';
        sendButton.disabled = true;
        sendButton.textContent = 'Aguarde...';

        showLoadingIndicator();

        try {
            const response = await fetch('/api/analise/chat', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ question: question })
            });
            
            hideLoadingIndicator();

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Ocorreu um erro no servidor.');
            }

            const data = await response.json();
            addMessage(data.analysis, 'ai');

        } catch (error) {
            hideLoadingIndicator();
            addMessage(`Erro: ${error.message}`, 'ai');
        } finally {
            sendButton.disabled = false;
            sendButton.textContent = 'Enviar';
            chatInput.focus();
        }
    }

    // Event listener para o formulário
    chatForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const question = chatInput.value.trim();
        if (question) {
            sendMessage(question);
        }
    });

    // --- NOVA LÓGICA DE INICIALIZAÇÃO ---
    // Renderiza o histórico que veio do HTML e decide se inicia um novo chat.
    function initializeChat() {
        const historyMessages = chatBox.querySelectorAll('.chat-message');
        
        // Renderiza o conteúdo markdown do histórico
        historyMessages.forEach(msgDiv => {
            const contentDiv = msgDiv.querySelector('.hidden-content');
            if (contentDiv) {
                const rawContent = contentDiv.textContent;
                if (msgDiv.dataset.role === 'ai') {
                    msgDiv.innerHTML = marked.parse(rawContent);
                } else {
                    msgDiv.textContent = rawContent;
                }
            }
        });

        // Se o chat estiver vazio (nenhum histórico), envia a mensagem inicial.
        if (historyMessages.length === 0) {
            sendMessage("Faça um resumo financeiro do meu último mês e do mês atual, destacando pontos positivos e áreas para melhoria.");
        } else {
            // Se já tem histórico, apenas rola para o final.
            chatBox.scrollTop = chatBox.scrollHeight;
        }
    }

    initializeChat();
});