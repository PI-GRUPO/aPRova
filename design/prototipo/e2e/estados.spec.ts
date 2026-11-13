import { expect, type Page, test } from "@playwright/test";
import { abrirCasoReal, abrirReciboDoCasoReal, estados, rotasFixas } from "./apoio";

async function exigirEstado(pagina: Page, marcador: string) {
	const principal = pagina.getByRole("main");
	await expect(principal).toHaveAttribute("data-estado", marcador);
	await expect(pagina.getByRole("heading", { level: 1 })).toBeVisible();

	if (marcador === "carregando") {
		await expect(principal.getByRole("status")).toContainText("Carregando");
		await expect(principal).toHaveAttribute("aria-busy", "true");
	}
	if (marcador === "erro") {
		await expect(principal.getByRole("alert")).toContainText("Não foi possível carregar");
		await expect(principal.getByRole("link", { name: "Tentar de novo" })).toBeVisible();
	}
	if (marcador === "vazio") {
		await expect(principal.getByTestId("estado-vazio")).toBeVisible();
		await expect(principal.getByRole("alert")).toHaveCount(0);
	}
}

test.describe("os quatro estados de cada tela", () => {
	for (const rota of rotasFixas) {
		test(`${rota.caminho} com dados`, async ({ page }) => {
			await page.goto(rota.caminho);
			await expect(page.getByRole("main")).toHaveAttribute("data-estado", "dados");
		});

		for (const estado of estados) {
			test(`${rota.caminho}?estado=${estado.parametro}`, async ({ page }) => {
				await page.goto(`${rota.caminho}?estado=${estado.parametro}`);
				await exigirEstado(page, estado.marcador);
			});
		}
	}

	for (const estado of estados) {
		test(`detalhe da análise com ?estado=${estado.parametro}`, async ({ page }) => {
			await abrirCasoReal(page);
			await page.goto(`${page.url()}?estado=${estado.parametro}`);
			await exigirEstado(page, estado.marcador);
		});

		test(`detalhe do recibo com ?estado=${estado.parametro}`, async ({ page }) => {
			await abrirReciboDoCasoReal(page);
			await page.goto(`${page.url()}?estado=${estado.parametro}`);
			await exigirEstado(page, estado.marcador);
		});
	}

	test("Tentar de novo volta ao estado com dados", async ({ page }) => {
		await page.goto("/?estado=erro");
		await page.getByRole("link", { name: "Tentar de novo" }).click();
		await expect(page.getByRole("main")).toHaveAttribute("data-estado", "dados");
	});

	test("valor desconhecido em ?estado= mostra os dados", async ({ page }) => {
		await page.goto("/?estado=qualquer");
		await expect(page.getByRole("main")).toHaveAttribute("data-estado", "dados");
	});
});
