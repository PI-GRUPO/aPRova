import { expect, type Locator, type Page, test } from "@playwright/test";

function fila(pagina: Page) {
	return pagina.getByRole("region", { name: "Aguardando aprovação" });
}

function linhasDaFila(pagina: Page) {
	return fila(pagina)
		.getByRole("row")
		.filter({ has: pagina.getByRole("cell") });
}

async function baseDe(alvo: Locator) {
	const caixa = await alvo.boundingBox();
	if (!caixa) throw new Error("elemento sem caixa");
	return caixa.y + caixa.height;
}

test.describe("faixa de métricas", () => {
	test("mostra quatro números com unidade e barra de proporção", async ({ page }) => {
		await page.goto("/");
		const metricas = page.getByTestId("metrica");
		await expect(metricas).toHaveCount(4);
		for (const [rotulo, valor] of [
			["Aguardando aprovação", "5"],
			["Taxa de bloqueio", "47%"],
			["Falhas no sandbox", "31%"],
			["Análises recebidas", "41"],
		] as const) {
			const metrica = metricas.filter({ hasText: rotulo });
			await expect(metrica.getByTestId("metrica-valor")).toHaveText(valor);
			await expect(metrica.getByTestId("proporcao")).toBeVisible();
		}
	});

	test("o valor usa algarismos tabulares em 32px", async ({ page }) => {
		await page.goto("/");
		const estilo = await page
			.getByTestId("metrica-valor")
			.first()
			.evaluate((elemento) => {
				const computado = getComputedStyle(elemento);
				return { tamanho: computado.fontSize, numerais: computado.fontVariantNumeric };
			});
		expect(estilo.tamanho).toBe("32px");
		expect(estilo.numerais).toContain("tabular-nums");
	});
});

test.describe("fila de aprovação", () => {
	test("não tem coluna Nível e diz o nível uma vez, no título", async ({ page }) => {
		await page.goto("/");
		await expect(fila(page).getByRole("columnheader", { name: "Nível" })).toHaveCount(0);
		await expect(fila(page).getByTestId("selo-risco-alto")).toHaveCount(1);
		await expect(linhasDaFila(page)).toHaveCount(5);
	});

	test("as linhas têm 32px de altura", async ({ page }) => {
		await page.goto("/");
		const caixa = await linhasDaFila(page).first().boundingBox();
		expect(caixa?.height).toBeGreaterThanOrEqual(32);
		expect(caixa?.height).toBeLessThan(34);
	});

	test("a busca filtra por branch e avisa quando nada corresponde", async ({ page }) => {
		await page.goto("/");
		const busca = fila(page).getByRole("searchbox", { name: "Buscar na fila" });
		await busca.fill("juros");
		await expect(linhasDaFila(page)).toHaveCount(1);
		await expect(linhasDaFila(page)).toContainText("cobranca-api");
		await busca.fill("nada-que-exista");
		await expect(linhasDaFila(page)).toHaveCount(0);
		await expect(fila(page).getByText("Nenhuma análise da fila corresponde ao filtro.")).toBeVisible();
	});

	test("o filtro de repositório reduz a fila", async ({ page }) => {
		await page.goto("/");
		await fila(page).getByRole("combobox", { name: "Repositório" }).click();
		await page.getByRole("option", { name: "cobranca-api" }).click();
		await expect(linhasDaFila(page)).toHaveCount(2);
		await expect(fila(page).getByText("2 de 5 análises")).toBeVisible();
	});

	test("a coluna Espera ordena nos dois sentidos", async ({ page }) => {
		await page.goto("/");
		const cabecalho = fila(page).getByRole("columnheader", { name: /Espera/ });
		await expect(cabecalho).toHaveAttribute("aria-sort", "descending");
		await expect(linhasDaFila(page).first()).toContainText("aPRova-TESTES");
		await cabecalho.getByRole("button").click();
		await expect(cabecalho).toHaveAttribute("aria-sort", "ascending");
		await expect(linhasDaFila(page).first()).toContainText("47 min");
	});

	test("cada motivo leva à sua regra", async ({ page }) => {
		await page.goto("/");
		const motivo = fila(page).getByRole("link", { name: "falha_sandbox" });
		await expect(motivo).toHaveAttribute("href", "/regras#falha_sandbox");
	});
});

test.describe("painéis do período", () => {
	test("as falhas por tipo dizem que estão dentro do total", async ({ page }) => {
		await page.goto("/");
		const falhas = page.getByRole("region", { name: "Falhas por tipo" });
		await expect(falhas.getByText("das 13 falhas")).toBeVisible();
		await expect(falhas.getByRole("row").filter({ hasText: "Infraestrutura" })).toContainText("9");
	});

	test("os conceitos descartados não existem mais", async ({ page }) => {
		await page.goto("/?conceito=b");
		await expect(page.getByRole("heading", { name: "Pior nível de risco por dia" })).toHaveCount(0);
		await expect(page.getByTestId("metrica")).toHaveCount(4);
	});

	test("nenhuma coluna deixa buraco no desktop", async ({ page }) => {
		await page.goto("/");
		const paineis = page.getByTestId("painel-do-periodo");
		await expect(paineis).toHaveCount(3);
		const bases = await Promise.all((await paineis.all()).map(baseDe));
		expect(Math.max(...bases) - Math.min(...bases)).toBeLessThanOrEqual(1);
		const larguraDaFila = (await fila(page).boundingBox())?.width ?? 0;
		const larguraDaFaixa = (await page.getByTestId("faixa-de-metricas").boundingBox())?.width ?? 0;
		expect(Math.abs(larguraDaFila - larguraDaFaixa)).toBeLessThanOrEqual(1);
	});
});

test.describe("moldura", () => {
	test("a barra lateral ocupa a altura toda e o rodapé fica à vista", async ({ page }) => {
		await page.setViewportSize({ width: 1440, height: 480 });
		await page.goto("/");
		const alturaDoDocumento = await page.evaluate(() => document.documentElement.scrollHeight);
		expect(alturaDoDocumento).toBeGreaterThan(480);
		const lateral = await page.getByTestId("barra-lateral").boundingBox();
		expect(lateral?.height).toBeGreaterThanOrEqual(alturaDoDocumento - 1);

		await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
		await expect(page.getByRole("button", { name: /Usar tema/ })).toBeInViewport({ ratio: 1 });
	});

	test("em 390px as quatro abas cabem sem rolagem horizontal", async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/");
		const navegacao = page.getByRole("navigation", { name: "Principal" });
		const sobra = await navegacao.evaluate((elemento) => elemento.scrollWidth - elemento.clientWidth);
		expect(sobra).toBeLessThanOrEqual(0);
		for (const nome of ["Visão geral", "Análises", "Recibos", "Regras do gate"]) {
			const caixa = await navegacao.getByRole("link", { name: nome, exact: true }).boundingBox();
			expect(caixa?.x).toBeGreaterThanOrEqual(0);
			expect((caixa?.x ?? 0) + (caixa?.width ?? 0)).toBeLessThanOrEqual(390);
		}
		const larguraDaPagina = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(larguraDaPagina).toBeLessThanOrEqual(390);
	});
});

test.describe("correções da critique", () => {
	test("o cabeçalho diz quando os dados foram atualizados", async ({ page }) => {
		await page.goto("/");
		await expect(page.getByRole("main").getByText(/atualizado em 11 nov 2026, 14:00/)).toBeVisible();
	});

	test("o título da fila escreve risco alto por extenso", async ({ page }) => {
		await page.goto("/");
		await expect(fila(page).getByTestId("selo-risco-alto")).toHaveText("risco alto");
	});

	test("a evidência continua visível em 900px e caminho de arquivo vem em Plex Mono", async ({ page }) => {
		await page.setViewportSize({ width: 900, height: 900 });
		await page.goto("/");
		await expect(fila(page).getByRole("columnheader", { name: "Evidência" })).toBeVisible();
		const caminho = fila(page).getByRole("cell").getByText("go.mod", { exact: true }).filter({ visible: true });
		await expect(caminho).toBeVisible();
		expect(await caminho.evaluate((elemento) => getComputedStyle(elemento).fontFamily)).toContain("IBM Plex Mono");
		const larguraDaPagina = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(larguraDaPagina).toBeLessThanOrEqual(900);
	});

	for (const largura of [768, 900, 1100, 1280]) {
		test(`a tabela da fila não rola por dentro em ${largura}px`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: 900 });
			await page.goto("/");
			const sobra = await fila(page)
				.getByRole("table")
				.evaluate((tabela) => {
					const moldura = tabela.parentElement;
					return moldura ? moldura.scrollWidth - moldura.clientWidth : 0;
				});
			expect(sobra).toBeLessThanOrEqual(0);
		});
	}

	test("a linha avisa quando há mais evidências do que a mostrada", async ({ page }) => {
		await page.goto("/");
		const linha = linhasDaFila(page).filter({ hasText: "#97" });
		await expect(linha.getByText(/^e mais \d+$/).filter({ visible: true })).toHaveCount(1);
	});

	test("branch e commit podem ser selecionados sem abrir a análise", async ({ page }) => {
		await page.goto("/");
		const branch = fila(page).getByText("teste-peso-zero");
		await branch.click();
		await expect(page).toHaveURL(/\/$/);
	});

	test("cada dia do gráfico diz seus números ao passar o ponteiro", async ({ page }) => {
		await page.goto("/");
		const dias = page.getByTestId("dia-do-grafico");
		await expect(dias).toHaveCount(30);
		await expect(dias.last()).toHaveAttribute("title", /^11 nov: \d+ alto, \d+ médio, \d+ baixo$/);
	});

	test("no celular os links de motivo têm pelo menos 24px de altura", async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/");
		const caixa = await fila(page).getByRole("link", { name: "falha_sandbox" }).boundingBox();
		expect(caixa?.height).toBeGreaterThanOrEqual(24);
	});
});
