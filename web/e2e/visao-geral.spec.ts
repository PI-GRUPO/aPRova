import { expect, test } from "@playwright/test";
import { autenticarFixture } from "./sessao-dashboard";

const snapshot = "2026-11-16T10:00:00Z";
const item = {
	id: "00000000-0000-4000-8000-000000000011",
	repositorio: "equipe/projeto",
	numero_pr: 18,
	sha_head: "abc1234",
	autor: "revisora",
	estado: "aguardando_aprovacao",
	nivel: "alto",
	recebido_em: snapshot,
	criado_em: snapshot,
	atualizado_em: snapshot,
};
const itens = Array.from({ length: 26 }, (_, indice) => ({
	...item,
	id: `00000000-0000-4000-8000-${String(indice + 11).padStart(12, "0")}`,
	numero_pr: item.numero_pr + indice,
}));
const itemDaSegundaPagina = itens[25];
const dados = { installation_id: 42, snapshot, pagina: 1, limite: 25, total: itens.length, tem_mais: true, itens: itens.slice(0, 25) };
test.beforeEach(async ({ page }) => {
	await autenticarFixture(page);
	await page.route("**/api/repositorios?*", (r) => r.fulfill({ json: { installation_id: 42, repositorios: [item.repositorio] } }));
	await page.route("**/api/analises?*", (r) => {
		const q = new URL(r.request().url()).searchParams;
		expect(q.get("estado")).toBe("aguardando_aprovacao");
		expect(q.get("nivel")).toBe("alto");
		expect(q.get("inicio")).toBe("1970-01-01");
		expect(q.get("installation_id")).toBe("42");
		expect(q.get("limite")).toBe("25");
		if (q.get("pagina") === "2") {
			expect(q.get("snapshot")).toBe(snapshot);
			return r.fulfill({ json: { ...dados, pagina: 2, tem_mais: false, itens: [itemDaSegundaPagina] } });
		}
		return r.fulfill({ json: dados });
	});
});
for (const largura of [1440, 390, 360])
	for (const tema of ["escuro", "claro"]) {
		test(`visão geral em ${largura}px e tema ${tema}`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
			await page.addInitScript((t) => {
				localStorage.setItem("aprova-tema", t);
			}, tema);
			const escritas: string[] = [];
			page.on("request", (r) => {
				if (r.method() !== "GET") escritas.push(r.url());
			});
			await page.goto("/?repositorio=equipe%2Fprojeto");
			await expect(page.getByText("47%", { exact: true })).toBeVisible();
			await expect(page.getByText(/Métricas simuladas/)).toBeVisible();
			const fila = page.getByRole("region", { name: "Fila de pendências" });
			await expect(fila.getByRole("link", { name: "equipe/projeto #18" })).toBeVisible();
			await expect(fila.getByRole("status")).toHaveText("25 nesta página · 26 no total · página 1");
			await expect(fila.getByRole("link", { name: "equipe/projeto #42" })).toBeVisible();
			await page.evaluate(() => document.fonts.ready);
			await page.screenshot({ path: `../.capturas-mcp/dashboard/local-overview-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			if (largura === 1440) {
				await page.getByRole("searchbox").fill("inexistente");
				await expect(fila.getByText(/Nenhuma análise/)).toBeVisible();
				await page.getByRole("searchbox").fill("abc");
			}
			await fila.getByRole("button", { name: "Próxima" }).click();
			await expect(page).toHaveURL(/pagina=2/);
			await page.route("**/api/analises/**", (r) =>
				r.fulfill({
					json: {
						installation_id: 42,
						analise: itemDaSegundaPagina,
						branch: null,
						url_pr: "https://github.com/equipe/projeto/pull/43",
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
			await expect(fila.getByRole("status")).toHaveText("1 nesta página · 26 no total · página 2");
			await fila.getByRole("link", { name: "equipe/projeto #43" }).click();
			await expect(page.getByRole("heading", { name: "equipe/projeto / PR #43" })).toBeVisible();
			await page.getByRole("link", { name: "Voltar à visão geral" }).click();
			await page.reload();
			await expect(fila.getByRole("link", { name: "equipe/projeto #43" })).toBeVisible();
			if (largura === 1440) await expect(page.getByRole("searchbox")).toHaveValue("abc");
			expect(escritas).toEqual([]);
		});
	}
test("fila permite recuperar de erro HTTP e de resposta fora do escopo", async ({ page }) => {
	let resposta = "indisponivel";
	await page.route("**/api/analises?*", (r) => {
		if (resposta === "indisponivel") return r.fulfill({ status: 503, json: {} });
		return r.fulfill({ json: { ...dados, installation_id: resposta === "fora-do-escopo" ? 43 : 42 } });
	});
	await page.goto("/");
	const fila = page.getByRole("region", { name: "Fila de pendências" });
	await expect(fila.getByText(/temporariamente indisponível/)).toBeVisible();
	await expect(page.getByText("47%", { exact: true })).toBeVisible();
	resposta = "fora-do-escopo";
	await fila.getByRole("button", { name: "Tentar novamente" }).click();
	await expect(fila.getByText(/contrato esperado/)).toBeVisible();
	await expect(fila.getByRole("link")).toHaveCount(0);
	resposta = "autorizada";
	await fila.getByRole("button", { name: "Tentar novamente" }).click();
	await expect(fila.getByRole("link", { name: "equipe/projeto #18" })).toBeVisible();
});

test("trocar o repositório cancela a leitura anterior e reinicia a paginação", async ({ page }) => {
	let liberar!: () => void;
	const bloqueada = new Promise<void>((resolve) => {
		liberar = resolve;
	});
	const falhas: string[] = [];
	page.on("requestfailed", (r) => falhas.push(r.url()));
	await page.route("**/api/analises?*", async (r) => {
		const q = new URL(r.request().url()).searchParams;
		if (!q.has("repositorio")) {
			await bloqueada;
			return r.fulfill({ json: { ...dados, pagina: 2, itens: [{ ...item, repositorio: "outra/equipe" }] } });
		}
		expect(q.get("repositorio")).toBe(item.repositorio);
		expect(q.has("pagina")).toBe(false);
		expect(q.has("snapshot")).toBe(false);
		return r.fulfill({ json: dados });
	});
	const anterior = page.waitForRequest((r) => new URL(r.url()).pathname === "/api/analises");
	await page.goto(`/?pagina=2&snapshot=${snapshot}`);
	const pedido = await anterior;
	try {
		await page.getByRole("combobox", { name: "Repositório" }).click();
		await page.getByRole("option", { name: item.repositorio }).click();
		await expect(page.getByRole("link", { name: "equipe/projeto #18" })).toBeVisible();
	} finally {
		liberar();
	}
	await expect.poll(() => falhas).toContain(pedido.url());
	await expect(page.getByRole("region", { name: "Fila de pendências" })).not.toContainText("outra/equipe");
});

test("instalação fora da sessão impede todas as leituras da fila", async ({ page }) => {
	const consultas: string[] = [];
	page.on("request", (r) => {
		if (["/api/repositorios", "/api/analises"].includes(new URL(r.url()).pathname)) consultas.push(r.url());
	});
	await page.goto("/?installation_id=43");
	await expect(page.getByRole("heading", { name: "Instalação não autorizada" })).toBeVisible();
	expect(consultas).toEqual([]);
});
