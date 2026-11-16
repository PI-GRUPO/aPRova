import { expect, type Page, test } from "@playwright/test";

const sessao = {
	autenticado: true,
	login_disponivel: true,
	usuario: { id: 7, login: "pessoa-fixture" },
	instalacoes: [
		{ id: 42, conta: "equipe" },
		{ id: 43, conta: "outra-equipe" },
	],
	csrf_token: "a".repeat(43),
	expira_em: new Date(Date.now() + 3600000).toISOString(),
};
const relatorio = {
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
};
async function simular(page: Page, estado: unknown = sessao) {
	await page.route("**/api/sessao", (r) => r.fulfill({ json: estado }));
	await page.route("**/api/web/relatorio-semanal?*", (r) =>
		r.fulfill({ json: { ...relatorio, installation_id: Number(new URL(r.request().url()).searchParams.get("installation_id")) } }),
	);
}
for (const largura of [1440, 390, 360])
	test(`login, conta e relatório semanal nos dois temas em ${largura}px`, async ({ page }) => {
		await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
		await simular(page, { autenticado: false, login_disponivel: true });
		await page.goto("/entrar");
		await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toHaveAttribute("href", "/auth/github");
		for (const tema of ["escuro", "claro"]) {
			if (tema === "claro") await page.getByRole("button", { name: "Usar tema claro" }).click();
			await page.evaluate(() => document.fonts.ready);
			await page.screenshot({ path: `../.capturas-mcp/f3/local-login-${largura}-${tema}.png`, fullPage: true });
		}
		await page.unroute("**/api/sessao");
		await page.route("**/api/sessao", (r) => r.fulfill({ json: sessao }));
		await page.goto("/");
		await expect(page.getByText("75%", { exact: true })).toBeVisible();
		await expect(page.getByText("6 concluídas de 8 finalizadas", { exact: true })).toBeVisible();
		for (const tema of ["claro", "escuro"]) {
			if (tema === "escuro") await page.getByRole("button", { name: "Usar tema escuro" }).click();
			await page.screenshot({ path: `../.capturas-mcp/f3/local-semanal-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
		}
		await page.goto("/conta");
		await expect(page.locator(".dados-identidade").getByText("@pessoa-fixture", { exact: true })).toBeVisible();
		await expect(page.getByRole("button", { name: "Conectar minha conta Slack" })).toBeDisabled();
		await page.screenshot({ path: `../.capturas-mcp/f3/local-conta-${largura}.png`, fullPage: true });
	});
test("troca de escopo descarta dados e URL adulterada não dispara relatório", async ({ page }) => {
	await simular(page);
	await page.goto("/");
	await expect(page.getByText("75%", { exact: true })).toBeVisible();
	await page.getByRole("combobox", { name: "Instalação", exact: true }).click();
	await page.getByRole("option", { name: "outra-equipe · 43" }).click();
	await expect(page).toHaveURL(/installation_id=43/);
	await expect(page.getByText("Instalação 43 · 1 repositórios autorizados na consulta")).toBeVisible();
	let consultas = 0;
	await page.route("**/api/web/relatorio-semanal?*", (r) => {
		consultas++;
		return r.abort();
	});
	await page.goto("/?installation_id=999");
	await expect(page.getByRole("heading", { name: "Instalação não autorizada" })).toBeVisible();
	expect(consultas).toBe(0);
	await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
});
test("taxa nula preserva ausência de denominador e zero execuções", async ({ page }) => {
	await simular(page);
	await page.route("**/api/web/relatorio-semanal?*", (r) =>
		r.fulfill({ json: { ...relatorio, total_execucoes: 0, concluidas: 0, rejeitadas: 0, falhas: 0, em_andamento: 0, taxa_sucesso: null } }),
	);
	await page.goto("/");
	await expect(page.getByText("Sem dados", { exact: true })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Sem análises neste período" })).toBeVisible();
});
for (const status of [401, 403, 503])
	test(`relatório ${status} limpa acesso ou informa falha`, async ({ page }) => {
		await simular(page);
		await page.route("**/api/web/relatorio-semanal?*", (r) => r.fulfill({ status, json: { status: "falha" } }));
		await page.goto("/");
		if (status === 401) {
			await expect(page).toHaveURL(/\/entrar$/);
		} else {
			await expect(page.getByRole("heading", { name: "Relatório indisponível" })).toBeVisible();
		}
		await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
	});
test("sessão sem instalação não consulta relatório", async ({ page }) => {
	await simular(page, { ...sessao, instalacoes: [] });
	let consultas = 0;
	await page.route("**/api/web/relatorio-semanal?*", (r) => {
		consultas++;
		return r.abort();
	});
	await page.goto("/");
	await expect(page.getByRole("heading", { name: "Nenhuma instalação autorizada" })).toBeVisible();
	expect(consultas).toBe(0);
});
test("fluxo HTTP com handlers reais de login, CSRF, relatório e logout", async ({ page, request }) => {
	await page.goto("http://127.0.0.1:18080/entrar");
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await expect(page).toHaveURL("http://127.0.0.1:18080/");
	await expect(page.getByText("75%", { exact: true })).toBeVisible();
	const cookies = await page.context().cookies();
	const cookie = cookies.find((c) => c.name === "aprova_sessao");
	expect(cookie?.httpOnly).toBeTruthy();
	const cabecalho = cookies.map((c) => `${c.name}=${c.value}`).join("; ");
	const negado = await request.get("http://127.0.0.1:18080/api/web/relatorio-semanal?installation_id=999", {
		headers: { Cookie: cabecalho },
	});
	expect(negado.status()).toBe(403);
	const csrf = await request.post("http://127.0.0.1:18080/api/sessao/sair", {
		headers: { Cookie: cabecalho, Origin: "http://invasor.test" },
	});
	expect(csrf.status()).toBe(403);
	await page.getByRole("button", { name: "Sair", exact: true }).click();
	await expect(page.getByRole("heading", { name: "Entre para continuar" })).toBeVisible();
	await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
	const revogada = await request.get("http://127.0.0.1:18080/api/web/relatorio-semanal?installation_id=42", {
		headers: { Cookie: cabecalho },
	});
	expect(revogada.status()).toBe(401);
});
