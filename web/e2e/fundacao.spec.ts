import { expect, test } from "@playwright/test";

const rotas = [
	"/",
	"/analises",
	"/analises/00000000-0000-0000-0000-000000000001",
	"/recibos",
	"/recibos/00000000-0000-0000-0000-000000000001",
	"/repositorios",
	"/regras",
	"/auditoria",
	"/configuracao",
	"/sugestoes",
	"/conta",
	"/sla",
	"/entrar",
];
for (const largura of [1440, 390, 360]) {
	test(`navegação, temas e geometria em ${largura}px`, async ({ page }) => {
		test.setTimeout(60000);
		await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
		await page.goto("/endereco-inexistente");
		await expect(page.getByRole("heading", { name: "Página não encontrada", exact: true })).toBeVisible();
		await page.evaluate(() => document.fonts.ready);
		await page.screenshot({ path: `../.capturas-mcp/f1/local-404-${largura}-escuro.png`, fullPage: true });
		await page.locator(".arte-404").screenshot({ path: `../.capturas-mcp/f1/local-composicao-${largura}.png` });
		const imagens = await page.locator("img:visible").evaluateAll((imgs) =>
			imgs.map((elemento) => {
				if (!(elemento instanceof HTMLImageElement)) throw new Error("Ativo sem elemento de imagem");
				return {
					ok: elemento.complete && elemento.naturalWidth > 0,
					w: elemento.width,
					h: elemento.height,
					naturalW: elemento.naturalWidth,
					naturalH: elemento.naturalHeight,
				};
			}),
		);
		expect(imagens.filter((i) => !i.ok || Math.abs(i.w - i.naturalW) > 1 || Math.abs(i.h - i.naturalH) > 1)).toEqual([]);
		await page.getByRole("button", { name: "Usar tema claro" }).click();
		await expect(page.locator("html")).toHaveAttribute("data-tema", "claro");
		await page.screenshot({ path: `../.capturas-mcp/f1/local-404-${largura}-claro.png`, fullPage: true });
		await page.reload();
		await expect(page.locator("html")).toHaveAttribute("data-tema", "claro");
		expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
		if (largura < 768) {
			await page.getByRole("button", { name: "Menu completo" }).click();
			await expect(page.getByRole("dialog", { name: "Navegação" })).toBeVisible();
		}
		const menu = page.getByRole("navigation", { name: largura < 768 ? "Todas as telas" : "Principal", exact: true });
		await expect(menu).toBeVisible();
		for (const nome of ["Configuração do gate", "Sugestões da IA", "Limites de SLA"]) {
			await expect(menu.getByRole("link", { name: nome, exact: true })).toHaveCount(0);
		}
		await expect(menu.getByRole("link", { name: "Conta e integrações", exact: true })).toBeVisible();
		if (largura < 768) {
			await menu.getByRole("link", { name: "Recibos", exact: true }).click();
			await expect(page).toHaveURL(/\/recibos$/);
			await expect(page.getByRole("dialog")).not.toBeVisible();
		}
		for (const rota of rotas) {
			await page.goto(rota);
			await expect(page.getByRole("heading", { level: 1 })).not.toHaveText("Página não encontrada");
			await expect(page.getByText(/Azulpasta|@julia-ed2|41 em 30 dias/)).toHaveCount(0);
		}
		await page.goto("/analises");
		await page.goto("/recibos");
		await page.goBack();
		await expect(page).toHaveURL(/\/analises$/);
		await page.reload();
		await expect(page.getByRole("heading", { name: "Análises", exact: true })).toBeVisible();
	});
}

test("modal valida o formulário, mantém o foco e retorna ao gatilho", async ({ page }) => {
	await page.goto("/__teste");
	const abrir = page.getByRole("button", { name: "Abrir formulário" });
	await abrir.click();
	const campo = page.getByRole("textbox", { name: "Justificativa de teste" });
	await expect(campo).toBeFocused();
	await page.getByRole("button", { name: "Confirmar", exact: true }).click();
	await expect(page.getByRole("alert")).toHaveText("Informe um valor.");
	await campo.fill("Validação de comportamento.");
	await page.keyboard.press("Shift+Tab");
	await expect(page.getByRole("dialog").locator(":focus")).toHaveCount(1);
	await page.keyboard.press("Escape");
	await expect(page.getByRole("dialog")).not.toBeVisible();
	await expect(abrir).toBeFocused();
	await abrir.click();
	await page.getByRole("button", { name: "Confirmar", exact: true }).click();
	await expect(page.getByRole("dialog")).not.toBeVisible();
});

test("filtros, limpeza e ordenação da tabela", async ({ page }) => {
	await page.goto("/__teste");
	await page.getByLabel("Buscar cenário").fill("Cenário B");
	await expect(page.getByRole("cell", { name: "Cenário A", exact: true })).toHaveCount(0);
	await page.getByRole("button", { name: "Limpar filtros" }).click();
	await page.getByRole("combobox", { name: "Grupo", exact: true }).click();
	await page.getByRole("option", { name: "Somente A" }).click();
	await expect(page.getByRole("cell", { name: "Cenário B", exact: true })).toHaveCount(0);
	await page.getByRole("button", { name: "Limpar filtros" }).click();
	await page.getByRole("button", { name: "Nome" }).click();
	await expect(page.getByRole("columnheader", { name: "Nome" })).toHaveAttribute("aria-sort", "descending");
	await expect(page.locator("tbody tr").first()).toContainText("Cenário B");
});

test("cliente trata saúde, prontidão 503 e falha de rede", async ({ page }) => {
	await page.route("**/health", (route) => route.fulfill({ json: { status: "ok" } }));
	await page.route("**/ready", (route) =>
		route.fulfill({ status: 503, json: { status: "indisponivel", dependencias: { postgres: "indisponivel" } } }),
	);
	await page.route("**/receipts/keys", (route) => route.abort());
	await page.goto("/__teste");
	await page.getByRole("button", { name: "Consultar health", exact: true }).click();
	await expect(page.getByRole("status")).toHaveText("Serviço disponível.");
	await page.getByRole("button", { name: "Consultar ready", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("503");
	await page.getByRole("button", { name: "Consultar rede", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("rede:");
});

test("cliente só consulta rotas cadastradas e mantém filtros na query", async ({ page }) => {
	await page.goto("/__teste");
	const resultado = await page.evaluate(async () => {
		const { consultarJSON } = await import("/src/servicos/http");
		const destinos: string[] = [];
		window.fetch = async (url) => {
			destinos.push(String(url));
			return Response.json({});
		};
		const rejeicoes = await Promise.allSettled([
			consultarJSON("/api/rota-externa-ao-cadastro" as Parameters<typeof consultarJSON>[0]),
			consultarJSON("recibo", { id: "identificador-invalido" }),
		]);
		const id = "00000000-0000-0000-0000-000000000001";
		await consultarJSON("analise", { id, parametros: new URLSearchParams({ repositorio: "org/repo & teste" }) });
		return { rejeicoes: rejeicoes.map((r) => r.status), destinos };
	});
	expect(resultado.rejeicoes).toEqual(["rejected", "rejected"]);
	expect(resultado.destinos).toEqual([
		`${new URL(page.url()).origin}/api/analises/00000000-0000-0000-0000-000000000001?repositorio=org%2Frepo+%26+teste`,
	]);
});

test("cliente valida DTOs e rejeita HTML e JSON malformado", async ({ page }) => {
	await page.route("**/health", (route) => route.fulfill({ json: { status: "inventado" } }));
	await page.route("**/ready", (route) => route.fulfill({ contentType: "text/html", path: "e2e/fixtures/login.html" }));
	await page.route("**/receipts/keys", (route) => route.fulfill({ contentType: "application/json", body: "{" }));
	await page.goto("/__teste");
	await page.getByRole("button", { name: "Consultar health", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("Resposta de saúde inválida");
	await page.getByRole("button", { name: "Consultar ready", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("não retornou JSON");
	await page.getByRole("button", { name: "Consultar rede", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("resposta:");
});

test("cliente distingue falha ao preparar a requisição de falha de rede", async ({ page }) => {
	await page.goto("/__teste");
	const resultado = await page.evaluate(async () => {
		const { consultarJSON, ErroHTTP } = await import("/src/servicos/http");
		let enviou = false;
		window.fetch = async () => {
			enviou = true;
			return Response.json({});
		};
		try {
			await consultarJSON("sair", { metodo: "POST", corpo: { valor: BigInt(1) } });
			return { enviou, tipo: "sucesso" };
		} catch (erro) {
			return { enviou, tipo: erro instanceof ErroHTTP ? erro.tipo : "desconhecido" };
		}
	});
	expect(resultado).toEqual({ enviou: false, tipo: "requisicao" });
});

test("timeout e cancelamento não viram sucesso", async ({ page }) => {
	await page.route("**/receipts/keys", (route) => {
		// Mantém o GET interceptado para exercitar timeout e cancelamento do cliente.
		expect(route.request().method()).toBe("GET");
	});
	await page.goto("/__teste");
	await page.getByRole("button", { name: "Consultar timeout", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("timeout:");
	await page.getByRole("button", { name: "Consultar rede", exact: true }).click();
	await page.getByRole("button", { name: "Cancelar consulta", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("cancelado:");
});

test("401 privado volta para entrar", async ({ page }) => {
	await page.route("**/api/sessao", (route) => route.fulfill({ json: { autenticado: false, login_disponivel: true } }));
	await page.route("**/receipts/keys", (route) => route.fulfill({ status: 401 }));
	await page.goto("/__teste");
	await page.getByRole("button", { name: "Consultar privada", exact: true }).click();
	await expect(page).toHaveURL(/\/entrar$/);
	await expect(page.getByRole("heading", { name: /Acompanhe decisões/ })).toBeVisible();
});

test("teclado alcança o link de pular e tema sem armazenamento", async ({ page }) => {
	await page.addInitScript(() => {
		Object.defineProperty(window, "localStorage", {
			get() {
				throw new Error("indisponível");
			},
		});
	});
	await page.goto("/");
	await page.keyboard.press("Tab");
	await expect(page.getByRole("link", { name: "Pular para o conteúdo" })).toBeFocused();
	await page.keyboard.press("Enter");
	await expect(page.locator("main")).toBeFocused();
	await page.getByRole("button", { name: "Usar tema claro" }).click();
	await expect(page.locator("html")).toHaveAttribute("data-tema", "claro");
});

test("timeout durante a leitura do corpo preserva o tipo do erro", async ({ page }) => {
	await page.addInitScript(() => {
		const original = window.fetch.bind(window);
		window.fetch = (url, opcoes) => {
			if (!String(url).endsWith("/receipts/keys")) return original(url, opcoes);
			const corpo = new ReadableStream({
				start(controlador) {
					controlador.enqueue(new TextEncoder().encode('{"status":'));
					opcoes?.signal?.addEventListener(
						"abort",
						() => {
							controlador.error(new DOMException("Cancelado", "AbortError"));
						},
						{ once: true },
					);
				},
			});
			return Promise.resolve(new Response(corpo, { headers: { "Content-Type": "application/json" } }));
		};
	});
	await page.goto("/__teste");
	await page.getByRole("button", { name: "Consultar timeout", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("timeout:");
});
