import { expect, test } from "@playwright/test";

const origem = "http://127.0.0.1:18080";
test("handlers reais de saúde e prontidão, APIs ausentes e navegação pelo Go", async ({ request, page }) => {
	const saude = await request.get(`${origem}/health`);
	expect(saude.status()).toBe(200);
	expect(await saude.json()).toEqual({ status: "ok" });
	const pronto = await request.get(`${origem}/ready`);
	expect(pronto.status()).toBe(503);
	expect(await pronto.json()).toEqual({ status: "indisponivel", dependencias: { postgres: "indisponivel" } });
	for (const caminho of ["/api/ausente", "/receipts/invalido/ausente", "/assets/inexistente.js"]) {
		const resposta = await request.get(`${origem}${caminho}`, { headers: { Accept: "text/html" } });
		expect(resposta.status()).toBe(404);
		expect(resposta.headers()["content-type"]).not.toContain("text/html");
	}
	const privada = await request.get(`${origem}/api/analises`, { headers: { Accept: "text/html" } });
	expect(privada.status()).toBe(401);
	await page.goto(`${origem}/analises`);
	await expect(page.getByRole("heading", { name: "Análises", exact: true })).toBeVisible();
	await page.reload();
	await expect(page.getByRole("heading", { name: "Análises", exact: true })).toBeVisible();
	await page.goto(`${origem}/__teste`);
	await expect(page.getByRole("heading", { name: "Página não encontrada", exact: true })).toBeVisible();
	await expect(page.getByText("Ambiente isolado de testes")).toHaveCount(0);
});
