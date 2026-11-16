import { expect, test } from "@playwright/test";

for (const largura of [1440, 390, 360]) {
	for (const tema of ["escuro", "claro"]) {
		test(`visão geral em ${largura}px e tema ${tema}`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
			await page.addInitScript((t) => localStorage.setItem("aprova-tema", t), tema);
			await page.route("**/api/sessao", (r) => r.fulfill({ json: { autenticado: false, login_disponivel: false } }));
			const escritas: string[] = [];
			page.on("request", (r) => {
				if (r.method() !== "GET") escritas.push(r.url());
			});
			await page.goto("/");
			await expect(page.getByRole("heading", { name: "Visão geral", exact: true })).toBeVisible();
			await expect(page.getByText("47%", { exact: true })).toBeVisible();
			await expect(page.getByText("Dados simulados · métricas de 30 dias e fila de referência")).toBeVisible();
			await page.evaluate(() => document.fonts.ready);
			await page.screenshot({ path: `../.capturas-mcp/dashboard/local-overview-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			if (largura === 1440) {
				await page.getByRole("searchbox", { name: "Buscar na fila" }).fill("go.mod");
				await expect(page.getByText("1 de 5 análises")).toBeVisible();
				await page.getByRole("button", { name: "cobranca-api #214", exact: true }).filter({ visible: true }).click();
				await expect(page.getByRole("dialog")).toContainText("Nenhuma ação será enviada ao núcleo");
				await page.keyboard.press("Escape");
				await expect(page.getByRole("dialog")).toHaveCount(0);
				await page.reload();
				await expect(page.getByRole("searchbox", { name: "Buscar na fila" })).toHaveValue("go.mod");
			} else {
				await page.getByRole("button", { name: "aPRova-TESTES #1", exact: true }).filter({ visible: true }).click();
				await expect(page.getByRole("dialog")).toContainText("Prévia com dados simulados");
				await page.keyboard.press("Escape");
			}
			const celula = page.getByRole("button", { name: "8 de nov. · risco alto: 2 análise(s)", exact: true });
			// A formatação do Intl pode variar entre versões do Chromium.
			const alvo = (await celula.count()) ? celula : page.locator(".nivel-alto.intensidade-2").first();
			await alvo.click();
			await expect(page.getByRole("dialog")).toContainText("Contagem simulada");
			expect(escritas).toEqual([]);
		});
	}
}
