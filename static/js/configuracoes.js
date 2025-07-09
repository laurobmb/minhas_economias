document.addEventListener('DOMContentLoaded', () => {
    const darkModeToggle = document.getElementById('dark-mode-toggle');

    if (darkModeToggle) {
        darkModeToggle.addEventListener('change', async () => {
            const isEnabled = darkModeToggle.checked;

            // 1. Atualiza a UI imediatamente (otimismo)
            document.documentElement.classList.toggle('dark', isEnabled);

            // 2. Envia a atualização para o backend
            try {
                const response = await fetch('/api/user/settings', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        dark_mode: isEnabled
                    }),
                });

                if (!response.ok) {
                    // Se falhar, reverte a UI e avisa o usuário
                    console.error('Falha ao salvar a preferência de tema.');
                    alert('Não foi possível salvar sua preferência. Tente novamente.');
                    document.documentElement.classList.toggle('dark', !isEnabled);
                    darkModeToggle.checked = !isEnabled;
                }
                
                // Sucesso, não precisa fazer nada
                
            } catch (error) {
                console.error('Erro de rede ao salvar preferência de tema:', error);
                alert('Erro de conexão. Não foi possível salvar sua preferência.');
                document.documentElement.classList.toggle('dark', !isEnabled);
                darkModeToggle.checked = !isEnabled;
            }
        });
    }
});