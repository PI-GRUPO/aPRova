import { expect, type Page, test } from "@playwright/test";

const fundoEscuro = "rgb(8, 9, 10)";
const fundoClaro = "rgb(233, 234, 236)";

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
	expect(noEscuro).toBe("rgb(235, 87, 87)");
	expect(noClaro).toBe("rgb(180, 47, 40)");
});

test("o texto usa Inter e os identificadores continuam em Plex Mono", async ({ page }) => {
	await page.goto("/");
	expect(await page.evaluate(() => getComputedStyle(document.body).fontFamily)).toContain("Inter Variable");
	const identificador = page.getByRole("main").getByText("teste-peso-zero");
	expect(await identificador.evaluate((elemento) => getComputedStyle(elemento).fontFamily)).toContain("IBM Plex Mono");
	expect(await page.evaluate(() => document.fonts.check('14px "Inter Variable"'))).toBe(true);
});

test("o acento limão aparece só no foco e no item ativo, e os links não têm cor própria", async ({ page }) => {
	await page.goto("/");
	const corDoTexto = await page.evaluate(() => getComputedStyle(document.body).color);
	const link = page.getByRole("main").getByRole("link", { name: /aPRova-TESTES/ });
	expect(await link.evaluate((elemento) => getComputedStyle(elemento).color)).toBe(corDoTexto);
	expect(await link.evaluate((elemento) => getComputedStyle(elemento).textDecorationLine)).toBe("underline");

	await page.keyboard.press("Tab");
	const focado = page.locator(":focus");
	expect(await focado.evaluate((elemento) => getComputedStyle(elemento).outlineColor)).toBe("rgb(228, 242, 34)");

	const ativo = page.getByRole("navigation", { name: "Principal" }).getByRole("link", { name: "Visão geral", exact: true });
	expect(await ativo.locator("svg").evaluate((elemento) => getComputedStyle(elemento).color)).toBe("rgb(228, 242, 34)");

	await page.getByRole("button", { name: "Usar tema claro" }).click();
	expect(await ativo.locator("svg").evaluate((elemento) => getComputedStyle(elemento).color)).toBe("rgb(95, 102, 0)");
});
