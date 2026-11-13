import { readFileSync } from "node:fs";
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
	test("mostra quatro números, com barra só nas porcentagens", async ({ page }) => {
		await page.goto("/");
		const metricas = page.getByTestId("metrica");
		await expect(metricas).toHaveCount(4);
		for (const [rotulo, valor, temBarra] of [
			["Espera mais antiga", "2 d 2 h", false],
			["Taxa de bloqueio", "47%", true],
			["Falhas no sandbox", "31%", true],
			["Análises recebidas", "41", false],
		] as const) {
			const metrica = metricas.filter({ hasText: rotulo });
			await expect(metrica.getByTestId("metrica-valor")).toHaveText(valor);
			await expect(metrica.getByTestId("proporcao")).toHaveCount(temBarra ? 1 : 0);
		}
	});

	test("a espera mais antiga diz qual PR é e quantos esperam", async ({ page }) => {
		await page.goto("/");
		const metrica = page.getByTestId("metrica").filter({ hasText: "Espera mais antiga" });
		await expect(metrica).toContainText("aPRova-TESTES #1");
		await expect(metrica).toContainText("1 de 5 na fila");
		await expect(page.getByText(/de 15 bloqueadas/)).toHaveCount(0);
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
		const bases = await Promise.all((await paineis.all()).slice(0, 2).map(baseDe));
		expect(Math.abs((bases[0] ?? 0) - (bases[1] ?? 0))).toBeLessThanOrEqual(1);
		const larguraDaFaixa = (await page.getByTestId("faixa-de-metricas").boundingBox())?.width ?? 0;
		for (const bloco of [fila(page), page.getByRole("region", { name: "Análises por dia e nível" })]) {
			const largura = (await bloco.boundingBox())?.width ?? 0;
			expect(Math.abs(largura - larguraDaFaixa)).toBeLessThanOrEqual(1);
		}
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

	test("a grade é um único ponto de tabulação", async ({ page }) => {
		await page.goto("/");
		const grade = mapa(page).getByRole("grid");
		await expect(grade).toHaveAttribute("tabindex", "0");
		await expect(mapa(page).getByRole("gridcell")).toHaveCount(90);
		const focaveisDentro = await grade
			.locator("a, button")
			.evaluateAll((elementos) => elementos.filter((elemento) => elemento.getAttribute("tabindex") !== "-1").length);
		expect(focaveisDentro).toBe(0);
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

test.describe("estados honestos", () => {
	test("carregando não afirma hora de atualização", async ({ page }) => {
		await page.goto("/?estado=carregando");
		await expect(page.getByRole("main").getByText(/Atualizando/)).toBeVisible();
		await expect(page.getByRole("main").getByText(/atualizado às/)).toHaveCount(0);
	});

	test("o erro diz a última atualização boa e não promete o que não sabe", async ({ page }) => {
		await page.goto("/?estado=erro");
		const principal = page.getByRole("main");
		await expect(principal.getByText(/Última atualização às 14:00/)).toBeVisible();
		await expect(principal.getByText(/Nenhuma análise foi perdida/)).toHaveCount(0);
		await expect(principal.getByRole("alert")).toContainText("o bloqueio no GitHub não depende desta tela");
	});

	test("o esqueleto tem a forma da tela e não faz a página pular", async ({ page }) => {
		await page.goto("/");
		const faixa = await page.getByTestId("faixa-de-metricas").boundingBox();
		const fila = await page.getByRole("region", { name: "Aguardando aprovação" }).boundingBox();
		await page.goto("/?estado=carregando");
		const faixaDoEsqueleto = await page.getByTestId("esqueleto-da-faixa").boundingBox();
		const filaDoEsqueleto = await page.getByTestId("esqueleto-da-fila").boundingBox();
		expect(Math.abs((faixaDoEsqueleto?.height ?? 0) - (faixa?.height ?? 0))).toBeLessThanOrEqual(8);
		expect(Math.abs((filaDoEsqueleto?.y ?? 0) - (fila?.y ?? 0))).toBeLessThanOrEqual(8);
		expect(Math.abs((filaDoEsqueleto?.height ?? 0) - (fila?.height ?? 0))).toBeLessThanOrEqual(24);
	});
});

test.describe("evidência e busca", () => {
	test("em 1440px o nome do arquivo da evidência aparece inteiro", async ({ page }) => {
		await page.goto("/");
		const arquivo = fila(page).getByText("000031_parcelas.down.sql", { exact: true }).filter({ visible: true });
		await expect(arquivo).toHaveCount(1);
		const cortado = await arquivo.evaluate((elemento) => elemento.scrollWidth > elemento.clientWidth);
		expect(cortado).toBe(false);
	});

	test("a busca procura na evidência e aceita espaço no lugar do sublinhado", async ({ page }) => {
		await page.goto("/");
		const busca = fila(page).getByRole("searchbox", { name: "Buscar na fila" });
		await busca.fill("migrations");
		await expect(linhasDaFila(page)).toHaveCount(1);
		await busca.fill("falha sandbox");
		await expect(linhasDaFila(page)).toHaveCount(1);
		await expect(linhasDaFila(page)).toContainText("aPRova-TESTES");
	});
});

test.describe("recibo e vocabulário", () => {
	test("uma linha discreta mostra o último recibo assinado", async ({ page }) => {
		await page.goto("/");
		const recibo = page.getByRole("region", { name: "Último recibo assinado" });
		await expect(recibo).toContainText("f27414ff");
		await expect(recibo).toContainText("cobranca-api #218");
		await expect(recibo).toContainText("Ed25519");
		await expect(recibo.getByRole("link", { name: "Ver os recibos" })).toHaveAttribute("href", "/recibos");
	});

	test("os rótulos dizem o que contam", async ({ page }) => {
		await page.goto("/");
		await expect(page.getByText("Alterações pedidas")).toBeVisible();
		await expect(page.getByText("Com alterações")).toHaveCount(0);
		await expect(page.getByText(/9 em andamento ou paradas antes do gate/)).toBeVisible();
		await expect(page.getByText(/cada tentativa conta/)).toBeVisible();
		await expect(page.getByRole("region", { name: "Bloqueadas" })).toContainText("15 no período, por desfecho");
	});
});

interface AnaliseDosDados {
	id: string;
	received_at: string;
	risk_gate: { level: string } | null;
}

const analisesDosDados: AnaliseDosDados[] = JSON.parse(readFileSync(new URL("../src/dados/analises.json", import.meta.url), "utf8"));
const meses = ["jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"];
const nivelPorNome: Record<string, string> = { alto: "alto", médio: "medio", baixo: "baixo" };

function diaEMesEmBrasilia(data: string) {
	const partes = new Intl.DateTimeFormat("en-CA", {
		timeZone: "America/Sao_Paulo",
		year: "numeric",
		month: "numeric",
		day: "numeric",
	}).formatToParts(new Date(data));
	const valor = (tipo: string) => Number(partes.find((parte) => parte.type === tipo)?.value);
	return `${valor("day")} ${meses[valor("month") - 1]}`;
}

function idsDaCelula(rotulo: string) {
	const [dia, resto] = rotulo.split(", ");
	const nivel = nivelPorNome[(resto ?? "").split(":")[0] ?? ""];
	return analisesDosDados
		.filter((analise) => analise.risk_gate?.level === nivel && diaEMesEmBrasilia(analise.received_at) === dia)
		.map((analise) => `/analises/${analise.id}`)
		.sort();
}

test.describe("mapa de calor leva às análises", () => {
	function mapa(pagina: Page) {
		return pagina.getByRole("region", { name: "Análises por dia e nível" });
	}

	test("a célula com uma análise leva exatamente a ela", async ({ page }) => {
		await page.goto("/");
		const celula = mapa(page).locator('[data-testid="celula-do-mapa"][data-quantas="1"]').first();
		const rotulo = (await celula.getAttribute("aria-label")) ?? "";
		const [esperado] = idsDaCelula(rotulo);
		expect(esperado).toBeTruthy();
		await celula.click();
		await expect(page).toHaveURL(new RegExp(`${esperado}$`));
	});

	test("a célula com várias análises lista cada uma com o link exato", async ({ page }) => {
		await page.goto("/");
		const celula = mapa(page).locator('[data-testid="celula-do-mapa"][data-quantas="3"]').first();
		const rotulo = (await celula.getAttribute("aria-label")) ?? "";
		await celula.click();
		const lista = page.getByRole("dialog", { name: rotulo });
		await expect(lista).toBeVisible();
		const destinos = (await lista.getByRole("link").evaluateAll((links) => links.map((link) => link.getAttribute("href") ?? ""))).sort();
		expect(destinos).toEqual(idsDaCelula(rotulo));
		expect(destinos).toHaveLength(3);
	});

	test("a célula vazia não leva a lugar nenhum", async ({ page }) => {
		await page.goto("/");
		const vazia = mapa(page).locator('[data-testid="celula-do-mapa"][data-quantas="0"]').first();
		await expect(vazia.locator("a, button")).toHaveCount(0);
	});

	test("Enter na célula ativa abre a análise pelo teclado", async ({ page }) => {
		await page.goto("/");
		const grade = mapa(page).getByRole("grid");
		await grade.focus();
		await page.keyboard.press("ArrowLeft");
		const ativa = page.locator(`[id="${await grade.getAttribute("aria-activedescendant")}"]`);
		await expect(ativa).toHaveAttribute("data-quantas", "1");
		const [esperado] = idsDaCelula((await ativa.getAttribute("aria-label")) ?? "");
		await page.keyboard.press("Enter");
		await expect(page).toHaveURL(new RegExp(`${esperado}$`));
	});

	for (const largura of [390, 1440]) {
		test(`em ${largura}px as células clicáveis têm pelo menos 24px e a página não rola de lado`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: 900 });
			await page.goto("/");
			const caixas = await mapa(page)
				.locator('[data-testid="celula-do-mapa"]:not([data-quantas="0"])')
				.evaluateAll((celulas) =>
					celulas.map((celula) => celula.getBoundingClientRect()).map(({ width, height }) => Math.min(width, height)),
				);
			expect(Math.min(...caixas)).toBeGreaterThanOrEqual(24);
			expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(largura);
			const hoje = await mapa(page).getByText("hoje", { exact: true }).boundingBox();
			expect(hoje?.x ?? -1).toBeGreaterThanOrEqual(0);
			expect((hoje?.x ?? 0) + (hoje?.width ?? 0)).toBeLessThanOrEqual(largura);
		});
	}
});
