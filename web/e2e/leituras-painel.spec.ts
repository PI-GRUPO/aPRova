import { readFile } from "node:fs/promises";
import { expect, type Route, test } from "@playwright/test";

const origem = "http://127.0.0.1:18080";
for (const largura of [1440, 390]) {
	test(`visão geral mantém a instalação ao navegar e permite trocá-la em ${largura}px`, async ({ page }) => {
		await page.setViewportSize({ width: largura, height: 900 });
		await page.goto(`${origem}/entrar`);
		await page.getByRole("link", { name: "Continuar com GitHub" }).click();
		await page.goto(`${origem}/repositorios?installation_id=43&pagina=2`);
		const menu = page.getByRole("navigation", { name: largura === 1440 ? "Principal" : "Principal no celular", exact: true });
		await menu.getByRole("link", { name: "Visão geral", exact: true }).click();
		await expect(page).toHaveURL(`${origem}/?installation_id=43`);
		const metricas = page.locator(".metrica-overview").filter({ has: page.getByRole("heading", { name: "Análises recebidas" }) });
		await expect(metricas.locator(".valor-overview")).toHaveText("0");
		await expect(page.getByRole("combobox", { name: "Instalação", exact: true })).toHaveText("segunda-equipe");
		await page.getByRole("combobox", { name: "Instalação", exact: true }).click();
		await page.getByRole("option", { name: "equipe-e2e", exact: true }).click();
		await expect(page).toHaveURL(`${origem}/?installation_id=42`);
		await expect(metricas.locator(".valor-overview")).toHaveText("1");
		await page.reload();
		await expect(metricas.locator(".valor-overview")).toHaveText("1");
		for (const tema of ["escuro", "claro"]) {
			await page.evaluate((valor) => localStorage.setItem("aprova-tema", valor), tema);
			await page.reload();
			await expect(metricas.locator(".valor-overview")).toHaveText("1");
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			await page.screenshot({ path: `../.capturas-mcp/dashboard/instalacao-overview-${largura}-${tema}.png`, fullPage: true });
		}
		await menu.getByRole("link", { name: "Regras", exact: largura !== 1440 }).click();
		await expect(page).toHaveURL(`${origem}/regras?installation_id=42`);
		const voltar = page.getByRole("link", { name: "Voltar à Visão geral", includeHidden: true });
		await expect(voltar).toHaveAttribute("href", "/?installation_id=42");
		if (largura === 1440) await menu.getByRole("link", { name: "Visão geral", exact: true }).click();
		else await voltar.click();
		await expect(page).toHaveURL(`${origem}/?installation_id=42`);
		await expect(metricas.locator(".valor-overview")).toHaveText("1");
	});
}
for (const largura of [1440, 390]) {
	test(`leituras do build de produção, escopo e exportação em ${largura}px`, async ({ page }) => {
		await page.setViewportSize({ width: largura, height: 900 });
		await page.goto(`${origem}/entrar`);
		await page.getByRole("link", { name: "Continuar com GitHub" }).click();
		for (const tema of ["escuro", "claro"]) {
			await page.evaluate((t) => localStorage.setItem("aprova-tema", t), tema);
			await page.goto(`${origem}/?installation_id=42`);
			await expect(page.getByText(/Dados da API · 30 dias/)).toBeVisible();
			await expect(page.getByRole("heading", { name: "Falhas por tipo" })).toBeVisible();
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			await page.screenshot({ path: `../.capturas-mcp/dashboard/leituras-overview-${largura}-${tema}.png`, fullPage: true });
			await page.locator("button.celula-mapa").first().click();
			await expect(page.getByRole("dialog")).toContainText("Nenhuma análise neste recorte.");
			await page.keyboard.press("Escape");
			await page.goto(`${origem}/repositorios?installation_id=42`);
			await expect(page.getByRole("link", { name: "equipe-e2e/projeto", exact: true })).toBeVisible();
			await page.goto(`${origem}/conta`);
			await expect(page.getByText("Vínculo configurado para seu login GitHub.")).toBeVisible();
			await expect(page.getByRole("button", { name: "Desvincular conta" })).toBeDisabled();
		}
		await page.goto(`${origem}/recibos?installation_id=42&repositorio=equipe-e2e%2Fprojeto`);
		if (largura < 500) await page.getByText("Período, cadeia e exportação", { exact: true }).click();
		await expect(page.getByRole("button", { name: "Verificar cadeia" })).toBeDisabled();
		for (const formato of ["json", "csv"]) {
			const download = page.waitForEvent("download");
			await page.getByRole("button", { name: `Exportar ${formato.toUpperCase()}` }).click();
			const arq = await download;
			expect(arq.suggestedFilename()).toBe(`recibos.${formato}`);
			const caminho = await arq.path();
			expect(caminho).toBeTruthy();
			expect(await readFile(caminho as string, "utf8")).toContain("equipe-e2e/projeto");
		}
		for (const [rota, titulo] of [
			["regras", "Regras do gate"],
			["auditoria", "Auditoria de ações"],
		]) {
			await page.goto(`${origem}/${rota}?installation_id=42`);
			await expect(page.getByRole("heading", { level: 1, name: titulo })).toBeVisible();
			if (rota === "regras") {
				if (largura < 500) await page.getByRole("button", { name: "Estatísticas e configuração" }).click();
				await expect(page.getByRole("cell", { name: "desligada", exact: true })).toBeVisible();
			} else {
				await expect(page.getByRole("cell", { name: "permissao_insuficiente" })).toBeVisible();
				await page.goto(`${origem}/auditoria?installation_id=42&origem=Painel`);
				await expect(page.getByText("Nenhum registro encontrado para os filtros selecionados.")).toBeVisible();
			}
		}
	});
}
test("auditoria aceita o mesmo corte em outro fuso e recusa diferença de um nanosegundo", async ({ page }) => {
	await page.goto(`${origem}/entrar`);
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await expect(page).toHaveURL(`${origem}/`);
	const instante = Date.now() - 60_000;
	const utc = new Date(instante).toISOString().slice(0, 19);
	const local = new Date(instante - 3 * 3_600_000).toISOString().slice(0, 19);
	const filtros = new URLSearchParams({ installation_id: "42", snapshot: `${local}.1234567809-03:00` });
	await page.goto(`${origem}/auditoria?${filtros}`);
	await expect(page.getByRole("cell", { name: "permissao_insuficiente" })).toBeVisible();

	await page.route("**/api/web/auditoria?*", async (rota) => {
		const resposta = await rota.fetch();
		const corpo = await resposta.json();
		await rota.fulfill({ response: resposta, json: { ...corpo, snapshot: `${utc}.123456781Z` } });
	});
	await page.reload();
	await expect(page.getByText("Resposta do painel fora do contrato esperado.")).toBeVisible();
	await expect(page.getByRole("cell", { name: "permissao_insuficiente" })).toHaveCount(0);
});

test("falha real das métricas preserva a fila e permite nova consulta", async ({ page }) => {
	await page.goto(`${origem}/entrar`);
	await page.route("**/api/metricas?*", (r) => r.fulfill({ status: 503, json: {} }));
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await expect(page.getByRole("heading", { name: "Métricas indisponíveis" })).toBeVisible();
	await expect(page.getByText("0 nesta página · 0 no total · página 1")).toBeVisible();
	await page.unroute("**/api/metricas?*");
	await page.getByRole("button", { name: "Tentar novamente" }).click();
	await expect(page.getByText(/Dados da API · 30 dias/)).toBeVisible();
});

test("visão geral reaparece enquanto as métricas do mesmo escopo são revalidadas", async ({ page }) => {
	await page.goto(`${origem}/entrar`);
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await expect(page.getByText(/Dados da API · 30 dias/)).toBeVisible();

	const menu = page.getByRole("navigation", { name: "Principal", exact: true });
	await menu.getByRole("link", { name: "Análises", exact: true }).click();
	await expect(page.getByRole("heading", { name: "Análises", exact: true })).toBeVisible();
	let capturar!: (rota: Route) => void;
	const pendente = new Promise<Route>((resolve) => {
		capturar = resolve;
	});
	await page.route("**/api/metricas?*", capturar, { times: 1 });
	await menu.getByRole("link", { name: "Visão geral", exact: true }).click();
	const consulta = await pendente;
	await expect(page.getByText(/Dados da API · 30 dias/)).toBeVisible();
	await expect(page.getByText("Carregando métricas…")).toHaveCount(0);
	const atualizada = page.waitForResponse((r) => r.url() === consulta.request().url());
	await consulta.continue();
	expect((await atualizada).ok()).toBeTruthy();
});

test("a página do recorte do mapa pode ser compartilhada e recarregada", async ({ page }) => {
	await page.goto(`${origem}/entrar`);
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	const escopo = { installation_id: "42", repositorio: "equipe-e2e/projeto" };
	const corte = new Date(Date.now() - 60000).toISOString().replace(/\.\d{3}Z$/, ".123456780Z");
	const data = corte.slice(0, 10);
	const url = new URLSearchParams({ ...escopo, mapa_pagina: "2", mapa_snapshot: corte, mapa_dia: data, mapa_nivel: "alto" });
	const resposta = page.waitForResponse(
		(r) => new URL(r.url()).pathname === "/api/analises" && new URL(r.url()).searchParams.get("pagina") === "2",
	);
	await page.goto(`${origem}/?${url}`);
	const q = new URL((await resposta).url()).searchParams;
	expect(q.get("repositorio")).toBe(escopo.repositorio);
	expect(q.get("snapshot")).toBe(corte);
	await page.getByRole("dialog").getByRole("button", { name: "Anterior" }).click();
	await expect(page).toHaveURL(/mapa_pagina=1/);
	await page.reload();
	await expect(page.getByRole("dialog")).toContainText("Nenhuma análise neste recorte.");
});
