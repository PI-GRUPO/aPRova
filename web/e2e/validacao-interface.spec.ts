import { expect, type Page, test } from "@playwright/test";

// Fixtures exclusivas da auditoria de interface; não autenticam usuários reais.
const instante = "2026-11-16T12:00:00Z";
const id = "00000000-0000-4000-8000-000000000003";
const recibo = "6d290152-af24-4462-a525-72adf3b22d6c";
const analise = {
	id,
	repositorio: "equipe/projeto",
	numero_pr: 12,
	sha_head: "a".repeat(40),
	autor: "agente[bot]",
	estado: "aguardando_aprovacao",
	nivel: "alto",
	recebido_em: instante,
	criado_em: instante,
	atualizado_em: instante,
};
const telas = [
	["visao-geral", "/", false],
	["analises", "/analises?installation_id=42", true],
	["detalhe", `/analises/${id}?installation_id=42`, true],
	["demonstracao", "/analises/demonstracao/00000000-0000-4000-8000-000000000001", false],
	["recibos", "/recibos", false],
	["recibo", `/recibos/${recibo}`, false],
	["repositorios", "/repositorios", false],
	["catalogo-real", "/repositorios", true],
	["regras", "/regras", false],
	["auditoria", "/auditoria", false],
	["configuracao", "/configuracao", false],
	["sugestoes", "/sugestoes", false],
	["conta", "/conta", true],
	["sla", "/sla", false],
	["relatorio", "/relatorio-semanal", true],
	["login", "/entrar", false],
	["404", "/endereco-inexistente-para-validacao", false],
] as const;

async function conferirInterface(page: Page) {
	await page.evaluate(() => document.fonts.ready);
	await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1);
	await expect(page.getByRole("main")).toHaveCount(1);
	await expect
		.poll(() =>
			page.locator("img:visible").evaluateAll((itens) =>
				itens.every((el) => {
					const img = el as HTMLImageElement;
					return img.complete && img.naturalWidth > 0;
				}),
			),
		)
		.toBeTruthy();
	const problemas = await page.evaluate(() => {
		const erros: string[] = [];
		if (document.documentElement.scrollWidth > innerWidth + 1) erros.push("Rolagem horizontal na página");
		const ids = [...document.querySelectorAll("[id]")].map((el) => el.id);
		if (new Set(ids).size !== ids.length) erros.push("IDs duplicados");
		const visivel = (el: Element) => el.getClientRects().length > 0 && getComputedStyle(el).visibility !== "hidden";
		const nome = (el: Element) =>
			el.getAttribute("aria-label") ||
			el.getAttribute("title") ||
			el.textContent?.trim() ||
			el
				.getAttribute("aria-labelledby")
				?.split(" ")
				.map((id) => document.getElementById(id)?.textContent)
				.join(" ");
		for (const el of document.querySelectorAll("button,a[href]")) {
			if (visivel(el) && !nome(el)) erros.push(`Controle sem nome: ${el.tagName}`);
		}
		for (const el of document.querySelectorAll<HTMLInputElement>("input:not([type=hidden]),textarea,select")) {
			if (visivel(el) && !el.labels?.length && !el.getAttribute("aria-label") && !el.getAttribute("aria-labelledby"))
				erros.push(`Campo sem rótulo: ${el.outerHTML.slice(0, 80)}`);
		}
		for (const img of document.querySelectorAll<HTMLImageElement>("img")) {
			// Zoom da arte 404 pode arredondar dimensões SVG fracionárias em um pixel.
			if (visivel(img) && (Math.abs(img.width - img.naturalWidth) > 1 || Math.abs(img.height - img.naturalHeight) > 1))
				erros.push(`Ativo deformado: ${img.getAttribute("src")}`);
		}
		for (const el of document.querySelectorAll<HTMLAnchorElement>("a[href]")) {
			if (!el.getAttribute("href")?.startsWith("#") && !["http:", "https:"].includes(new URL(el.href).protocol))
				erros.push("Protocolo de link inválido");
		}
		return erros;
	});
	expect(problemas).toEqual([]);
}

for (const largura of [1440, 1024, 768, 390, 320])
	for (const tema of ["escuro", "claro"]) {
		test(`auditoria transversal em ${largura}px e ${tema}`, async ({ page }) => {
			test.setTimeout(120000);
			await page.setViewportSize({ width: largura, height: largura >= 768 ? 900 : 844 });
			await page.addInitScript((t) => localStorage.setItem("aprova-tema", t), tema);
			let autenticado = false;
			const erros: string[] = [];
			page.on("pageerror", (erro) => erros.push(erro.message));
			await page.route("**/api/sessao", (r) =>
				r.fulfill({
					json: autenticado
						? {
								autenticado: true,
								login_disponivel: true,
								usuario: { id: 7, login: "pessoa-fixture" },
								instalacoes: [{ id: 42, conta: "equipe" }],
								csrf_token: "a".repeat(43),
								expira_em: new Date(Date.now() + 3600000).toISOString(),
							}
						: { autenticado: false, login_disponivel: true },
				}),
			);
			await page.route("**/api/repositorios?*", (r) => r.fulfill({ json: { installation_id: 42, repositorios: ["equipe/projeto"] } }));
			await page.route("**/api/analises**", (r) =>
				r.fulfill({
					json:
						new URL(r.request().url()).pathname === "/api/analises"
							? {
									installation_id: 42,
									snapshot: instante,
									pagina: 1,
									limite: 20,
									total: 1,
									tem_mais: false,
									itens: [analise],
								}
							: {
									installation_id: 42,
									analise,
									branch: null,
									url_pr: "https://github.com/equipe/projeto/pull/12",
									url_check: null,
									atual: true,
									gate: null,
									etapas: [],
									tentativas: [],
									aprovacoes: [],
									recibo_id: null,
									testes_estruturados_disponiveis: false,
									rede_isolada: null,
								},
				}),
			);
			await page.route("**/api/web/relatorio-semanal?*", (r) =>
				r.fulfill({
					json: {
						installation_id: 42,
						repositorios: ["equipe/projeto"],
						inicio: "2026-11-09T00:00:00Z",
						fim: "2026-11-16T00:00:00Z",
						total_execucoes: 10,
						concluidas: 6,
						rejeitadas: 1,
						falhas: 1,
						em_andamento: 2,
						taxa_sucesso: 0.75,
						exigiram_aprovacao_humana: 3,
						risco_medio: 2,
						revertidas: 1,
					},
				}),
			);
			await page.route("**/receipts/**", (r) =>
				r.fulfill({
					json: r.request().url().endsWith("/keys")
						? { chaves: [] }
						: {
								id: recibo,
								task_run_id: id,
								key_id: "chave-teste",
								hash_sha256: "a".repeat(64),
								payload_base64: "YQ==",
								assinatura_base64: "Yg==",
								recibo_anterior_id: null,
								payload: { repositorio: "equipe/projeto", numero_pr: 12, head_sha: analise.sha_head, emitido_em: instante, nivel: "alto" },
							},
				}),
			);
			for (const [nome, caminho, sessao] of telas) {
				await test.step(nome, async () => {
					autenticado = sessao;
					await page.goto(caminho);
					await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
					await expect(page.getByText(/^(Carregando|Confirmando)/)).toHaveCount(0);
					await conferirInterface(page);
					if ([1440, 390, 320].includes(largura))
						await page.screenshot({ path: `../.capturas-mcp/validacao-final/local-${nome}-${largura}-${tema}.png`, fullPage: true });
				});
			}
			expect(erros).toEqual([]);
		});
	}

for (const largura of [1440, 320])
	for (const tema of ["escuro", "claro"]) {
		test(`modais: teclado, foco e rolagem em ${largura}px e ${tema}`, async ({ page }) => {
			test.setTimeout(60000);
			await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 568 });
			await page.addInitScript((t) => localStorage.setItem("aprova-tema", t), tema);
			await page.route("**/api/sessao", (r) => r.fulfill({ json: { autenticado: false, login_disponivel: false } }));
			await page.route("**/receipts/keys", (r) => r.fulfill({ json: { chaves: [] } }));
			const casos = [
				["configuracao", "/configuracao", "Adicionar caminho"],
				["decisao", "/analises/demonstracao/00000000-0000-4000-8000-000000000001", "Aprovar"],
				["alteracao", "/analises/demonstracao/00000000-0000-4000-8000-000000000001", "Pedir alterações"],
				["exportacao", "/recibos", "Exportar JSON"],
				["cadeia", "/recibos", "Verificar cadeia"],
			] as const;
			for (const [nome, caminho, rotulo] of casos) {
				await page.goto(caminho);
				if (largura < 768 && caminho === "/recibos") await page.getByText("Período, cadeia e exportação", { exact: true }).click();
				const gatilho = page.getByRole("button", { name: rotulo, exact: true });
				await gatilho.click();
				const dialogo = page.getByRole("dialog");
				await expect(dialogo).toBeVisible();
				await page.evaluate(() => document.fonts.ready);
				const geometria = await dialogo.boundingBox();
				if (!geometria) throw new Error("Modal sem geometria");
				expect(geometria.x).toBeGreaterThanOrEqual(15);
				expect(geometria.y).toBeGreaterThanOrEqual(15);
				expect(geometria.x + geometria.width).toBeLessThanOrEqual(largura - 15);
				expect(await dialogo.evaluate((el) => el.scrollWidth <= el.clientWidth + 1)).toBeTruthy();
				for (const tecla of ["Tab", "Tab", "Tab", "Tab", "Tab", "Tab", "Shift+Tab", "Shift+Tab"]) {
					await page.keyboard.press(tecla);
					expect(await dialogo.evaluate((el) => el.contains(document.activeElement))).toBeTruthy();
				}
				await page.screenshot({ path: `../.capturas-mcp/validacao-final/modal-${nome}-${largura}-${tema}.png` });
				await page.keyboard.press("Escape");
				await expect(dialogo).toHaveCount(0);
				await expect(gatilho).toBeFocused();
			}
		});
	}
