#!/usr/bin/env python3

import os
import unittest
import sys
import logging
import time
import shutil
import sqlite3
import psycopg2
import csv
from urllib.parse import quote_plus
from datetime import datetime, timezone
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, NoSuchElementException

# --- Configurações ---
logging.basicConfig(format='%(asctime)s: %(name)s: %(levelname)s: %(message)s', level=logging.INFO, datefmt='%H:%M:%S')
logger = logging.getLogger('MinhasEconomiasTest')

BASE_URL = os.getenv('APP_URL', 'http://localhost:8080')
DEFAULT_TIMEOUT = 10
IS_CONTAINER = os.getenv('CONTAINER', 'false').lower() == 'true'
STEP_DELAY = 0

# --- Credenciais de Teste ---
TEST_USER_EMAIL = "lauro@localnet.com"
TEST_USER_PASS = "1q2w3e"

# --- Variáveis de Ambiente do Banco de Dados ---
DB_TYPE = os.getenv('DB_TYPE', 'sqlite3')
DB_NAME = os.getenv('DB_NAME', 'extratos.db')
DB_USER = os.getenv('DB_USER', 'postgres')
DB_PASS = os.getenv('DB_PASS', 'postgres')
DB_HOST = os.getenv('DB_HOST', 'localhost')
DB_PORT = os.getenv('DB_PORT', '5432')


class MinhasEconomiasTest(unittest.TestCase):
    
    @classmethod
    def setUpClass(cls):
        """Configura o ambiente uma vez antes de todos os testes."""
        cls.download_dir = os.path.join(os.getcwd(), "test_downloads")
        if os.path.exists(cls.download_dir):
            shutil.rmtree(cls.download_dir)
        os.makedirs(cls.download_dir)
        
        options = webdriver.ChromeOptions()
        prefs = {"download.default_directory": cls.download_dir, "download.prompt_for_download": False}
        options.add_experimental_option("prefs", prefs)
        options.add_argument("--start-maximized")

        if IS_CONTAINER:
            logger.info("Rodando em modo container (headless).")
            options.add_argument('--headless')
            options.add_argument('--no-sandbox')
            options.add_argument('--disable-dev-shm-usage')
            options.add_argument('--window-size=1920,1080')
            
        try:
            cls.browser = webdriver.Chrome(options=options)
            cls.wait = WebDriverWait(cls.browser, DEFAULT_TIMEOUT)
        except Exception as e:
            logger.error(f"Não foi possível iniciar o ChromeDriver: {e}")
            cls.browser = None
            sys.exit(1)
            
    @classmethod
    def tearDownClass(cls):
        """Encerra o navegador e limpa o banco de dados após todos os testes."""
        if cls.browser:
            cls.browser.quit()
        
        if os.path.exists(cls.download_dir):
            shutil.rmtree(cls.download_dir)
        logger.info("Navegador encerrado e pasta de downloads limpa.")

        logger.info(f"Iniciando limpeza do banco de dados ({DB_TYPE})...")
        time.sleep(1)

        conn = None
        try:
            if DB_TYPE == 'postgres':
                conn = psycopg2.connect(dbname=DB_NAME, user=DB_USER, password=DB_PASS, host=DB_HOST, port=DB_PORT)
            elif DB_TYPE == 'sqlite3':
                if not os.path.exists(DB_NAME):
                    logger.warning(f"Arquivo de banco de dados SQLite '{DB_NAME}' não encontrado. Pulando limpeza.")
                    return
                conn = sqlite3.connect(DB_NAME)
            else:
                logger.error(f"DB_TYPE '{DB_TYPE}' não suportado pelo script de teste.")
                return

            cursor = conn.cursor()
            queries = [
                "DELETE FROM movimentacoes WHERE conta LIKE 'Conta Saldo %';",
                "DELETE FROM movimentacoes WHERE conta = 'Conta Teste Grafico';",
                "DELETE FROM movimentacoes WHERE conta = 'Conta Teste CRUD';",
                "DELETE FROM movimentacoes WHERE conta = 'Conta Validação';",
                f"DELETE FROM user_profiles WHERE user_id = (SELECT id FROM users WHERE email = '{TEST_USER_EMAIL}');", # Limpa o perfil
                "DELETE FROM movimentacoes WHERE conta LIKE 'Conta Origem %';", # Limpeza do novo teste
                "DELETE FROM movimentacoes WHERE conta LIKE 'Conta Destino %';", # Limpeza do novo teste
            ]
            
            total_deleted = 0
            for query in queries:
                cursor.execute(query)
                if cursor.rowcount > 0:
                    total_deleted += cursor.rowcount
            conn.commit()
            logger.info(f"Limpeza do banco de dados concluída. {total_deleted} registros de teste removidos.")
        except (sqlite3.Error, psycopg2.Error) as e:
            logger.error(f"Ocorreu um erro ao limpar o banco de dados: {e}")
        finally:
            if conn:
                conn.close()

    def setUp(self):
        """Executado antes de cada teste. Realiza o login."""
        self.browser.get(f'{BASE_URL}/login')
        try:
            logout_button = self.browser.find_element(By.XPATH, "//button[text()='Sair']")
            logout_button.click()
            self.wait.until(EC.url_to_be(f'{BASE_URL}/login'))
        except NoSuchElementException:
            pass

        self.wait.until(EC.visibility_of_element_located((By.ID, "email"))).send_keys(TEST_USER_EMAIL)
        self.browser.find_element(By.ID, "password").send_keys(TEST_USER_PASS)
        self.browser.find_element(By.CSS_SELECTOR, "button[type='submit']").click()
        self.wait.until(EC.url_to_be(f'{BASE_URL}/'))
        self.wait.until(EC.title_contains("Saldos"))
        logger.info(f"Login como '{TEST_USER_EMAIL}' realizado com sucesso para o teste.")

    def _delay(self):
        """Pausa a execução para facilitar a visualização."""
        time.sleep(STEP_DELAY)

    def test_01_fluxo_crud_transacao(self):
        """Testa o fluxo completo: Criar, Ler, Atualizar e Deletar uma transação."""
        logger.info("--- INICIANDO TESTE 01: FLUXO CRUD ---")
        self.browser.get(f'{BASE_URL}/transacoes')
        self.wait.until(EC.title_contains("Transações"))
        
        timestamp = int(time.time())
        descricao_inicial = f"Pagamento Teste Selenium {timestamp}"
        descricao_editada = f"Pagamento Teste Selenium EDITADO {timestamp}"
        
        logger.info(f"CRIANDO nova transação: '{descricao_inicial}'")
        self.wait.until(EC.visibility_of_element_located((By.ID, "new_descricao"))).send_keys(descricao_inicial)
        self.browser.find_element(By.ID, "new_valor").send_keys("-150,77")
        self.browser.find_element(By.ID, "new_categoria").send_keys("Testes CRUD")
        self.browser.find_element(By.ID, "new_conta").send_keys("Conta Teste CRUD")
        self.browser.find_element(By.ID, "submit-movement-button").click()
        
        xpath_linha_inicial = f"//tr[contains(., '{descricao_inicial}')]"
        linha_criada = self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_linha_inicial)))
        self.assertIsNotNone(linha_criada, f"A transação '{descricao_inicial}' não apareceu na tabela.")
        logger.info("SUCESSO: Transação criada encontrada.")

        logger.info(f"EDITANDO a transação para '{descricao_editada}'")
        linha_criada.find_element(By.XPATH, ".//button[text()='Editar']").click()
        self.wait.until(EC.text_to_be_present_in_element_value((By.ID, "new_descricao"), descricao_inicial))
        campo_descricao = self.browser.find_element(By.ID, "new_descricao")
        campo_descricao.clear()
        campo_descricao.send_keys(descricao_editada)
        self.browser.find_element(By.ID, "submit-movement-button").click()
        
        xpath_linha_editada = f"//tr[contains(., '{descricao_editada}')]"
        linha_editada = self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_linha_editada)))
        self.assertIsNotNone(linha_editada, f"A transação editada '{descricao_editada}' não apareceu.")
        logger.info("SUCESSO: Transação editada encontrada.")

        logger.info(f"EXCLUINDO a transação '{descricao_editada}'")
        linha_editada.find_element(By.XPATH, ".//button[text()='Excluir']").click()
        
        self.wait.until(EC.alert_is_present()).accept()
        
        logger.info("Aguardando e aceitando o alerta de sucesso da exclusão...")
        self.wait.until(EC.alert_is_present()).accept()
        
        self.assertTrue(self.wait.until(EC.staleness_of(linha_editada)), f"A transação '{descricao_editada}' não foi removida.")
        logger.info("SUCESSO: Transação removida.")

    def test_02_configuracoes_dark_mode(self):
        """Testa a funcionalidade de ativar e desativar o dark mode."""
        logger.info("--- INICIANDO TESTE 02: DARK MODE ---")
        self.browser.get(f'{BASE_URL}/configuracoes')
        self.wait.until(EC.title_contains("Configurações"))

        toggle_label_selector = (By.CSS_SELECTOR, "label[for='dark-mode-toggle']")
        toggle_label = self.wait.until(EC.element_to_be_clickable(toggle_label_selector))
        html_element = self.browser.find_element(By.TAG_NAME, 'html')

        if 'dark' in html_element.get_attribute('class'):
            logger.info("Modo escuro estava ativo, desativando para iniciar o teste.")
            toggle_label.click()
            self.wait.until(lambda d: 'dark' not in d.find_element(By.TAG_NAME, 'html').get_attribute('class'))

        logger.info("Ativando o modo escuro...")
        toggle_label.click()
        self.wait.until(lambda d: 'dark' in d.find_element(By.TAG_NAME, 'html').get_attribute('class'))
        self.assertIn('dark', self.browser.find_element(By.TAG_NAME, 'html').get_attribute('class'))
        logger.info("SUCESSO: Modo escuro ativado.")
        self._delay()

        logger.info("Desativando o modo escuro...")
        toggle_label.click()
        self.wait.until(lambda d: 'dark' not in d.find_element(By.TAG_NAME, 'html').get_attribute('class'))
        self.assertNotIn('dark', self.browser.find_element(By.TAG_NAME, 'html').get_attribute('class'))
        logger.info("SUCESSO: Modo escuro desativado.")

    def test_03_profile_and_password(self):
        """Testa a edição do perfil e a validação da alteração de senha."""
        logger.info("--- INICIANDO TESTE 03: PERFIL E SENHA ---")
        self.browser.get(f'{BASE_URL}/configuracoes')
        self.wait.until(EC.title_contains("Configurações"))

        # Teste de Edição de Perfil
        cidade_teste = f"Cidade Teste {int(time.time())}"
        logger.info(f"Editando perfil. Alterando cidade para: '{cidade_teste}'")
        cidade_input = self.wait.until(EC.visibility_of_element_located((By.ID, "city")))
        cidade_input.clear()
        cidade_input.send_keys(cidade_teste)
        self.browser.find_element(By.ID, "save-profile-button").click()
        self.wait.until(EC.alert_is_present()).accept()
        logger.info("SUCESSO: Alerta de perfil salvo foi exibido e aceite.")

        self.browser.refresh()
        cidade_input_reloaded = self.wait.until(EC.visibility_of_element_located((By.ID, "city")))
        self.assertEqual(cidade_input_reloaded.get_attribute('value'), cidade_teste)
        logger.info("SUCESSO: A cidade alterada foi persistida após recarregar a página.")
        self._delay()

        # Teste de Validação de Senha (cenário de erro)
        logger.info("Testando validação de senha com senha atual incorreta.")
        self.browser.find_element(By.ID, "current_password").send_keys("senha_incorreta_propositalmente")
        self.browser.find_element(By.ID, "new_password").send_keys("qualquercoisa123")
        self.browser.find_element(By.ID, "confirm_new_password").send_keys("qualquercoisa123")
        self.browser.find_element(By.ID, "save-password-button").click()
        
        alert = self.wait.until(EC.alert_is_present())
        alert_text = alert.text
        alert.accept()
        
        self.assertIn("A senha atual está incorreta", alert_text)
        logger.info("SUCESSO: Alerta de erro para senha atual incorreta foi exibido corretamente.")


    def test_04_export_csv_functionality(self):
        """Testa a funcionalidade de exportação para CSV, com e sem filtros."""
        logger.info("--- INICIANDO TESTE 04: EXPORTAÇÃO CSV ---")

        user_id_query = f"SELECT id FROM users WHERE email = '{TEST_USER_EMAIL}'"
        desc_aluguel = f"Aluguel Teste Selenium {int(time.time())}"
        desc_salario = f"Salario Teste Selenium {int(time.time())}"
        conn = None
        try:
            if DB_TYPE == 'postgres':
                conn = psycopg2.connect(dbname=DB_NAME, user=DB_USER, password=DB_PASS, host=DB_HOST, port=DB_PORT)
            else: # sqlite3
                conn = sqlite3.connect(DB_NAME)
            
            cursor = conn.cursor()
            cursor.execute(user_id_query)
            test_user_id = cursor.fetchone()[0]

            logger.info(f"Inserindo dados de teste para o usuário ID: {test_user_id}")
            
            insert_query = "INSERT INTO movimentacoes (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (%s, %s, %s, %s, %s, %s, %s)"
            if DB_TYPE == 'sqlite3':
                insert_query = "INSERT INTO movimentacoes (user_id, data_ocorrencia, descricao, valor, categoria, conta, consolidado) VALUES (?, ?, ?, ?, ?, ?, ?)"

            cursor.execute(insert_query, (test_user_id, '2025-01-10', desc_aluguel, -1500.50, 'Moradia Teste', 'Conta Teste Export', True))
            cursor.execute(insert_query, (test_user_id, '2025-01-15', desc_salario, 5000.75, 'Renda Teste', 'Conta Teste Export', True))
            conn.commit()
        finally:
            if conn:
                conn.close()
            
        self.browser.get(f'{BASE_URL}/transacoes')
        self.wait.until(EC.title_contains("Transações"))

        def _verify_download(filename_part, expected_content_list, unexpected_content_list=None):
            logger.info(f"Verificando download que deve conter: {expected_content_list}...")
            downloaded_file_path = None
            for _ in range(15):
                files = [f for f in os.listdir(self.download_dir) if f.endswith('.csv') and not f.endswith('.crdownload')]
                if files:
                    downloaded_file_path = os.path.join(self.download_dir, files[0])
                    break
                time.sleep(1)
            
            self.assertIsNotNone(downloaded_file_path, "Nenhum arquivo .csv foi encontrado na pasta de downloads.")
            self.assertTrue(os.path.getsize(downloaded_file_path) > 0, "O arquivo CSV baixado está vazio.")
            
            with open(downloaded_file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                for expected in expected_content_list:
                    self.assertIn(expected, content, f"Conteúdo esperado '{expected}' não encontrado no CSV.")
                
                if unexpected_content_list:
                    for unexpected in unexpected_content_list:
                        self.assertNotIn(unexpected, content, f"Conteúdo inesperado '{unexpected}' foi encontrado no CSV.")
            
            logger.info(f"SUCESSO: Arquivo '{os.path.basename(downloaded_file_path)}' baixado e verificado.")
            os.remove(downloaded_file_path)

        # --- Teste 1: Exportação de TODOS os dados ---
        logger.info("Testando exportação de TODOS os dados (limpando filtros de data)...")
        start_date_input = self.wait.until(EC.visibility_of_element_located((By.ID, "start_date")))
        end_date_input = self.browser.find_element(By.ID, "end_date")
        
        start_date_input.clear()
        end_date_input.clear()
        self.browser.execute_script("arguments[0].dispatchEvent(new Event('change'))", end_date_input)

        export_button_locator = (By.ID, "export-csv-button")
        self.wait.until(EC.presence_of_element_located(export_button_locator))
        self.wait.until(
            lambda driver: "start_date" not in driver.find_element(*export_button_locator).get_attribute('href'),
            "O link de exportação não foi atualizado após limpar as datas."
        )
        
        export_button = self.browser.find_element(*export_button_locator)
        export_button.click()
        
        _verify_download(
            "backup_minhas_economias",
            expected_content_list=['Data Ocorrência;Descrição;Valor', desc_aluguel, desc_salario]
        )
        
        # --- Teste 2: Exportação com filtro ---
        logger.info(f"Testando exportação com filtro de descrição ('{desc_aluguel}')...")
        search_box = self.browser.find_element(By.ID, "search_descricao")
        search_box.clear()
        search_box.send_keys(desc_aluguel)
        
        self.browser.execute_script("arguments[0].dispatchEvent(new Event('change'))", search_box)
        
        export_button = self.browser.find_element(*export_button_locator)
        
        desc_aluguel_encoded = quote_plus(desc_aluguel)
        self.wait.until(
            lambda driver: desc_aluguel_encoded in driver.find_element(*export_button_locator).get_attribute('href'),
            f"O link do botão de exportação não foi atualizado com o filtro codificado ('{desc_aluguel_encoded}')."
        )
        
        logger.info(f"Link com filtro: {export_button.get_attribute('href')}")
        export_button.click()
        
        _verify_download(
            "backup_minhas_economias",
            expected_content_list=[desc_aluguel],
            unexpected_content_list=[desc_salario]
        )

    def test_05_fluxo_crud_investimentos(self):
        """Testa a adição, edição e exclusão de ativos nacionais e internacionais."""
        logger.info("--- INICIANDO TESTE 05: FLUXO CRUD INVESTIMENTOS ---")
        self.browser.get(f'{BASE_URL}/investimentos')
        self.wait.until(EC.title_contains("Investimentos"))

        logger.info("Aguardando carregamento assíncrono dos preços...")
        self.wait.until(
            EC.none_of(EC.presence_of_all_elements_located((By.CLASS_NAME, "spinner")))
        )
        logger.info("SUCESSO: Preços carregados.")
        self._delay()

        # --- Teste de Ativo Nacional (Ação) ---
        ticker_nacional = "CIEL3"
        logger.info(f"Adicionando ativo nacional: {ticker_nacional}")
        self.browser.find_element(By.ID, "add-nacional-ticker").send_keys(ticker_nacional)
        self.browser.find_element(By.ID, "add-nacional-quantidade").send_keys("300")
        self.browser.find_element(By.CSS_SELECTOR, "#add-nacional-form button").click()
        self.wait.until(EC.alert_is_present()).accept()
        self.browser.refresh()

        logger.info("Verificando se o ativo nacional foi adicionado...")
        linha_nacional = self.wait.until(EC.visibility_of_element_located((By.CSS_SELECTOR, f'tr[data-ticker="{ticker_nacional}"]')))
        self.assertIn(ticker_nacional, linha_nacional.text)
        logger.info("SUCESSO: Ativo nacional adicionado.")
        self._delay()

        logger.info(f"Editando ativo nacional: {ticker_nacional}")
        linha_nacional.find_element(By.CSS_SELECTOR, ".edit-button").click()
        edit_form = self.wait.until(EC.visibility_of_element_located((By.ID, "edit-investment-section")))
        self.assertTrue(edit_form.is_displayed())
        
        edit_quantity_input = edit_form.find_element(By.ID, "edit-quantity")
        edit_quantity_input.clear()
        edit_quantity_input.send_keys("350")
        edit_form.find_element(By.ID, "submit-edit-button").click()
        self.wait.until(EC.alert_is_present()).accept()
        self.browser.refresh()

        logger.info("Verificando se a quantidade foi atualizada...")
        linha_nacional_editada = self.wait.until(EC.visibility_of_element_located((By.CSS_SELECTOR, f'tr[data-ticker="{ticker_nacional}"]')))
        self.assertIn("350", linha_nacional_editada.text)
        logger.info("SUCESSO: Ativo nacional editado.")
        self._delay()

        logger.info(f"Excluindo ativo nacional: {ticker_nacional}")
        linha_nacional_editada.find_element(By.CSS_SELECTOR, ".delete-button").click()
        self.wait.until(EC.alert_is_present()).accept() # Confirmação
        self.wait.until(EC.alert_is_present()).accept() # Sucesso
        self.browser.refresh()

        logger.info("Verificando se o ativo foi removido...")
        self.wait.until(EC.invisibility_of_element_located((By.CSS_SELECTOR, f'tr[data-ticker="{ticker_nacional}"]')))
        logger.info("SUCESSO: Ativo nacional excluído.")

        # --- Teste de Ativo Internacional ---
        ticker_internacional = "DIS"
        logger.info(f"Adicionando ativo internacional: {ticker_internacional}")
        self.browser.find_element(By.ID, "add-internacional-ticker").send_keys(ticker_internacional)
        self.browser.find_element(By.ID, "add-internacional-descricao").send_keys("Ações da Disney")
        self.browser.find_element(By.ID, "add-internacional-quantidade").send_keys("25.5")
        self.browser.find_element(By.CSS_SELECTOR, "#add-internacional-form button").click()
        self.wait.until(EC.alert_is_present()).accept()
        self.browser.refresh()

        logger.info("Verificando se o ativo internacional foi adicionado...")
        linha_internacional = self.wait.until(EC.visibility_of_element_located((By.CSS_SELECTOR, f'tr[data-ticker="{ticker_internacional}"]')))
        self.assertIn(ticker_internacional, linha_internacional.text)
        logger.info("SUCESSO: Ativo internacional adicionado.")
        self._delay()

        logger.info(f"Excluindo ativo internacional: {ticker_internacional}")
        linha_internacional.find_element(By.CSS_SELECTOR, ".delete-button").click()
        self.wait.until(EC.alert_is_present()).accept() # Confirmação
        self.wait.until(EC.alert_is_present()).accept() # Sucesso
        self.browser.refresh()

        logger.info("Verificando se o ativo foi removido...")
        self.wait.until(EC.invisibility_of_element_located((By.CSS_SELECTOR, f'tr[data-ticker="{ticker_internacional}"]')))
        logger.info("SUCESSO: Ativo internacional excluído.")

    def test_06_fluxo_transferencia(self):
        """Testa o fluxo completo de criação de uma transferência entre contas."""
        logger.info("--- INICIANDO TESTE 06: FLUXO DE TRANSFERÊNCIA ---")
        self.browser.get(f'{BASE_URL}/transacoes')
        self.wait.until(EC.title_contains("Transações"))
        
        # Gerar dados únicos para o teste
        timestamp = int(time.time())
        descricao_base = f"Transf Viagem {timestamp}"
        conta_origem = f"Conta Origem {timestamp}"
        conta_destino = f"Conta Destino {timestamp}"
        valor_transferencia = "98,55"
        valor_assert = "98.55"
        
        logger.info("Selecionando o tipo 'Transferência' no formulário.")
        
        # Localiza e clica no seletor de tipo para abrir as opções
        tipo_display = self.wait.until(EC.element_to_be_clickable((By.ID, "tipo-movimentacao-display")))
        tipo_display.click()
        
        # Clica na opção "Transferência"
        transferencia_label = self.wait.until(EC.element_to_be_clickable((By.XPATH, "//label[contains(., 'Transferência')]")))
        transferencia_label.click()
        
        logger.info("Verificando se o formulário foi alterado para o modo de transferência.")
        
        # Campos que devem aparecer
        conta_origem_input = self.wait.until(EC.visibility_of_element_located((By.ID, "new_conta_origem")))
        conta_destino_input = self.browser.find_element(By.ID, "new_conta_destino")
        self.assertTrue(conta_origem_input.is_displayed(), "O campo 'Conta de Origem' não ficou visível.")
        self.assertTrue(conta_destino_input.is_displayed(), "O campo 'Conta de Destino' não ficou visível.")

        # Campos que devem desaparecer
        self.wait.until(EC.invisibility_of_element_located((By.ID, "group-categoria")))
        self.wait.until(EC.invisibility_of_element_located((By.ID, "group-conta")))
        logger.info("SUCESSO: Formulário alterado corretamente.")
        self._delay()
        
        logger.info(f"Preenchendo o formulário para transferir {valor_transferencia} de '{conta_origem}' para '{conta_destino}'.")
        self.browser.find_element(By.ID, "new_descricao").send_keys(descricao_base)
        self.browser.find_element(By.ID, "new_valor").send_keys(valor_transferencia) 
        conta_origem_input.send_keys(conta_origem)
        conta_destino_input.send_keys(conta_destino)
        
        self.browser.find_element(By.ID, "submit-movement-button").click()
        
        logger.info("Verificando se as duas movimentações da transferência foram criadas.")
        
        valor_formatado_debito = f"-{float(valor_assert):.2f}"
        valor_formatado_credito = f"{float(valor_assert):.2f}"

        xpath_debito = f"//tr[contains(., '{conta_origem}') and (contains(., '{valor_formatado_debito.replace('.', ',')}') or contains(., '{valor_formatado_debito}')) and contains(., 'Transferência') and contains(., '{descricao_base}')]"
        linha_debito = self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_debito)))
        self.assertIsNotNone(linha_debito, "A movimentação de débito da transferência não foi encontrada.")
        logger.info("SUCESSO: Movimentação de débito encontrada.")
        
        xpath_credito = f"//tr[contains(., '{conta_destino}') and (contains(., '{valor_formatado_credito.replace('.', ',')}') or contains(., '{valor_formatado_credito}')) and contains(., 'Transferência') and contains(., '{descricao_base}')]"
        linha_credito = self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_credito)))
        self.assertIsNotNone(linha_credito, "A movimentação de crédito da transferência não foi encontrada.")
        logger.info("SUCESSO: Movimentação de crédito encontrada.")
        self._delay()
        
        logger.info("Iniciando limpeza das movimentações de transferência criadas.")
        
        # Pega os IDs para uma exclusão mais segura
        id_debito = linha_debito.get_attribute("data-id")
        id_credito = linha_credito.get_attribute("data-id")

        # Exclui a linha de débito
        delete_button_debito = self.wait.until(EC.element_to_be_clickable((By.CSS_SELECTOR, f"tr[data-id='{id_debito}'] .delete-button")))
        delete_button_debito.click()
        self.wait.until(EC.alert_is_present()).accept()
        self.wait.until(EC.alert_is_present()).accept()
        logger.info(f"Linha de débito (ID: {id_debito}) excluída.")
        
        self.wait.until(EC.invisibility_of_element_located((By.CSS_SELECTOR, f"tr[data-id='{id_debito}']")))
        
        # Exclui a linha de crédito
        delete_button_credito = self.wait.until(EC.element_to_be_clickable((By.CSS_SELECTOR, f"tr[data-id='{id_credito}'] .delete-button")))
        delete_button_credito.click()
        self.wait.until(EC.alert_is_present()).accept()
        self.wait.until(EC.alert_is_present()).accept()
        logger.info(f"Linha de crédito (ID: {id_credito}) excluída.")

        self.wait.until(EC.invisibility_of_element_located((By.CSS_SELECTOR, f"tr[data-id='{id_credito}']")))
        logger.info("SUCESSO: Limpeza concluída.")


if __name__ == '__main__':
    if os.system(f'go run create_user.go -email="{TEST_USER_EMAIL}" -password="{TEST_USER_PASS}"') != 0:
        logger.warning(f"Pode ter ocorrido um erro ao criar/verificar o usuário de teste. Se os testes falharem no login, verifique se o usuário '{TEST_USER_EMAIL}' existe.")
    
    unittest.main(verbosity=2, failfast=True)