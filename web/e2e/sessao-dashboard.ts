import type { Page } from "@playwright/test";
export async function autenticarFixture(page: Page) {
	await page.route("**/api/sessao", (r) =>
		r.fulfill({
			json: {
				autenticado: true,
				login_disponivel: true,
				usuario: { id: 7, login: "revisor" },
				instalacoes: [{ id: 42, conta: "equipe" }],
				csrf_token: "x".repeat(43),
				expira_em: new Date(Date.now() + 3600000).toISOString(),
			},
		}),
	);
}
