import { expect, test } from "@playwright/test";

const telas = [
	["regras", "Regras do gate"],
	["auditoria", "Auditoria de ações"],
	["repositorios", "Repositórios e instalação"],
	["configuracao", "Configuração do gate"],
	["sugestoes", "Sugestões de regras pela IA"],
	["sla", "Limites de SLA"],
];
test.beforeEach(async ({ page }) => {
	await page.route("**/api/sessao", (r) => r.fulfill({ json: { autenticado: false, login_disponivel: false } }));
});
for (const largura of [1440, 390, 360]) {
	for (const tema of ["escuro", "claro"]) {
		test(`gestão em ${largura}px e tema ${tema}`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
			await page.addInitScript((t) => localStorage.setItem("aprova-tema", t), tema);
			for (const [caminho, titulo] of telas) {
				await page.goto(`/${caminho}`);
				await expect(page.getByRole("heading", { name: titulo, exact: true })).toBeVisible();
				await expect(page.getByText("Demonstração · dados e alterações simulados nesta sessão da página.")).toBeVisible();
				await page.evaluate(() => document.fonts.ready);
				await page.screenshot({ path: `../.capturas-mcp/dashboard/local-${caminho}-${largura}-${tema}.png`, fullPage: true });
				expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), caminho).toBeTruthy();
			}
		});
	}
}
test("configuração revisa mudanças e mantém versões apenas na simulação", async ({ page }) => {
	const escritas: string[] = [];
	page.on("request", (r) => {
		if (r.method() !== "GET") escritas.push(r.url());
	});
	await page.goto("/configuracao");
	await expect(page.getByRole("button", { name: "Revisar mudanças" })).toBeDisabled();
	await page.getByRole("button", { name: "Adicionar caminho" }).click();
	await page.getByRole("textbox", { name: "Caminho da pasta ou arquivo" }).fill("/absoluto/");
	await page.getByRole("button", { name: "Adicionar", exact: true }).click();
	await expect(page.getByText("Informe um caminho relativo, único e com até 200 caracteres.")).toBeVisible();
	await page.getByRole("textbox", { name: "Caminho da pasta ou arquivo" }).fill("billing/");
	await page.keyboard.press("Enter");
	await expect(page.getByRole("dialog")).toHaveCount(0);
	await page.getByRole("spinbutton", { name: "Limite de arquivos" }).fill("30");
	await page.getByRole("button", { name: "Revisar mudanças" }).click();
	await expect(page.getByRole("dialog")).toContainText("billing/");
	const confirmar = page.getByRole("button", { name: "Confirmar e criar versão" });
	await expect(confirmar).toBeDisabled();
	await page.getByRole("checkbox").check();
	await confirmar.click();
	await expect(page.getByRole("status")).toContainText("Versão v4 registrada na simulação");
	await expect(page.getByRole("textbox", { name: "Versão em vigor" })).toHaveValue(/v4/);
	await expect(page.getByRole("button", { name: "Revisar mudanças" })).toBeDisabled();
	await page.getByRole("spinbutton", { name: "Limite de arquivos" }).fill("40");
	await page.getByRole("button", { name: "Descartar alterações" }).click();
	await expect(page.getByRole("spinbutton", { name: "Limite de arquivos" })).toHaveValue("30");
	await page.reload();
	await expect(page.getByRole("spinbutton", { name: "Limite de arquivos" })).toHaveValue("20");
	expect(escritas).toEqual([]);
});
test("SLA valida ordem dos prazos e isola repositórios", async ({ page }) => {
	await page.goto("/sla");
	await page.getByRole("spinbutton", { name: "Primeiro aviso (horas)" }).fill("9");
	await page.getByRole("button", { name: "Salvar limites" }).click();
	await expect(page.getByRole("alert")).toContainText("escalonamento deve ocorrer depois");
	await page.getByRole("spinbutton", { name: "Primeiro aviso (horas)" }).fill("3");
	await page.getByRole("button", { name: "Salvar limites" }).click();
	await expect(page.getByRole("status")).toContainText("Limites salvos na simulação");
	await expect(page.getByRole("cell", { name: "A partir de 3 horas", exact: true })).toBeVisible();
	await page.getByRole("combobox", { name: "Repositório" }).click();
	await page.getByRole("option", { name: "cobranca-api", exact: true }).click();
	await expect(page.getByRole("spinbutton", { name: "Primeiro aviso (horas)" })).toHaveValue("2");
	await page.getByRole("combobox", { name: "Repositório" }).click();
	await page.getByRole("option", { name: "aPRova-TESTES", exact: true }).click();
	await expect(page.getByRole("spinbutton", { name: "Primeiro aviso (horas)" })).toHaveValue("3");
});
test("auditoria filtra resultados e preserva filtro ao recarregar", async ({ page }) => {
	await page.goto("/auditoria");
	await expect(page.locator("tbody tr")).toHaveCount(4);
	await page.getByRole("checkbox", { name: "Somente recusadas" }).click();
	await expect(page.locator("tbody tr")).toHaveCount(2);
	await page.reload();
	await expect(page.getByRole("checkbox", { name: "Somente recusadas" })).toBeChecked();
	await expect(page.locator("tbody tr")).toHaveCount(2);
	await page.getByRole("combobox", { name: "Origem" }).click();
	await page.getByRole("option", { name: "Slack", exact: true }).click();
	await expect(page.locator("tbody tr")).toHaveCount(1);
});
test("sugestão só altera a configuração simulada após confirmação explícita", async ({ page }) => {
	await page.goto("/sugestoes");
	await page.locator("tbody button").first().click();
	await page.getByRole("button", { name: "Revisar configuração", exact: true }).click();
	await expect(page).toHaveURL(/configuracao.*proposta=/);
	await expect(page.getByRole("textbox", { name: "Versão em vigor" })).toHaveValue(/v3/);
	await expect(page.getByRole("spinbutton", { name: "Limite de arquivos" })).toHaveValue("30");
	await page.getByRole("button", { name: "Revisar mudanças" }).click();
	await page.getByRole("checkbox").check();
	await page.getByRole("button", { name: "Confirmar e criar versão" }).click();
	await expect(page.getByRole("status")).toContainText("Versão v4");
	await page.getByRole("link", { name: "Sugestões da IA" }).click();
	await expect(page.locator("tbody tr")).toHaveCount(1);
	await page.locator("tbody button").first().click();
	await page.getByRole("button", { name: "Recusar", exact: true }).click();
	await expect(page.locator("tbody tr")).toHaveCount(0);
	await expect(page.getByText("Nenhum registro encontrado para os filtros selecionados.")).toBeVisible();
});

test("catálogo autenticado consulta a API e não substitui falhas por mocks", async ({ page }) => {
	await page.route("**/api/sessao", (r) =>
		r.fulfill({
			json: {
				autenticado: true,
				login_disponivel: true,
				usuario: { id: 123, login: "demo" },
				instalacoes: [{ id: 42, conta: "equipe" }],
				csrf_token: "x".repeat(32),
				expira_em: new Date(Date.now() + 3600000).toISOString(),
			},
		}),
	);
	let falhar = true;
	const instalacoes: string[] = [];
	await page.route("**/api/repositorios?*", (r) => {
		instalacoes.push(new URL(r.request().url()).searchParams.get("installation_id") ?? "");
		return falhar ? r.fulfill({ status: 503, json: {} }) : r.fulfill({ json: { installation_id: 42, repositorios: ["equipe/real"] } });
	});
	await page.goto("/repositorios");
	await expect(page.getByRole("heading", { name: "Consulta de repositórios indisponível" })).toBeVisible();
	await expect(page.getByRole("cell", { name: "cobranca-api", exact: true })).toHaveCount(0);
	falhar = false;
	await page.getByRole("button", { name: "Tentar novamente" }).click();
	await expect(page.getByRole("cell", { name: "real", exact: true })).toBeVisible();
	await expect(page.getByRole("cell", { name: "Não confirmado", exact: true })).toBeVisible();
	await expect(page.getByRole("button", { name: "Configurar cobranca-api no GitHub" })).toHaveCount(0);
	await page.getByRole("link", { name: "real", exact: true }).click();
	await expect(page).toHaveURL(/analises\?installation_id=42&repositorio=equipe%2Freal/);
	expect(instalacoes.length).toBeGreaterThanOrEqual(2);
	expect(instalacoes.every((i) => i === "42")).toBeTruthy();
});
