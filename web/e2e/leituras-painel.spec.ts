import { readFile } from "node:fs/promises";
import { expect, type Route, test } from "@playwright/test";

const origem = "http://127.0.0.1:18080";
const recibo = "00000000-0000-4000-8000-000000000002";
test("auditoria identifica o recibo que aponta para um elo ausente", async ({ page }) => {
	await page.goto(`${origem}/entrar`);
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await page.goto(`${origem}/recibos?installation_id=43&periodo=7`);
	await page.getByRole("button", { name: "Verificar cadeia" }).click();
	await expect(page.getByText(/Primeiro recibo inválido: 00000000-0000-4000-8000-000000000003/)).toBeVisible();
	await expect(page.getByText(/Recibo anterior ausente: 00000000-0000-4000-8000-000000000005/)).toBeVisible();
});
for (const largura of [1440, 390]) {
	test(`relatório semanal mantém Visão geral ativa e permite voltar pela trilha em ${largura}px`, async ({ page }) => {
		await page.setViewportSize({ width: largura, height: 900 });
		await page.goto(`${origem}/entrar`);
		await page.getByRole("link", { name: "Continuar com GitHub" }).click();
		await expect(page).toHaveURL(`${origem}/`);
		const escopo = "installation_id=42&repositorio=equipe-e2e%2Fprojeto";
		const visaoGeral = `/?${escopo}`;
		await page.goto(`${origem}${visaoGeral}`);
		const consultar = page.getByRole("link", { name: "Consultar relatório semanal", exact: true });
		const instalacao = page.getByRole("combobox", { name: "Instalação", exact: true });
		await expect(consultar).toBeInViewport();
		await expect(consultar).toHaveCSS("background-color", "rgb(255, 255, 255)");
		await expect(consultar.locator("svg")).toBeVisible();
		await expect(instalacao).toBeVisible();
		await page.evaluate(() => document.fonts.ready);
		const [campo, botao] = await Promise.all([instalacao.boundingBox(), consultar.boundingBox()]);
		expect(botao?.y).toBe(campo?.y);
		expect(botao?.x ?? 0).toBeGreaterThan(campo?.x ?? 0);
		await page.getByRole("button", { name: "Usar tema claro" }).click();
		await expect(consultar).toHaveClass(/\bbotao-secundario\b/);
		await consultar.click();
		await expect(page).toHaveURL(`${origem}/relatorio-semanal?${escopo}`);
		await expect(page.getByText("75%", { exact: true })).toBeVisible();

		const menu = page.getByRole("navigation", { name: largura === 1440 ? "Principal" : "Principal no celular", exact: true });
		const itemVisaoGeral = menu.getByRole("link", { name: "Visão geral", exact: true });
		await expect(itemVisaoGeral).toHaveClass(/\bactive\b/);
		await expect(itemVisaoGeral).toHaveAttribute("aria-current", "location");
		if (largura < 768) {
			await page.getByRole("button", { name: "Menu completo" }).click();
			const completo = page.getByRole("navigation", { name: "Todas as telas", exact: true });
			await expect(completo.getByRole("link", { name: "Visão geral", exact: true })).toHaveClass(/\bactive\b/);
			await page.keyboard.press("Escape");
		}

		const trilha = page.getByRole("navigation", { name: "Trilha de navegação" });
		const inicioDaTrilha = trilha.getByRole("link", { name: "Visão geral", exact: true });
		await expect(trilha.locator('[aria-current="page"]')).toHaveText("Relatório semanal");
		await expect(trilha).toBeInViewport();
		await expect(inicioDaTrilha).toHaveAttribute("href", visaoGeral);
		await expect(page.getByRole("link", { name: "Voltar", exact: true })).toHaveCount(0);
		await inicioDaTrilha.click();
		await expect(page).toHaveURL(`${origem}${visaoGeral}`);
		await expect(itemVisaoGeral).toHaveAttribute("aria-current", "page");
	});
}

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
			await expect(page.getByRole("link", { name: "Ver recibo", exact: true })).toHaveAttribute(
				"href",
				`/recibos/${recibo}?installation_id=42`,
			);
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
		const cadeia = page.waitForResponse((r) => r.url() === `${origem}/api/web/recibos/cadeia?installation_id=42` && r.status() === 200);
		await page.getByRole("button", { name: "Verificar cadeia" }).click();
		await cadeia;
		await expect(page.getByText("Cadeia verificada. Recibos conferidos: 1.")).toBeVisible();
		for (const formato of ["json", "csv"]) {
			const download = page.waitForEvent("download");
			await page.getByRole("button", { name: `Exportar ${formato.toUpperCase()}` }).click();
			const arq = await download;
			expect(arq.suggestedFilename()).toBe(`recibos.${formato}`);
			const caminho = await arq.path();
			expect(caminho).toBeTruthy();
			expect(await readFile(caminho as string, "utf8")).toContain("equipe-e2e/projeto");
		}
		const previa = largura < 500 ? `Ver prévia do recibo ${recibo}` : "00000000…";
		await page.getByRole("button", { name: previa, exact: true }).click();
		await page.getByRole("link", { name: "Consultar recibo assinado" }).click();
		await expect(page).toHaveURL(`${origem}/recibos/${recibo}?installation_id=42`);
		await page.getByRole("link", { name: "Voltar para recibos" }).click();
		await expect(page).toHaveURL(`${origem}/recibos?installation_id=42`);
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
	await page.goto(`${origem}/auditoria?${filtros.toString()}`);
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

test("proporção de risco alto usa análises avaliadas como denominador", async ({ page }) => {
	await page.goto(`${origem}/entrar`);
	await page.route("**/api/metricas?*", async (rota) => {
		const resposta = await rota.fetch();
		const corpo = await resposta.json();
		const mapa = [{ data: corpo.snapshot.slice(0, 10), alto: 1, medio: 1, baixo: 2, nao_avaliado: 0 }];
		const dados = { ...corpo.dados, recebidas: 4, avaliadas: 4, alto: 1, mapa };
		await rota.fulfill({ response: resposta, json: { ...corpo, dados } });
	});
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	const indicador = page.getByRole("region", { name: "Métricas dos últimos 30 dias" });
	const metrica = indicador.locator(".metrica-overview").filter({ hasText: "Proporção de risco alto" });
	await expect(metrica.getByText("25%", { exact: true })).toBeVisible();
	await expect(metrica.locator(".contexto-metrica-desktop")).toHaveText("1 de 4 avaliadas");
});

test("esperas da visão geral e da fila passam de minutos para horas e dias", async ({ page }) => {
	const esperaMaisAntiga = (5 * 24 * 60 + 3 * 60 + 25) * 60;
	const casos = [
		{ segundos: esperaMaisAntiga, texto: "5d 3h 25m" },
		{ segundos: 86_400, texto: "1d 0h 0m" },
		{ segundos: 86_399, texto: "23h 59m" },
		{ segundos: 3_600, texto: "1h 0m" },
		{ segundos: 3_599, texto: "59m" },
		{ segundos: 59, texto: "0m" },
	];
	await page.goto(`${origem}/entrar`);
	await page.route("**/api/metricas?*", async (rota) => {
		const resposta = await rota.fetch();
		const corpo = await resposta.json();
		const desde = new Date(Date.parse(corpo.snapshot) - esperaMaisAntiga * 1000).toISOString();
		const dados = { ...corpo.dados, pendentes: casos.length, desde };
		await rota.fulfill({ response: resposta, json: { ...corpo, dados } });
	});
	await page.route(
		(url) => url.pathname === "/api/analises" && url.searchParams.get("estado") === "aguardando_aprovacao",
		async (rota) => {
			const resposta = await rota.fetch();
			const corpo = await resposta.json();
			const itens = casos.map(({ segundos }, indice) => ({
				id: `00000000-0000-4000-8000-${String(indice + 1).padStart(12, "0")}`,
				repositorio: "equipe-e2e/projeto",
				numero_pr: indice + 1,
				sha_head: "abc1234",
				autor: "revisora",
				estado: "aguardando_aprovacao",
				nivel: "alto",
				recebido_em: corpo.snapshot,
				criado_em: corpo.snapshot,
				atualizado_em: corpo.snapshot,
				espera_segundos: segundos,
			}));
			await rota.fulfill({ response: resposta, json: { ...corpo, total: itens.length, itens } });
		},
	);
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	const metrica = page.locator(".metrica-overview").filter({ has: page.getByRole("heading", { name: "Espera mais antiga" }) });
	const fila = page.getByRole("region", { name: "Fila de pendências" });
	await expect(metrica.locator(".valor-overview")).toHaveText("5d 3h 25m");
	for (const { texto } of casos) await expect(fila.getByRole("cell", { name: texto, exact: true })).toBeVisible();
	await page.setViewportSize({ width: 390, height: 900 });
	await expect(metrica.getByText("5d 3h 25m", { exact: true })).toBeVisible();
	for (const { texto } of casos) await expect(fila.getByText(`Espera ${texto}`, { exact: true })).toBeVisible();
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

for (const { rota, titulo, endpoint, tabela } of [
	{ rota: "repositorios", titulo: "Repositórios", endpoint: "/api/repositorios", tabela: "Saúde por repositório" },
	{ rota: "recibos", titulo: "Recibos", endpoint: "/api/web/recibos", tabela: "Recibos por repositório" },
	{ rota: "regras", titulo: "Regras do gate", endpoint: "/api/web/regras", tabela: "Regras e estatísticas por regra" },
	{ rota: "auditoria", titulo: "Auditoria de ações", endpoint: "/api/web/auditoria", tabela: "Histórico de ações" },
]) {
	test(`${titulo} reaparece enquanto a consulta é atualizada`, async ({ page }) => {
		await page.goto(`${origem}/entrar`);
		await page.getByRole("link", { name: "Continuar com GitHub" }).click();
		await expect(page).toHaveURL(`${origem}/`);
		await page.goto(`${origem}/${rota}?installation_id=42`);
		const conteudo = page.getByRole("table", { name: tabela, exact: true });
		await expect(conteudo).toBeVisible();

		const menu = page.getByRole("navigation", { name: "Principal", exact: true });
		await menu.getByRole("link", { name: "Visão geral", exact: true }).click();
		await expect(conteudo).toHaveCount(0);
		let capturar!: (rota: Route) => void;
		const pendente = new Promise<Route>((resolve) => {
			capturar = resolve;
		});
		await page.route(
			(url) => url.pathname === endpoint && (rota !== "repositorios" || url.searchParams.get("detalhes") === "1"),
			capturar,
			{
				times: 1,
			},
		);
		await menu.getByRole("link", { name: titulo, exact: true }).click();
		const consulta = await pendente;
		await expect(conteudo).toBeVisible();
		const atualizada = page.waitForResponse((r) => r.url() === consulta.request().url());
		await consulta.continue();
		expect((await atualizada).ok()).toBeTruthy();
	});
}

test("um filtro novo de auditoria aguarda seus próprios dados", async ({ page }) => {
	await page.goto(`${origem}/entrar`);
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await expect(page).toHaveURL(`${origem}/`);
	await page.goto(`${origem}/auditoria?installation_id=42`);
	const tentativaAnterior = page.getByRole("cell", { name: "permissao_insuficiente" });
	await expect(tentativaAnterior).toBeVisible();
	let capturar!: (rota: Route) => void;
	const pendente = new Promise<Route>((resolve) => {
		capturar = resolve;
	});
	await page.route((url) => url.pathname === "/api/web/auditoria" && url.searchParams.get("origem") === "Painel", capturar, { times: 1 });
	await page.getByRole("combobox", { name: "Origem", exact: true }).click();
	await page.getByRole("option", { name: "Painel", exact: true }).click();
	const consulta = await pendente;
	await expect(page.getByText("Carregando…", { exact: true })).toBeVisible();
	await expect(tentativaAnterior).toHaveCount(0);
	await consulta.continue();
	await expect(page.getByText("Nenhum registro encontrado para os filtros selecionados.")).toBeVisible();
});

test("a página do recorte do mapa pode ser compartilhada e recarregada", async ({ page }) => {
	await page.goto(`${origem}/entrar`);
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	const escopo = { installation_id: "42", repositorio: "equipe-e2e/projeto" };
	const corte = new Date(Date.now() - 60000).toISOString().replace(/\.\d{3}Z$/, ".123456780Z");
	const data = corte.slice(0, 10);
	const parametros = new URLSearchParams({ ...escopo, mapa_pagina: "2", mapa_snapshot: corte, mapa_dia: data, mapa_nivel: "alto" });
	const resposta = page.waitForResponse(
		(r) => new URL(r.url()).pathname === "/api/analises" && new URL(r.url()).searchParams.get("pagina") === "2",
	);
	await page.goto(`${origem}/?${parametros.toString()}`);
	const q = new URL((await resposta).url()).searchParams;
	expect(q.get("repositorio")).toBe(escopo.repositorio);
	expect(q.get("snapshot")).toBe(corte);
	await page.getByRole("dialog").getByRole("button", { name: "Anterior" }).click();
	await expect(page).toHaveURL(/mapa_pagina=1/);
	await page.reload();
	await expect(page.getByRole("dialog")).toContainText("Nenhuma análise neste recorte.");
});
