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

	test("as linhas têm 36px de altura", async ({ page }) => {
		await page.goto("/");
		const caixa = await linhasDaFila(page).first().boundingBox();
		expect(caixa?.height).toBeGreaterThanOrEqual(36);
		expect(caixa?.height).toBeLessThan(38);
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
		await expect(page.getByRole("main").getByText("Últimos 30 dias · 13 out a 11 nov 2026 · atualizado às 14:00")).toBeVisible();
	});

	test("o título da fila escreve risco alto por extenso", async ({ page }) => {
		await page.goto("/");
		await expect(fila(page).getByTestId("selo-risco-alto")).toHaveText("risco alto");
	});

	test("a evidência continua visível em 900px e caminho de arquivo vem em Plex Mono", async ({ page }) => {
		await page.setViewportSize({ width: 900, height: 900 });
		await page.goto("/");
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

	test("o mapa de calor tem uma célula por dia e por nível, com valor no ponteiro", async ({ page }) => {
		await page.goto("/");
		const mapa = page.getByRole("region", { name: "Análises por dia e nível" });
		const celulas = mapa.getByTestId("celula-do-mapa");
		await expect(celulas).toHaveCount(90);
		await expect(celulas.last()).toHaveAttribute("aria-label", /^11 nov, baixo: \d+ (análise|análises)$/);
		for (const nivel of ["alto", "médio", "baixo"]) {
			await expect(mapa.getByTestId("linha-do-mapa").filter({ hasText: nivel })).toHaveCount(1);
		}
	});

	test("a intensidade da célula cresce com a contagem e a escala fica à vista", async ({ page }) => {
		await page.goto("/");
		const mapa = page.getByRole("region", { name: "Análises por dia e nível" });
		const intensidades = await mapa
			.getByTestId("celula-do-mapa")
			.evaluateAll((celulas) =>
				celulas.map((celula) => [Number(celula.getAttribute("data-quantas")), Number(celula.getAttribute("data-passo"))]),
			);
		for (const [quantas, passo] of intensidades) {
			expect(passo === 0).toBe(quantas === 0);
		}
		const maiorPasso = Math.max(...intensidades.map(([, passo]) => passo));
		const maiorContagem = Math.max(...intensidades.map(([quantas]) => quantas));
		expect(intensidades.find(([quantas]) => quantas === maiorContagem)?.[1]).toBe(maiorPasso);
		await expect(mapa.getByTestId("escala-do-mapa")).toBeVisible();
		await expect(mapa.getByTestId("escala-do-mapa")).toContainText("0");
	});

	test("no celular os links de motivo têm pelo menos 24px de altura", async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/");
		const caixa = await fila(page).getByRole("link", { name: "falha_sandbox" }).boundingBox();
		expect(caixa?.height).toBeGreaterThanOrEqual(24);
	});
});

test.describe("ajustes de leitura", () => {
	test("o rótulo da métrica tem a cor do texto e o apoio fica secundário", async ({ page }) => {
		await page.goto("/");
		const metrica = page.getByTestId("metrica").first();
		const corDoTexto = await page.evaluate(() => getComputedStyle(document.body).color);
		const cores = await metrica.evaluate((elemento) => {
			const cor = (seletor: string) => getComputedStyle(elemento.querySelector(seletor) as Element).color;
			return { rotulo: cor("dt"), apoio: cor("dd:last-child") };
		});
		expect(cores.rotulo).toBe(corDoTexto);
		expect(cores.apoio).not.toBe(corDoTexto);
	});

	test("o link do pull request tem peso maior que o texto da linha", async ({ page }) => {
		await page.goto("/");
		const link = fila(page).getByRole("link", { name: /aPRova-TESTES/ });
		expect(Number(await link.evaluate((elemento) => getComputedStyle(elemento).fontWeight))).toBeGreaterThanOrEqual(510);
	});

	test("o selo do título explica o que é risco alto", async ({ page }) => {
		await page.goto("/");
		await expect(fila(page).getByTestId("selo-risco-alto")).toHaveAttribute("title", /bloqueia o merge/);
	});

	test("a célula mais fraca do mapa de calor usa pelo menos 75% da cor do nível", async ({ page }) => {
		await page.goto("/");
		const forcas = await page
			.getByTestId("celula-do-mapa")
			.evaluateAll((celulas) =>
				celulas.filter((celula) => celula.getAttribute("data-passo") !== "0").map((celula) => Number(celula.getAttribute("data-forca"))),
			);
		expect(forcas.length).toBeGreaterThan(0);
		expect(Math.min(...forcas)).toBeGreaterThanOrEqual(75);
	});
});

test.describe("mapa de calor pelo teclado", () => {
	function mapa(pagina: Page) {
		return pagina.getByRole("region", { name: "Análises por dia e nível" });
	}

	test("a grade é um único ponto de tabulação e as células não são alvos", async ({ page }) => {
		await page.goto("/");
		const grade = mapa(page).getByRole("grid");
		await expect(grade).toHaveAttribute("tabindex", "0");
		await expect(mapa(page).locator("button")).toHaveCount(0);
		await expect(mapa(page).getByRole("gridcell")).toHaveCount(90);
	});

	test("o foco mostra o valor da célula ativa e as setas percorrem dias e níveis", async ({ page }) => {
		await page.goto("/");
		const grade = mapa(page).getByRole("grid");
		await grade.focus();
		const dica = mapa(page).getByRole("tooltip");
		await expect(dica).toHaveText(/^11 nov, alto: \d+ (análise|análises)$/);
		const ativa = await grade.getAttribute("aria-activedescendant");
		await expect(page.locator(`[id="${ativa}"]`)).toHaveAttribute("aria-label", /^11 nov, alto: /);

		await page.keyboard.press("ArrowLeft");
		await expect(dica).toHaveText(/^10 nov, alto: /);
		await page.keyboard.press("ArrowDown");
		await expect(dica).toHaveText(/^10 nov, médio: /);
		await page.keyboard.press("Home");
		await expect(dica).toHaveText(/^13 out, médio: /);
		await page.keyboard.press("Escape");
		await expect(mapa(page).getByRole("tooltip")).toHaveCount(0);
	});

	test("passar o ponteiro também mostra o valor", async ({ page }) => {
		await page.goto("/");
		await mapa(page).getByTestId("celula-do-mapa").nth(5).hover();
		await expect(mapa(page).getByRole("tooltip")).toHaveText(/^18 out, alto: /);
	});

	test("o quadrado do zero na escala tem a cor da célula vazia", async ({ page }) => {
		await page.goto("/");
		const zero = await mapa(page)
			.getByTestId("passo-da-escala")
			.first()
			.evaluate((elemento) => getComputedStyle(elemento).backgroundColor);
		const vazia = await mapa(page)
			.locator('[data-testid="celula-do-mapa"][data-passo="0"]')
			.first()
			.evaluate((elemento) => getComputedStyle(elemento).backgroundColor);
		expect(zero).toBe(vazia);
	});
});

test("em 768px a evidência aparece inteira abaixo do pull request", async ({ page }) => {
	await page.setViewportSize({ width: 768, height: 900 });
	await page.goto("/");
	const caminho = fila(page).getByText("requirements.txt", { exact: true }).filter({ visible: true });
	await expect(caminho).toHaveCount(1);
	const caixa = await caminho.boundingBox();
	expect(caixa?.width).toBeGreaterThan(80);
});
