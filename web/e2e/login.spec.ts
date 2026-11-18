import { expect, test } from "@playwright/test";

for (const largura of [1440, 390, 360, 320])
	for (const tema of ["escuro", "claro"]) {
		test(`login conforme Figma em ${largura}px e ${tema}`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
			await page.addInitScript((t) => localStorage.setItem("aprova-tema", t), tema);
			await page.route("**/api/sessao", (r) => r.fulfill({ json: { autenticado: false, login_disponivel: true } }));
			await page.goto("/entrar");
			const login = page.getByRole("link", { name: "Continuar com GitHub", exact: true });
			await expect(login).toHaveAttribute("href", "/auth/github");
			await expect(page.getByText("Protótipo · dados simulados", { exact: true })).toBeVisible();
			await expect(page.getByRole("link", { name: "Consultar recibo público" })).toHaveCount(0);
			await expect(page.getByRole("button", { name: /Usar tema/ })).toHaveCount(0);
			await page.evaluate(() => document.fonts.ready);
			const cartao = await page.locator(".cartao-entrar").boundingBox();
			expect(cartao).not.toBeNull();
			if (!cartao) throw new Error("Cartão de login não encontrado");
			expect(Math.abs(cartao.x + cartao.width / 2 - largura / 2)).toBeLessThanOrEqual(1);
			expect(Math.abs(cartao.y + cartao.height / 2 - (largura === 1440 ? 900 : 844) / 2)).toBeLessThanOrEqual(1);
			if (largura === 1440) {
				expect(cartao.width).toBe(480);
				expect(cartao.height).toBe(369);
			}
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			expect(
				await page.locator("img").evaluateAll((imgs) =>
					imgs.every((el) => {
						const i = el as HTMLImageElement;
						return i.complete && i.naturalWidth > 0 && i.width === i.naturalWidth && i.height === i.naturalHeight;
					}),
				),
			).toBeTruthy();
			await page.screenshot({ path: `../.capturas-mcp/login/local-${largura}-${tema}.png`, fullPage: true });
			await page.keyboard.press("Tab");
			await expect(login).toBeFocused();
		});
	}
test("login mantém erro recuperável, indisponibilidade e sessão autenticada", async ({ page }) => {
	let falhar = true;
	await page.route("**/api/sessao", (r) =>
		falhar ? r.fulfill({ status: 503, json: {} }) : r.fulfill({ json: { autenticado: false, login_disponivel: true } }),
	);
	await page.goto("/entrar?erro=autenticacao");
	await expect(page.getByText("Não foi possível concluir a autenticação. Inicie uma nova tentativa.")).toBeVisible();
	await expect(page.getByRole("button", { name: "Tentar novamente" })).toBeVisible();
	await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toHaveCount(0);
	falhar = false;
	await page.getByRole("button", { name: "Tentar novamente" }).click();
	await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toBeVisible();
	await page.unroute("**/api/sessao");
	await page.route("**/api/sessao", (r) => r.fulfill({ json: { autenticado: false, login_disponivel: false } }));
	await page.reload();
	await expect(page.getByRole("heading", { name: "Login indisponível" })).toBeVisible();
	await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toHaveCount(0);
	await page.unroute("**/api/sessao");
	await page.route("**/api/sessao", (r) =>
		r.fulfill({
			json: {
				autenticado: true,
				login_disponivel: true,
				usuario: { id: 7, login: "revisor-demo" },
				instalacoes: [],
				csrf_token: "a".repeat(43),
				expira_em: new Date(Date.now() + 3600000).toISOString(),
			},
		}),
	);
	await page.reload();
	await expect(page.getByRole("link", { name: "Continuar para o dashboard" })).toHaveAttribute("href", "/");
});
