import { expect, test } from "@playwright/test";
import { autenticarFixture } from "./sessao-dashboard";

test.beforeEach(async ({ page }) => {
	await autenticarFixture(page);
	await page.route("**/api/repositorios?*", (r) => r.fulfill({ json: { installation_id: 42, repositorios: [] } }));
	await page.route("**/api/analises?*", (r) =>
		r.fulfill({
			json: { installation_id: 42, itens: [], total: 0, pagina: 1, limite: 20, tem_mais: false, snapshot: new Date().toISOString() },
		}),
	);
});
for (const largura of [1440, 390, 360])
	for (const tema of ["escuro", "claro"]) {
		test(`seleção do mapa em ${largura}px e tema ${tema}`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
			await page.addInitScript((t) => {
				localStorage.setItem("aprova-tema", t);
			}, tema);
			const proibidas: string[] = [];
			page.on("request", (r) => {
				const caminho = new URL(r.url()).pathname;
				if (r.method() !== "GET" || caminho.startsWith("/api/analises/") || caminho.startsWith("/receipts/")) proibidas.push(r.url());
			});
			await page.goto("/?fila=go.mod&repositorio=cobranca-api");
			for (const [dia, quantidade, primeiro] of [
				["2026-11-08", 2, "cobranca-api"],
				["2026-11-11", 3, "painel-entregas"],
			] as const) {
				const celula = page.locator(`button.celula-mapa[data-dia="${dia}"][data-nivel="alto"]`);
				await celula.focus();
				await page.keyboard.press("Enter");
				const modal = page.getByRole("dialog");
				await expect(modal.getByRole("link")).toHaveCount(quantidade);
				await expect(modal).toContainText("Dados simulados");
				await expect(modal.getByRole("link").first()).toContainText(primeiro);
				await page.evaluate(() => document.fonts.ready);
				await modal.screenshot({ path: `../.capturas-mcp/mapa/local-${dia}-${largura}-${tema}.png` });
				expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
				expect(await modal.evaluate((el) => el.contains(document.activeElement))).toBeTruthy();
				await modal.getByRole("link").first().focus();
				await page.keyboard.press("Enter");
				await expect(page).toHaveURL((url) => url.pathname === `/analises/demonstracao/mapa-${dia}-alto-1`);
				await expect(page.locator("main")).toContainText(/dados simulados/i);
				await expect(page.getByRole("heading", { level: 1 })).toContainText(primeiro);
				await expect(page.locator(".estado-do-detalhe")).toContainText("risco alto");
				await page.getByRole("link", { name: "Voltar à visão geral", exact: true }).click();
				await expect(page.getByRole("dialog").getByRole("link")).toHaveCount(quantidade);
				await expect(page).toHaveURL(`/?fila=go.mod&repositorio=cobranca-api&mapa_dia=${dia}&mapa_nivel=alto`);
				await page.keyboard.press("Escape");
				await expect(page.getByRole("dialog")).toHaveCount(0);
				await expect(celula).toBeFocused();
			}
			expect(proibidas).toEqual([]);
		});
	}
test("célula vazia, seleção única, reload e parâmetros inválidos são distintos", async ({ page }) => {
	await page.goto("/?mapa_dia=2026-10-13&mapa_nivel=alto");
	await expect(page.getByRole("dialog")).toContainText("Nenhuma análise neste recorte");
	await expect(page.getByRole("dialog").getByRole("link")).toHaveCount(0);
	await page.getByRole("button", { name: "Fechar recorte", exact: true }).click();
	await page.locator('button.celula-mapa[data-dia="2026-10-15"][data-nivel="medio"]').click();
	await expect(page.getByRole("dialog").getByRole("link")).toHaveCount(1);
	await page.reload();
	await expect(page.getByRole("dialog").getByRole("link")).toHaveCount(1);
	await page.getByRole("dialog").getByRole("link").click();
	await expect(page.locator(".estado-do-detalhe")).toContainText("risco médio");
	await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
	await page.goto("/?fila=preservado&mapa_dia=2026-02-31&mapa_nivel=outro");
	await expect(page.getByText("Recorte do mapa inválido. Escolha um dia e nível da janela apresentada.")).toBeVisible();
	await expect(page.getByRole("dialog")).toHaveCount(0);
	await page.getByRole("button", { name: "Limpar recorte" }).click();
	await expect(page).toHaveURL(/\/?fila=preservado$/);
});
test("todas as contagens correspondem a referências do mesmo dia e risco", async ({ page }) => {
	await page.goto("/");
	const validacao = await page.evaluate(async () => {
		const mapa = await import("/src/servicos/mapa");
		const visao = await import("/src/servicos/visao-geral");
		const analises = await import("/src/servicos/analises-demonstracao");
		const sinal = new AbortController().signal;
		const dados = await visao.consultarVisaoGeral({ instalacao: "42", repositorio: "" }, new URLSearchParams(), sinal);
		const problemas: string[] = [];
		let total = 0;
		const ids = new Set<string>();
		for (const dia of dados.mapa)
			for (const [nivel, quantidade] of [
				["alto", dia.alto],
				["medio", dia.medio],
				["baixo", dia.baixo],
			] as const) {
				const recorte = await mapa.consultarRecorteMapa("42", dia.data, nivel, sinal);
				if (recorte.itens.length !== quantidade) problemas.push(`contagem:${dia.data}:${nivel}`);
				for (const item of recorte.itens) {
					const detalhe = await analises.consultarAnaliseDemonstracao(item.id, sinal);
					if (
						detalhe?.origem !== "mock" ||
						detalhe.dados.analise.nivel !== nivel ||
						detalhe.dados.analise.recebido_em.slice(0, 10) !== dia.data ||
						detalhe.dados.analise.repositorio !== item.repositorio ||
						detalhe.dados.analise.numero_pr !== item.pr ||
						ids.has(item.id)
					)
						problemas.push(item.id);
					ids.add(item.id);
					total++;
				}
			}
		const falhas: string[] = [];
		for (const data of ["2026-02-31", "2099-01-01"]) {
			try {
				await mapa.consultarRecorteMapa("42", data, "alto", sinal);
			} catch {
				falhas.push(data);
			}
		}
		const cancelado = new AbortController();
		cancelado.abort();
		try {
			await mapa.consultarRecorteMapa("42", "2026-11-08", "alto", cancelado.signal);
		} catch {
			falhas.push("cancelado");
		}
		return { problemas, total, falhas };
	});
	expect(validacao.problemas).toEqual([]);
	expect(validacao.total).toBeGreaterThan(10);
	expect(validacao.falhas).toEqual(["2026-02-31", "2099-01-01", "cancelado"]);
});
test("falha ao carregar o recorte permite retry e nunca consulta dados reais", async ({ page }) => {
	let falhar = true;
	const proibidas: string[] = [];
	page.on("request", (r) => {
		if (new URL(r.url()).pathname.startsWith("/api/analises/")) proibidas.push(r.url());
	});
	await page.route("**/src/mocks/mapa.ts*", (r) => {
		if (falhar) {
			falhar = false;
			return r.abort();
		}
		return r.continue();
	});
	await page.goto("/?mapa_dia=2026-11-08&mapa_nivel=alto");
	await expect(page.getByRole("alert")).toContainText("Não foi possível carregar as referências");
	await page.getByRole("button", { name: "Tentar novamente", exact: true }).click();
	await expect(page.getByRole("dialog").getByRole("link")).toHaveCount(2);
	expect(proibidas).toEqual([]);
});
test("a decisão do detalhe selecionado permanece indisponível", async ({ page }) => {
	const escritas: string[] = [];
	page.on("request", (r) => {
		if (r.method() !== "GET") escritas.push(r.url());
	});
	await page.goto("/?mapa_dia=2026-11-11&mapa_nivel=alto");
	await page.getByRole("dialog").getByRole("link").first().click();
	await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
	expect(escritas).toEqual([]);
});
