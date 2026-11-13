import { expect, test } from "@playwright/test";
import { abrirCasoReal, abrirReciboDoCasoReal, exigirTitulo, rotasFixas } from "./apoio";

test.describe("rotas", () => {
	for (const rota of rotasFixas) {
		test(`${rota.caminho} abre com o título ${rota.titulo}`, async ({ page }) => {
			await page.goto(rota.caminho);
			await exigirTitulo(page, rota.titulo);
			await expect(page).toHaveTitle(`${rota.titulo} · aPRova`);
		});
	}

	test("a barra lateral leva às quatro seções", async ({ page }) => {
		await page.goto("/");
		const navegacao = page.getByRole("navigation", { name: "Principal" });
		for (const rota of [...rotasFixas].reverse()) {
			await navegacao.getByRole("link", { name: rota.titulo, exact: true }).click();
			await expect(page).toHaveURL(new RegExp(`${rota.caminho === "/" ? "/" : rota.caminho}$`));
			await exigirTitulo(page, rota.titulo);
			await expect(navegacao.getByRole("link", { name: rota.titulo, exact: true })).toHaveAttribute("aria-current", "page");
		}
	});

	test("rota desconhecida explica e devolve à Visão geral", async ({ page }) => {
		await page.goto("/nao-existe");
		await exigirTitulo(page, "Página não encontrada");
		await page.getByRole("main").getByRole("link", { name: "Visão geral" }).click();
		await exigirTitulo(page, "Visão geral");
	});
});

test.describe("página não encontrada", () => {
	test("a composição tipográfica é decorativa e não estoura a largura", async ({ page }) => {
		for (const largura of [390, 1440]) {
			await page.setViewportSize({ width: largura, height: 844 });
			await page.goto("/nao-existe");
			const fragmentos = page.getByTestId("fragmentos-404");
			await expect(fragmentos).toBeVisible();
			await expect(fragmentos).toHaveAttribute("aria-hidden", "true");
			expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(largura);
			await expect(page.getByRole("main").getByRole("link", { name: "Visão geral" })).toBeVisible();
		}
	});
});

test.describe("da lista ao recibo", () => {
	test("clicar numa análise leva ao detalhe certo", async ({ page }) => {
		await abrirCasoReal(page);
		await exigirTitulo(page, /aPRova-TESTES.*#1/);
		const principal = page.getByRole("main");
		await expect(principal.getByText("teste-peso-zero")).toBeVisible();
		await expect(principal.getByText("falha_sandbox").first()).toBeVisible();
		await expect(principal.getByText("1 de 11 testes falharam").first()).toBeVisible();
		await expect(page).toHaveTitle(/#1 · Análises · aPRova$/);
	});

	test("o detalhe mostra Aprovar e Reexecutar desabilitados, com a explicação", async ({ page }) => {
		await abrirCasoReal(page);
		for (const acao of ["Aprovar", "Reexecutar"]) {
			const botao = page.getByRole("button", { name: acao });
			await expect(botao).toBeVisible();
			await expect(botao).toHaveAttribute("aria-disabled", "true");
		}
		await expect(page.getByText(/só funcionam no produto real/i)).toBeVisible();
	});

	test("do detalhe, o link do recibo leva ao recibo certo", async ({ page }) => {
		await abrirCasoReal(page);
		const idDaAnalise = page.url().split("/").pop() ?? "";
		const hash = (await page.getByTestId("recibo-hash").textContent()) ?? "";
		expect(hash).toMatch(/^[0-9a-f]{64}$/);

		await page.getByRole("link", { name: /ver o recibo/i }).click();

		await expect(page).toHaveURL(/\/recibos\/[0-9a-f-]{36}$/);
		await exigirTitulo(page, /^Recibo /);
		await expect(page.getByTestId("recibo-hash")).toHaveText(hash);
		await expect(page.getByRole("main").getByRole("link", { name: /aPRova-TESTES.*#1/ })).toHaveAttribute(
			"href",
			`/analises/${idDaAnalise}`,
		);
	});

	test("o recibo aponta para o anterior da cadeia", async ({ page }) => {
		await abrirReciboDoCasoReal(page);
		const anterior = page.getByRole("link", { name: /recibo anterior/i });
		const destino = await anterior.getAttribute("href");
		await anterior.click();
		await expect(page).toHaveURL(new RegExp(`${destino}$`));
		await exigirTitulo(page, /^Recibo /);
	});
});
