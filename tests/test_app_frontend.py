#!/usr/bin/env python3

import os
import unittest
import sys
import logging
import time
import shutil
import sqlite3
from datetime import date
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException

# --- Configurações ---
logging.basicConfig(format='%(asctime)s: %(name)s: %(levelname)s: %(message)s', level=logging.INFO, datefmt='%H:%M:%S')
logger = logging.getLogger('MinhasEconomiasTest')

BASE_URL = os.getenv('APP_URL', 'http://localhost:8080')
DEFAULT_TIMEOUT = 10 
IS_CONTAINER = os.getenv('CONTAINER', 'false').lower() == 'true'
STEP_DELAY = 0

class MinhasEconomiasTest(unittest.TestCase):
    
    @classmethod
    def setUpClass(cls):
        """Configura o ambiente uma vez antes de todos os testes."""
        cls.download_dir = os.path.join(os.getcwd(), "test_downloads")
        if os.path.exists(cls.download_dir):
            shutil.rmtree(cls.download_dir)
        os.makedirs(cls.download_dir)
        
        options = webdriver.ChromeOptions()
        prefs = {
            "download.default_directory": cls.download_dir,
            "download.prompt_for_download": False
        }
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
        """Encerra o navegador, limpa os arquivos e o banco de dados após todos os testes."""
        if cls.browser:
            cls.browser.quit()
        
        if os.path.exists(cls.download_dir):
            shutil.rmtree(cls.download_dir)
        logger.info("Navegador encerrado e pasta de downloads limpa.")

        # --- CORREÇÃO: Adicionada pausa para evitar bloqueio do banco de dados ---
        logger.info("Aguardando 2 segundos para a aplicação Go liberar o arquivo de banco de dados...")
        time.sleep(2)

        logger.info("Iniciando limpeza do banco de dados...")
        db_path = "extratos.db"
        if not os.path.exists(db_path):
            logger.warning(f"Arquivo de banco de dados '{db_path}' não encontrado. Pulando limpeza.")
            return

        try:
            conn = sqlite3.connect(db_path)
            cursor = conn.cursor()
            
            queries = [
                "DELETE FROM movimentacoes WHERE conta LIKE 'Conta Saldo %';",
                "DELETE FROM movimentacoes WHERE conta = 'Conta Teste Grafico';",
                "DELETE FROM movimentacoes WHERE conta = 'Conta Teste CRUD';",
                "DELETE FROM movimentacoes WHERE conta = 'Conta Validação';"
            ]
            
            total_deleted = 0
            for query in queries:
                cursor.execute(query)
                total_deleted += cursor.rowcount

            conn.commit()
            if total_deleted > 0:
                logger.info(f"SUCESSO: Limpeza do banco de dados concluída. {total_deleted} registros de teste removidos.")
            else:
                logger.info("Limpeza do banco de dados concluída. Nenhum registro de teste encontrado para remover.")
            
        except sqlite3.Error as e:
            logger.error(f"Ocorreu um erro ao limpar o banco de dados: {e}")
        finally:
            if 'conn' in locals() and conn:
                conn.close()


    def _delay(self):
        """Pausa a execução para facilitar a visualização."""
        time.sleep(STEP_DELAY)

    def test_01_fluxo_crud_transacao(self):
        """Testa o fluxo completo: Criar, Ler, Atualizar e Deletar uma transação."""
        logger.info("--- INICIANDO TESTE 01: FLUXO CRUD ---")
        transacoes_url = f'{BASE_URL}/transacoes'
        timestamp = int(time.time())
        descricao_inicial = f"Pagamento Teste Selenium {timestamp}"
        descricao_editada = f"Pagamento Teste Selenium EDITADO {timestamp}"
        self.browser.get(transacoes_url)
        self._delay()
        self.assertIn("Transações Financeiras", self.browser.title)
        logger.info(f"CRIANDO nova transação: '{descricao_inicial}'")
        self.wait.until(EC.visibility_of_element_located((By.ID, "new_descricao"))).send_keys(descricao_inicial)
        self.browser.find_element(By.ID, "new_valor").send_keys("-150,77")
        self.browser.find_element(By.ID, "new_categoria").send_keys("Testes CRUD")
        self.browser.find_element(By.ID, "new_conta").send_keys("Conta Teste CRUD")
        self.browser.find_element(By.ID, "submit-movement-button").click()
        xpath_linha_inicial = f"//tr[contains(., '{descricao_inicial}')]"
        try:
            linha_criada = self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_linha_inicial)))
            self.assertIsNotNone(linha_criada)
            logger.info("SUCESSO: Transação criada encontrada.")
        except TimeoutException:
            self.fail(f"A transação '{descricao_inicial}' não apareceu na tabela.")
        logger.info(f"EDITANDO a transação para '{descricao_editada}'")
        linha_criada.find_element(By.XPATH, ".//button[text()='Editar']").click()
        self._delay()
        self.wait.until(EC.text_to_be_present_in_element_value((By.ID, "new_descricao"), descricao_inicial))
        campo_descricao = self.browser.find_element(By.ID, "new_descricao")
        campo_descricao.clear()
        campo_descricao.send_keys(descricao_editada)
        self.browser.find_element(By.ID, "submit-movement-button").click()
        xpath_linha_editada = f"//tr[contains(., '{descricao_editada}')]"
        try:
            self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_linha_editada)))
            logger.info("SUCESSO: Transação editada encontrada.")
        except TimeoutException:
            self.fail(f"A transação editada '{descricao_editada}' não apareceu.")
        logger.info(f"EXCLUINDO a transação '{descricao_editada}'")
        linha_para_excluir = self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_linha_editada)))
        linha_para_excluir.find_element(By.XPATH, ".//button[text()='Excluir']").click()
        self.wait.until(EC.alert_is_present()).accept()
        self.wait.until(EC.alert_is_present()).accept()
        try:
            self.wait.until(EC.staleness_of(linha_para_excluir))
            logger.info("SUCESSO: Transação removida.")
        except TimeoutException:
            self.fail(f"A transação '{descricao_editada}' não foi removida.")

    def test_02_pagina_inicial(self):
        """Testa a exibição de saldos na página inicial."""
        logger.info("--- INICIANDO TESTE 02: PÁGINA INICIAL ---")
        conta_teste = f"Conta Saldo {int(time.time())}"
        self.browser.get(f'{BASE_URL}/transacoes')
        self.wait.until(EC.visibility_of_element_located((By.ID, "new_descricao"))).send_keys("Receita para Saldo")
        self.browser.find_element(By.ID, "new_valor").send_keys("1250,00")
        self.browser.find_element(By.ID, "new_conta").send_keys(conta_teste)
        self.browser.find_element(By.ID, "submit-movement-button").click()
        self.wait.until(EC.visibility_of_element_located((By.XPATH, f"//tr[contains(., '{conta_teste}')]")))
        logger.info(f"Transação de teste criada para a conta '{conta_teste}'.")
        self._delay()
        self.browser.get(BASE_URL + "/")
        self.assertIn("Minhas Economias - Saldos", self.browser.title)
        xpath_saldo_card = f"//div[p[contains(text(), '{conta_teste}')]]//p[contains(text(), '1250.00')]"
        try:
            self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_saldo_card)))
            logger.info("SUCESSO: Saldo da conta de teste encontrado na página inicial.")
        except TimeoutException:
            self.fail("Não foi possível encontrar o saldo da conta de teste na página inicial.")
        self._delay()

    def test_03_download_pdf(self):
        """Testa a funcionalidade de download de PDF."""
        logger.info("--- INICIANDO TESTE 03: DOWNLOAD DE PDF ---")
        self.browser.get(f'{BASE_URL}/relatorio')
        self.assertIn("Relatório de Despesas", self.browser.title)
        download_button = self.wait.until(EC.element_to_be_clickable((By.ID, "save-pdf-button")))
        download_button.click()
        logger.info("Botão de download do PDF clicado.")
        data_hoje = date.today().strftime('%Y-%m-%d')
        nome_arquivo_esperado = f"Relatorio-MinhasEconomias-{data_hoje}.pdf"
        caminho_arquivo = os.path.join(self.download_dir, nome_arquivo_esperado)
        tempo_limite = 20
        download_completo = False
        logger.info(f"Aguardando o download do arquivo '{nome_arquivo_esperado}'...")
        for _ in range(tempo_limite):
            if os.path.exists(caminho_arquivo) and not any(".crdownload" in f for f in os.listdir(self.download_dir)):
                logger.info("SUCESSO: Arquivo PDF encontrado no diretório de downloads!")
                download_completo = True
                break
            time.sleep(1)
        self.assertTrue(download_completo, f"FALHA: O download do PDF '{nome_arquivo_esperado}' não foi concluído no tempo.")
        self.assertGreater(os.path.getsize(caminho_arquivo), 0, "O arquivo PDF baixado está vazio.")
    
    def test_04_clique_grafico_relatorio(self):
        """Testa a interatividade do gráfico na página de relatório."""
        logger.info("--- INICIANDO TESTE 04: INTERATIVIDADE DO GRÁFICO ---")
        timestamp = int(time.time())
        categoria_grafico = f"Categoria Grafico {timestamp}"
        descricao_grafico = f"Despesa para teste de gráfico {timestamp}"
        self.browser.get(f'{BASE_URL}/transacoes')
        self.wait.until(EC.visibility_of_element_located((By.ID, "new_descricao"))).send_keys(descricao_grafico)
        self.browser.find_element(By.ID, "new_valor").send_keys("-99.99")
        self.browser.find_element(By.ID, "new_categoria").send_keys(categoria_grafico)
        self.browser.find_element(By.ID, "new_conta").send_keys("Conta Teste Grafico")
        self.browser.find_element(By.ID, "submit-movement-button").click()
        self.wait.until(EC.url_contains('/transacoes'))
        logger.info(f"Transação de teste criada para a categoria '{categoria_grafico}'.")
        self._delay()
        self.browser.get(f'{BASE_URL}/relatorio')
        self.assertIn("Relatório de Despesas", self.browser.title)
        logger.info(f"Filtrando o relatório pela categoria '{categoria_grafico}'...")
        try:
            self.wait.until(EC.element_to_be_clickable((By.ID, "category-select-display"))).click()
            self._delay()
            categoria_checkbox_label = self.wait.until(EC.element_to_be_clickable((By.XPATH, f"//label[contains(., '{categoria_grafico}')]")))
            categoria_checkbox_label.click()
            self._delay()
            self.browser.find_element(By.TAG_NAME, 'body').click()
            self.browser.find_element(By.XPATH, "//button[text()='Filtrar Relatório']").click()
            logger.info("Filtro aplicado.")
        except TimeoutException:
            self.fail("Não foi possível encontrar e aplicar o filtro de categoria no relatório.")
        try:
            canvas = self.wait.until(EC.element_to_be_clickable((By.ID, "expensesPieChart")))
            time.sleep(2) 
            canvas.click()
            logger.info("Clicou no canvas do gráfico já filtrado.")
        except TimeoutException:
            self.fail("O gráfico não foi encontrado ou não está clicável após filtrar o relatório.")
        try:
            secao_transacoes = self.wait.until(EC.visibility_of_element_located((By.ID, "category-transactions-section")))
            logger.info("SUCESSO: A seção de detalhes da categoria está visível.")
            xpath_transacao_detalhe = f"//tbody[@id='category-transactions-tbody']//td[contains(text(), '{descricao_grafico}')]"
            self.wait.until(EC.visibility_of_element_located((By.XPATH, xpath_transacao_detalhe)))
            logger.info("SUCESSO: A transação de teste foi encontrada na tabela de detalhes.")
        except TimeoutException:
            self.fail("A tabela de detalhes da categoria não apareceu ou não continha a transação esperada.")

    def test_05_validacoes_formulario(self):
        """Testa as validações dos campos do formulário."""
        logger.info("--- INICIANDO TESTE 05: VALIDAÇÕES DE FORMULÁRIO ---")
        transacoes_url = f'{BASE_URL}/transacoes'

        logger.info("Cenário de Validação: Descrição com mais de 60 caracteres.")
        self.browser.get(transacoes_url)
        campo_descricao = self.wait.until(EC.visibility_of_element_located((By.ID, "new_descricao")))
        descricao_longa = 'a' * 61
        campo_descricao.send_keys(descricao_longa)
        self._delay()
        valor_no_campo = campo_descricao.get_attribute("value")
        self.assertEqual(len(valor_no_campo), 60, "O navegador deveria ter limitado a descrição a 60 caracteres.")
        logger.info("SUCESSO: Validação de frontend 'maxlength=60' funcionou como esperado.")
        
        logger.info("Cenário de Validação: Sanitização de valor com formato de texto.")
        self.browser.get(transacoes_url)
        campo_valor = self.wait.until(EC.visibility_of_element_located((By.ID, "new_valor")))
        campo_valor.send_keys("abc-123,45xyz")
        self._delay()
        valor_sanitizado = campo_valor.get_attribute("value")
        self.assertEqual(valor_sanitizado, "-123,45", "O JavaScript deveria ter limpado os caracteres inválidos.")
        logger.info("SUCESSO: Validação de frontend (JavaScript) para o campo valor funcionou.")


if __name__ == '__main__':
    unittest.main(verbosity=2, failfast=True)