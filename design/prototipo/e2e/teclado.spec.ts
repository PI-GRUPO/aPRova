import { expect, test } from "@playwright/test";
import { exigirFocoVisivel, exigirTitulo, tabularAte } from "./apoio";

test("navegação completa só com teclado, com foco sempre visível", async ({ page }) => {
	await page.goto("/");

	await page.keyboard.press("Tab");
	const pular = page.getByRole("link", { name: "Pular para o conteúdo" });
	await expect(pular).toBeFocused();
	await exigirFocoVisivel(pular);
	await page.keyboard.press("Enter");
	await expect(page.getByRole("main")).toBeFocused();

	await page.goto("/");
	const navegacao = page.getByRole("navigation", { name: "Principal" });
	await tabularAte(page, navegacao.getByRole("link", { name: "Análises", exact: true }));
	await page.keyboard.press("Enter");
	await exigirTitulo(page, "Análises");

	const linha = page.getByRole("row").filter({ hasText: "aPRova-TESTES" }).filter({ hasText: "teste-peso-zero" });
	await tabularAte(page, linha.getByRole("link", { name: /#1\b/ }), 200);
	await page.keyboard.press("Enter");
	await exigirTitulo(page, /aPRova-TESTES.*#1/);

	await tabularAte(page, page.getByRole("link", { name: /ver o recibo/i }));
	await page.keyboard.press("Enter");
	await exigirTitulo(page, /^Recibo /);

	await tabularAte(page, page.getByRole("button", { name: /Usar tema/ }), 200);
	await page.keyboard.press("Enter");
	await expect(page.locator("html")).toHaveAttribute("data-tema", "claro");
});

test("os filtros da tabela de análises funcionam pelo teclado", async ({ page }) => {
	await page.goto("/analises");
	const filtro = page.getByRole("combobox", { name: "Nível" });
	await tabularAte(page, filtro);
	await page.keyboard.press("Enter");
	await page.getByRole("option", { name: "alto" }).focus();
	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(/nivel=alto/);
	const linhas = page.getByRole("row").filter({ has: page.getByRole("cell") });
	await expect(linhas.first()).toBeVisible();
	for (const linhaDaTabela of await linhas.all()) {
		await expect(linhaDaTabela.getByTestId(/^selo-risco-/)).toHaveAttribute("data-testid", "selo-risco-alto");
	}
});

test("nenhum elemento focável fica sem contorno em cada tela", async ({ page }) => {
	for (const caminho of ["/", "/analises", "/recibos", "/regras"]) {
		await page.goto(caminho);
		const focaveis = await page.locator("a[href], button, [tabindex='0'], input, select, [role='combobox']").count();
		expect(focaveis).toBeGreaterThan(4);
		for (let passo = 0; passo < Math.min(focaveis, 60); passo++) {
			await page.keyboard.press("Tab");
			const focado = page.locator(":focus");
			if ((await focado.count()) === 0) break;
			await exigirFocoVisivel(focado);
		}
	}
});
