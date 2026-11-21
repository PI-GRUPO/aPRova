import { expect, test } from "@playwright/test";
import { autenticarFixture } from "./sessao-dashboard";

const id = "00000000-0000-4000-8000-000000000001";
const pedidos = ["aPRova-TESTES #1", "cobranca-api #214", "painel-entregas #97", "cobranca-api #217", "painel-entregas #99"];
for (const largura of [1440, 390, 360])
	for (const tema of ["escuro", "claro"]) {
		test(`fila até ações indisponíveis em ${largura}px e ${tema}`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
			await page.addInitScript((t) => {
				localStorage.setItem("aprova-tema", t);
			}, tema);
			await autenticarFixture(page);
			const proibidas: string[] = [];
			page.on("request", (r) => {
				const p = new URL(r.url()).pathname;
				if (r.method() !== "GET" || p.startsWith("/api/analises") || p.startsWith("/receipts/")) proibidas.push(r.url());
			});
			await page.goto("/?fila=peso&repositorio=aPRova-TESTES");
			await page.getByRole("link", { name: pedidos[0], exact: true }).filter({ visible: true }).click();
			await expect(page).toHaveURL(new RegExp(`/analises/demonstracao/${id}`));
			await expect(page.locator("main")).not.toContainText(/dados simulados/i);
			await expect(page.getByText("Verificação de assinatura indisponível.")).toBeVisible();
			await page.evaluate(() => document.fonts.ready);
			await page.screenshot({ path: `../.capturas-mcp/fila/local-detalhe-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			await expect(page.getByRole("link", { name: "Abrir PR", exact: true })).toHaveCount(0);
			await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
			await page.getByRole("link", { name: "Voltar à visão geral", exact: true }).click();
			await expect(page).toHaveURL(/fila=peso&repositorio=aPRova-TESTES/);
			if (largura === 1440) await expect(page.getByRole("searchbox", { name: "Buscar na fila" })).toHaveValue("peso");
			await page.getByRole("link", { name: pedidos[0], exact: true }).filter({ visible: true }).click();
			await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
			expect(proibidas).toEqual([]);
		});
	}
test("as cinco referências abrem com sessão e IDs desconhecidos não acionam a API", async ({ page }) => {
	await autenticarFixture(page);
	const consultas: string[] = [];
	page.on("request", (r) => {
		if (new URL(r.url()).pathname.startsWith("/api/analises")) consultas.push(r.url());
	});
	for (const nome of pedidos) {
		await page.goto("/");
		await page.getByRole("link", { name: nome, exact: true }).filter({ visible: true }).click();
		await expect(page.getByRole("heading", { level: 1 })).toContainText(nome.replace(" #", " / PR #"));
		await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
	}
	await page.goto("/analises/demonstracao/desconhecida?retorno=https://outro.exemplo");
	await expect(page.getByRole("heading", { name: "Análise não encontrada" })).toBeVisible();
	await expect(page.getByRole("link", { name: "Voltar à visão geral" })).toHaveAttribute("href", "/");
	expect(consultas).toEqual([]);
});
test("uma falha real de leitura continua sendo erro e não vira demonstração", async ({ page }) => {
	await page.route("**/api/sessao", (r) =>
		r.fulfill({
			json: {
				autenticado: true,
				login_disponivel: true,
				usuario: { id: 7, login: "revisor-demo" },
				instalacoes: [{ id: 42, conta: "equipe" }],
				csrf_token: "a".repeat(43),
				expira_em: new Date(Date.now() + 3600000).toISOString(),
			},
		}),
	);
	await page.route("**/api/analises/**", (r) => r.fulfill({ status: 503, json: {} }));
	await page.goto(`/analises/${id}?installation_id=42`);
	await expect(page.getByRole("heading", { name: "Não foi possível abrir a análise" })).toBeVisible();
	await expect(page.getByText("Demonstração · dados simulados, sem operações reais.")).toHaveCount(0);
	await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toHaveCount(0);
});

test("erro ao carregar a fixture pode ser repetido sem fallback à API", async ({ page }) => {
	await autenticarFixture(page);
	let falhar = true;
	await page.route("**/src/mocks/analises-demonstracao.ts*", (r) => {
		if (falhar) {
			falhar = false;
			return r.abort();
		}
		return r.continue();
	});
	await page.goto(`/analises/demonstracao/${id}`);
	await expect(page.getByRole("heading", { name: "Não foi possível carregar a análise" })).toBeVisible();
	await page.getByRole("button", { name: "Tentar novamente", exact: true }).click();
	await expect(page.getByRole("heading", { name: "aPRova-TESTES / PR #1", exact: true })).toBeVisible();
});
