import { expect, type Locator, type Page } from "@playwright/test";

export const rotasFixas = [
	{ caminho: "/", titulo: "Visão geral" },
	{ caminho: "/analises", titulo: "Análises" },
	{ caminho: "/recibos", titulo: "Recibos" },
	{ caminho: "/regras", titulo: "Regras do gate" },
] as const;

export const estados = [
	{ parametro: "vazio", marcador: "vazio" },
	{ parametro: "carregando", marcador: "carregando" },
	{ parametro: "erro", marcador: "erro" },
] as const;

export async function exigirTitulo(pagina: Page, titulo: string | RegExp) {
	await expect(pagina.getByRole("heading", { level: 1 })).toHaveText(titulo);
	await expect(pagina).toHaveTitle(/aPRova$/);
}

export async function abrirCasoReal(pagina: Page) {
	await pagina.goto("/analises");
	const linha = pagina.getByRole("row").filter({ hasText: "aPRova-TESTES" }).filter({ hasText: "teste-peso-zero" });
	await linha.getByRole("link", { name: /#1\b/ }).click();
	await expect(pagina).toHaveURL(/\/analises\/[0-9a-f-]{36}$/);
}

export async function abrirReciboDoCasoReal(pagina: Page) {
	await abrirCasoReal(pagina);
	await pagina.getByRole("link", { name: /ver o recibo/i }).click();
	await expect(pagina).toHaveURL(/\/recibos\/[0-9a-f-]{36}$/);
}

export async function exigirFocoVisivel(alvo: Locator) {
	const contorno = await alvo.evaluate((elemento) => {
		const estilo = getComputedStyle(elemento);
		return {
			estilo: estilo.outlineStyle,
			largura: Number.parseFloat(estilo.outlineWidth),
			sombra: estilo.boxShadow,
		};
	});
	const temContorno = contorno.estilo !== "none" && contorno.largura >= 2;
	expect(temContorno, `foco sem contorno visível: ${JSON.stringify(contorno)}`).toBe(true);
}

export async function tabularAte(pagina: Page, alvo: Locator, limite = 80) {
	for (let passo = 0; passo < limite; passo++) {
		await pagina.keyboard.press("Tab");
		const focado = pagina.locator(":focus");
		await expect(focado, "o foco não pode se perder no body").toHaveCount(1);
		await exigirFocoVisivel(focado);
		if (await alvo.evaluate((elemento) => elemento === document.activeElement)) return;
	}
	throw new Error("o alvo não foi alcançado só com Tab");
}
