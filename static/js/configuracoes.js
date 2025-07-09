document.addEventListener('DOMContentLoaded', () => {
    // Lógica existente para o Dark Mode
    const darkModeToggle = document.getElementById('dark-mode-toggle');
    if (darkModeToggle) {
        darkModeToggle.addEventListener('change', async () => {
            const isEnabled = darkModeToggle.checked;
            document.documentElement.classList.toggle('dark', isEnabled);
            try {
                await fetch('/api/user/settings', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ dark_mode: isEnabled }),
                });
            } catch (error) {
                console.error('Erro de rede ao salvar preferência de tema:', error);
                alert('Erro de conexão. Não foi possível salvar sua preferência.');
                document.documentElement.classList.toggle('dark', !isEnabled);
                darkModeToggle.checked = !isEnabled;
            }
        });
    }

    // Lógica existente para o Formulário de Perfil
    const profileForm = document.getElementById('profile-form');
    const saveProfileButton = document.getElementById('save-profile-button');
    if (profileForm && saveProfileButton) {
        profileForm.addEventListener('submit', async (event) => {
            event.preventDefault();
            const originalButtonText = saveProfileButton.textContent;
            saveProfileButton.textContent = 'Salvando...';
            saveProfileButton.disabled = true;
            const formData = new FormData(profileForm);
            const payload = {
                date_of_birth: formData.get('date_of_birth'),
                gender: formData.get('gender'),
                marital_status: formData.get('marital_status'),
                children_count: parseInt(formData.get('children_count'), 10) || 0,
                country: formData.get('country'),
                state: formData.get('state'),
                city: formData.get('city'),
            };
            try {
                const response = await fetch('/api/user/profile', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                const result = await response.json();
                if (response.ok) {
                    alert(result.message);
                } else {
                    alert('Erro: ' + result.error);
                }
            } catch (error) {
                alert('Erro de conexão. Não foi possível salvar seu perfil.');
            } finally {
                saveProfileButton.textContent = originalButtonText;
                saveProfileButton.disabled = false;
            }
        });
    }

    // --- NOVA LÓGICA PARA O FORMULÁRIO DE SENHA ---
    const passwordForm = document.getElementById('password-form');
    const savePasswordButton = document.getElementById('save-password-button');

    if (passwordForm && savePasswordButton) {
        passwordForm.addEventListener('submit', async (event) => {
            event.preventDefault();

            const newPassword = document.getElementById('new_password').value;
            const confirmNewPassword = document.getElementById('confirm_new_password').value;

            // Validação do lado do cliente
            if (newPassword.length < 6) {
                alert('A nova senha deve ter no mínimo 6 caracteres.');
                return;
            }
            if (newPassword !== confirmNewPassword) {
                alert('A nova senha e a confirmação não correspondem.');
                return;
            }

            const originalButtonText = savePasswordButton.textContent;
            savePasswordButton.textContent = 'Alterando...';
            savePasswordButton.disabled = true;

            const formData = new FormData(passwordForm);
            const payload = {
                current_password: formData.get('current_password'),
                new_password: newPassword,
                confirm_new_password: confirmNewPassword,
            };

            try {
                const response = await fetch('/api/user/password', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                const result = await response.json();

                if (response.ok) {
                    alert(result.message); // Exibe "Senha alterada com sucesso!"
                    passwordForm.reset(); // Limpa o formulário
                } else {
                    alert('Erro: ' + result.error);
                }

            } catch (error) {
                console.error('Erro de rede ao alterar senha:', error);
                alert('Erro de conexão. Não foi possível alterar sua senha.');
            } finally {
                savePasswordButton.textContent = originalButtonText;
                savePasswordButton.disabled = false;
            }
        });
    }
});
