import { expect, type Page, test } from "@playwright/test";

const fundoEscuro = "rgb(17, 18, 20)";
const fundoClaro = "rgb(245, 245, 246)";

function fundoDaPagina(pagina: Page) {
	return pagina.evaluate(() => getComputedStyle(document.body).backgroundColor);
}

test("o tema escuro é o padrão", async ({ page }) => {
	await page.goto("/");
	await expect(page.locator("html")).toHaveAttribute("data-tema", "escuro");
	expect(await fundoDaPagina(page)).toBe(fundoEscuro);
});

test("alternar o tema troca as cores e persiste ao recarregar", async ({ page }) => {
	await page.goto("/");
	const textoAntes = await page.evaluate(() => getComputedStyle(document.body).color);

	await page.getByRole("button", { name: "Usar tema claro" }).click();

	await expect(page.locator("html")).toHaveAttribute("data-tema", "claro");
	expect(await fundoDaPagina(page)).toBe(fundoClaro);
	expect(await page.evaluate(() => getComputedStyle(document.body).color)).not.toBe(textoAntes);
	await expect(page.getByRole("img", { name: "aPRova" })).toHaveAttribute("src", /fundo-claro/);

	await page.reload();
	await expect(page.locator("html")).toHaveAttribute("data-tema", "claro");
	expect(await fundoDaPagina(page)).toBe(fundoClaro);

	await page.goto("/analises");
	expect(await fundoDaPagina(page)).toBe(fundoClaro);

	await page.getByRole("button", { name: "Usar tema escuro" }).click();
	expect(await fundoDaPagina(page)).toBe(fundoEscuro);
	await page.reload();
	await expect(page.locator("html")).toHaveAttribute("data-tema", "escuro");
});

test("as cores de risco mudam com o tema", async ({ page }) => {
	await page.goto("/");
	const selo = page.getByTestId("selo-risco-alto").first();
	const noEscuro = await selo.evaluate((elemento) => getComputedStyle(elemento).color);
	await page.getByRole("button", { name: "Usar tema claro" }).click();
	const noClaro = await selo.evaluate((elemento) => getComputedStyle(elemento).color);
	expect(noEscuro).toBe("rgb(231, 93, 85)");
	expect(noClaro).toBe("rgb(180, 47, 40)");
});
