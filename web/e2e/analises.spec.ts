import { expect, type Page, type Route, test } from "@playwright/test";

const id = "00000000-0000-4000-8000-000000000003";
const id2 = "00000000-0000-4000-8000-000000000004";
const reciboID = "00000000-0000-4000-8000-000000000002";
const instante = "2026-11-16T12:00:00Z";
const sessao = {
	autenticado: true,
	login_disponivel: true,
	usuario: { id: 7, login: "pessoa-fixture" },
	instalacoes: [{ id: 42, conta: "equipe" }],
	csrf_token: "a".repeat(43),
	expira_em: new Date(Date.now() + 3600000).toISOString(),
};
const analise = {
	id,
	repositorio: "equipe/projeto",
	numero_pr: 12,
	sha_head: "a".repeat(40),
	autor: "agente[bot]",
	estado: "aguardando_aprovacao",
	nivel: "alto",
	recebido_em: instante,
	criado_em: instante,
	atualizado_em: instante,
};
const tentativa = {
	tentativa: 1,
	compilou: true,
	testes_passaram: 10,
	testes_falharam: 1,
	saida: "FAIL: teste de referência",
	saida_truncada: true,
	duracao_ms: 42459,
	tipo_falha: "teste",
	criado_em: instante,
};
const detalhe = {
	installation_id: 42,
	analise,
	branch: null,
	url_pr: "https://github.com/equipe/projeto/pull/12",
	url_check: "https://github.com/equipe/projeto/runs/123",
	atual: true,
	gate: {
		nivel_deterministico: "alto",
		nivel: "alto",
		motivo: "Os testes falharam.",
		motivos: [
			{ id: "falha_sandbox", nivel: "alto", descricao: "Os testes ou a compilação falharam.", evidencias: ["1 de 11 testes falhou."] },
		],
		calibracao_estado: "motivo_nao_calibravel",
		calibracao_razao: "Redução bloqueada por falha de teste.",
		vizinhos: [],
	},
	etapas: [
		{ id: 1, de: "pendente", para: "testando_sandbox", detalhe: "Início da execução isolada.", criado_em: instante },
		{ id: 2, de: "testando_sandbox", para: "avaliando_risco", detalhe: "Contadores persistidos.", criado_em: instante },
	],
	tentativas: [tentativa],
	aprovacoes: [
		{
			id: 1,
			decisao: "aprovada",
			revisor: "pessoa",
			origem: "slack",
			justificativa: "Registro de teste.",
			commit_sha: "b".repeat(40),
			aplicavel_ao_commit: false,
			decidido_em: instante,
		},
	],
	recibo_id: reciboID,
	testes_estruturados_disponiveis: false,
	rede_isolada: null,
};
async function simular(page: Page, d: unknown = detalhe) {
	await page.route("**/api/sessao", (r) => r.fulfill({ json: sessao }));
	await page.route("**/api/repositorios?*", (r) =>
		r.fulfill({ json: { installation_id: 42, repositorios: ["equipe/projeto", "equipe/outro"] } }),
	);
	await page.route("**/api/analises**", (r) => {
		const u = new URL(r.request().url());
		if (u.pathname !== "/api/analises") return r.fulfill({ json: d });
		const p = Number(u.searchParams.get("pagina") ?? 1);
		return r.fulfill({
			json: {
				installation_id: 42,
				snapshot: instante,
				pagina: p,
				limite: 2,
				total: 3,
				tem_mais: p === 1,
				itens:
					p === 1
						? [analise, { ...analise, id: id2, numero_pr: 13, nivel: null, estado: "falhou" }]
						: [{ ...analise, id: "00000000-0000-4000-8000-000000000005", numero_pr: 14, nivel: "baixo", estado: "concluido" }],
			},
		});
	});
	await page.route("**/receipts/**", (r) => {
		const p = new URL(r.request().url()).pathname;
		return r.fulfill({
			json: p.endsWith("/keys")
				? { chaves: [] }
				: p.endsWith("/verify")
					? {
							id: reciboID,
							key_id: "chave",
							hash_sha256: "a".repeat(64),
							valido: true,
							verificacoes: ["hash", "chave", "assinatura", "conteudo", "elo"].map((nome) => ({ nome, resultado: "ok" })),
						}
					: {
							id: reciboID,
							task_run_id: id,
							key_id: "chave",
							hash_sha256: "a".repeat(64),
							payload_base64: "YQ==",
							assinatura_base64: "Yg==",
							recibo_anterior_id: null,
							payload: null,
						},
		});
	});
}
for (const largura of [1440, 390, 360])
	test(`lista e detalhe fiéis, acessíveis e responsivos em ${largura}px`, async ({ page }) => {
		await simular(page);
		await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
		await page.goto("/analises?installation_id=42");
		await expect(page.getByText("1 a 2 de 3 análises · página 1")).toBeVisible();
		if (largura >= 768) await expect(page.getByRole("table").getByText("Não avaliado", { exact: true })).toBeVisible();
		else await expect(page.getByRole("heading", { name: "equipe/projeto #13", exact: true })).toBeVisible();
		for (const tema of ["escuro", "claro"]) {
			if (tema === "claro") await page.getByRole("button", { name: "Usar tema claro" }).click();
			await page.evaluate(() => document.fonts.ready);
			await page.screenshot({ path: `../.capturas-mcp/f4a/local-lista-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
		}
		await page.getByRole("link", { name: largura >= 768 ? "equipe/projeto #12" : "Abrir análise equipe/projeto #12", exact: true }).click();
		await expect(page.getByRole("heading", { name: "equipe/projeto / PR #12", exact: true })).toBeVisible();
		await expect(page.getByText("Não informada nesta análise", { exact: true })).toBeVisible();
		await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
		for (const tema of ["claro", "escuro"]) {
			if (tema === "escuro") await page.getByRole("button", { name: "Usar tema escuro" }).click();
			await page.screenshot({ path: `../.capturas-mcp/f4a/local-detalhe-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			expect(
				await page
					.locator("img:visible")
					.evaluateAll((imgs) =>
						imgs.every(
							(img) =>
								img instanceof HTMLImageElement &&
								img.complete &&
								img.naturalWidth > 0 &&
								img.width === img.naturalWidth &&
								img.height === img.naturalHeight,
						),
					),
			).toBeTruthy();
		}
		await expect(page.getByText(/Registro de outro commit/)).toBeVisible();
		await page.getByRole("link", { name: "Abrir Check Run", exact: true }).focus();
		await page.keyboard.press("Tab");
		await expect(page.getByRole("button", { name: "Ajuda: Motivo e evidências", exact: true })).toBeFocused();
		await expect(page.getByRole("tooltip")).toContainText("O gate registra o nível decidido e as regras acionadas.");
		await page.keyboard.press("Escape");
		await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
		await expect(page.getByText("Recibo válido.", { exact: true })).toBeVisible();
		await page.getByRole("button", { name: "Testando sandbox", exact: true }).click();
		await expect(page.getByRole("dialog")).toContainText("Início da execução isolada.");
		await page.keyboard.press("Escape");
		await page.getByRole("link", { name: "Voltar às análises", exact: true }).click();
		await expect(page).toHaveURL(/\/analises\?installation_id=42$/);
	});
test("saída da tentativa exibe marcação como texto literal", async ({ page }) => {
	const log = "<strong>Marcador & diagnóstico</strong>\nFAIL: teste de referência";
	await simular(page, { ...detalhe, tentativas: [{ ...tentativa, saida: log }] });
	await page.goto(`/analises/${id}?installation_id=42`);
	await page.getByText("Saída da tentativa 1", { exact: true }).click();
	const saida = page.locator("pre.saida-analise");
	await expect(saida).toBeVisible();
	await expect.poll(() => saida.textContent()).toBe(log);
	await expect(saida.locator("*")).toHaveCount(0);
});
test("filtros, ordenação, paginação e retorno preservam URL e snapshot", async ({ page }) => {
	await simular(page);
	await page.goto("/analises?installation_id=42");
	await page.getByRole("button", { name: "Próxima", exact: true }).click();
	await expect(page).toHaveURL(/pagina=2/);
	await expect(page).toHaveURL(/snapshot=/);
	await expect(page.getByText("3 a 3 de 3 análises · página 2")).toBeVisible();
	await page.getByRole("combobox", { name: "Nível", exact: true }).click();
	await page.getByRole("option", { name: "Alto", exact: true }).click();
	await expect(page).toHaveURL(/nivel=alto/);
	expect(new URL(page.url()).searchParams.has("pagina")).toBeFalsy();
	expect(new URL(page.url()).searchParams.has("snapshot")).toBeFalsy();
	await page.getByLabel("Buscar PR", { exact: true }).fill("12");
	await page.getByRole("button", { name: "Buscar", exact: true }).click();
	await expect(page).toHaveURL(/numero_pr=12/);
	await page.getByRole("button", { name: "Recebida em", exact: true }).click();
	await expect(page).toHaveURL(/ordem=antigas/);
	await page.getByRole("link", { name: "equipe/projeto #12", exact: true }).click();
	await page.reload();
	await expect(page.getByRole("heading", { name: "equipe/projeto / PR #12", exact: true })).toBeVisible();
	await page.getByRole("link", { name: "Voltar às análises" }).click();
	await expect(page).toHaveURL(/numero_pr=12/);
	await page.getByRole("button", { name: "Limpar filtros" }).click();
	await expect(page).toHaveURL(/\/analises\?installation_id=42$/);
});

test("retorno do detalhe mantém a lista visível até atualizar em segundo plano", async ({ page }) => {
	await simular(page);
	await page.goto("http://127.0.0.1:18080/analises");
	const itemAnterior = page.getByRole("link", { name: "equipe/projeto #12", exact: true });
	await itemAnterior.click();
	await expect(page.getByRole("heading", { name: "equipe/projeto / PR #12", exact: true })).toBeVisible();
	let capturar!: (rota: Route) => void;
	const pendente = new Promise<Route>((resolve) => {
		capturar = resolve;
	});
	await page.route("**/api/analises?*", capturar, { times: 1 });
	await page.getByRole("link", { name: "Voltar às análises" }).click();
	const consulta = await pendente;
	await expect(itemAnterior).toBeVisible();
	await expect(page.getByText("Carregando análises…")).toHaveCount(0);
	await consulta.fulfill({
		json: {
			installation_id: 42,
			snapshot: instante,
			pagina: 1,
			limite: 2,
			total: 1,
			tem_mais: false,
			itens: [{ ...analise, numero_pr: 17 }],
		},
	});
	await expect(page.getByRole("link", { name: "equipe/projeto #17", exact: true })).toBeVisible();
	await expect(itemAnterior).toHaveCount(0);
});

test("revalidação negada descarta as leituras guardadas da sessão", async ({ page }) => {
	await simular(page);
	await page.goto("http://127.0.0.1:18080/analises?installation_id=42");
	const item = page.getByRole("link", { name: "equipe/projeto #12", exact: true });
	const titulo = page.getByRole("heading", { name: "equipe/projeto / PR #12", exact: true });
	await item.click();
	await expect(titulo).toBeVisible();
	await page.getByRole("link", { name: "Voltar às análises" }).click();
	let capturar!: (rota: Route) => void;
	const pendente = new Promise<Route>((resolve) => {
		capturar = resolve;
	});
	await page.route("**/api/analises/*?*", capturar, { times: 1 });
	await item.click();
	const consulta = await pendente;
	await expect(titulo).toBeVisible();
	await consulta.fulfill({ status: 403, json: {} });
	await expect(page.getByText("Você não tem acesso a esta instalação ou repositório.")).toBeVisible();
	await expect(titulo).toHaveCount(0);
	await page.route("**/api/analises?*", () => {});
	await page.getByRole("link", { name: "Voltar às análises" }).click();
	await expect(page.getByText("Carregando análises…")).toBeVisible();
	await expect(item).toHaveCount(0);
});

test("análise substituída sem gate, histórico ou recibo não inventa evidências", async ({ page }) => {
	await simular(page, {
		...detalhe,
		atual: false,
		gate: null,
		analise: { ...analise, nivel: null, estado: "falhou" },
		etapas: [],
		tentativas: [],
		aprovacoes: [],
		recibo_id: null,
		url_check: null,
	});
	await page.goto(`/analises/${id}?installation_id=42`);
	await expect(page.getByRole("heading", { name: "Esta análise foi substituída" })).toBeVisible();
	await expect(page.getByText(/Risco não avaliado. Nenhum gate/)).toBeVisible();
	await expect(page.getByText("Histórico de transições não disponível.")).toBeVisible();
	await expect(page.getByText("Nenhuma tentativa de sandbox registrada.")).toBeVisible();
	await expect(page.getByText("Recibo ainda não emitido para esta análise.")).toBeVisible();
});

const etapasAteSandbox = ["pendente", "mapeando_contexto", "testando_sandbox"];
for (const { caso, estados, marcadores } of [
	{
		caso: "concluída",
		estados: [...etapasAteSandbox, "avaliando_risco", "aplicando", "concluido"],
		marcadores: ["Concluída", "Concluída", "Concluída", "Concluída", "Concluída"],
	},
	{
		caso: "aguardando aprovação",
		estados: [...etapasAteSandbox, "avaliando_risco", "aguardando_aprovacao"],
		marcadores: ["Concluída", "Concluída", "Concluída", "Aguardando"],
	},
	{
		caso: "interrompida no sandbox",
		estados: [...etapasAteSandbox, "falhou"],
		marcadores: ["Concluída", "Interrompida", "Falhou"],
	},
	{
		caso: "rejeitada na revisão",
		estados: [...etapasAteSandbox, "avaliando_risco", "aguardando_aprovacao", "rejeitado"],
		marcadores: ["Concluída", "Concluída", "Concluída", "Concluída", "Rejeitada"],
	},
]) {
	test(`linha do tempo mantém os checks das etapas registradas com análise ${caso}`, async ({ page }) => {
		await simular(page, {
			...detalhe,
			analise: { ...analise, estado: estados.at(-1) },
			etapas: estados.slice(1).map((para, i) => ({
				id: i + 1,
				de: estados.at(i),
				para,
				detalhe: "",
				criado_em: instante,
			})),
		});
		await page.goto(`/analises/${id}?installation_id=42`);
		const etapas = page.locator(".timeline-analise .marcador-etapa");
		await expect(etapas).toHaveText(marcadores);
		const icones = etapas.locator("img");
		await expect(icones).toHaveCount(marcadores.length);
		await expect
			.poll(() => icones.evaluateAll((imagens) => imagens.every((i) => i instanceof HTMLImageElement && i.complete && i.naturalWidth > 0)))
			.toBe(true);
	});
}

for (const status of [404, 403, 503])
	test(`detalhe ${status} não mostra conteúdo privado e permite repetir`, async ({ page }) => {
		await simular(page);
		await page.route("**/api/analises/*?*", (r) => r.fulfill({ status, json: { status: "erro" } }));
		await page.goto(`/analises/${id}?installation_id=42`);
		await expect(page.getByRole("heading", { name: "Não foi possível abrir a análise" })).toBeVisible();
		await expect(page.getByText("agente[bot]", { exact: true })).toHaveCount(0);
		await page.unroute("**/api/analises/*?*");
		await page.getByRole("button", { name: "Tentar novamente" }).click();
		await expect(page.getByRole("heading", { name: "equipe/projeto / PR #12", exact: true })).toBeVisible();
	});
test("recibo inválido com HTTP 200 não vira sucesso", async ({ page }) => {
	await simular(page);
	await page.route("**/verify", (r) =>
		r.fulfill({
			json: {
				id: reciboID,
				key_id: "chave",
				hash_sha256: "a".repeat(64),
				valido: false,
				falha: "assinatura",
				verificacoes: ["hash", "chave", "assinatura", "conteudo", "elo"].map((nome, i) => ({
					nome,
					resultado: i < 2 ? "ok" : i === 2 ? "falhou" : "nao_verificada",
				})),
			},
		}),
	);
	await page.goto(`/analises/${id}?installation_id=42`);
	await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
	await expect(page.getByText("Não foi possível validar o recibo.", { exact: true })).toBeVisible();
	await expect(page.getByText("Recibo válido.", { exact: true })).toHaveCount(0);
});
test("fluxo HTTP real autentica, lista, abre detalhe e verifica recibo", async ({ page, request }) => {
	await page.goto("http://127.0.0.1:18080/entrar");
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await page.goto("http://127.0.0.1:18080/analises?installation_id=42");
	await page.getByRole("link", { name: "equipe-e2e/projeto #1", exact: true }).click();
	await expect(page.getByRole("heading", { name: "equipe-e2e/projeto / PR #1", exact: true })).toBeVisible();
	await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
	await expect(page.getByText("Recibo válido.", { exact: true })).toBeVisible();
	const cookies = (await page.context().cookies()).map((c) => `${c.name}=${c.value}`).join("; ");
	const negado = await request.get(`http://127.0.0.1:18080/api/analises/${id2}?installation_id=42`, { headers: { Cookie: cookies } });
	expect(negado.status()).toBe(404);
	const hmac = await request.get(`http://127.0.0.1:18080/api/analises?installation_id=42`, {
		headers: { Cookie: cookies, "X-Aprova-Assinatura": "sha256=invalida" },
	});
	expect(hmac.status()).toBe(401);
});

for (const largura of [390, 360, 320])
	for (const tema of ["escuro", "claro"]) {
		test(`cards de análises do Figma em ${largura}px no tema ${tema}`, async ({ page }) => {
			await simular(page);
			await page.setViewportSize({ width: largura, height: 844 });
			await page.addInitScript((t) => {
				localStorage.setItem("aprova-tema", t);
			}, tema);
			const referencias = [
				["aPRova-TESTES", 1],
				["cobranca-api", 214],
				["painel-entregas", 97],
				["cobranca-api", 217],
				["painel-entregas", 99],
			] as const;
			await page.route("**/api/analises?*", (r) =>
				r.fulfill({
					json: {
						installation_id: 42,
						snapshot: instante,
						pagina: 1,
						limite: 5,
						total: 5,
						tem_mais: false,
						itens: referencias.map(([repositorio, numero_pr], i) => ({
							...analise,
							id: `00000000-0000-4000-8000-${String(i + 3).padStart(12, "0")}`,
							repositorio,
							numero_pr,
						})),
					},
				}),
			);
			await page.goto("/analises?installation_id=42");
			const cards = page.getByRole("list", { name: "Análises autorizadas" });
			await expect(cards.getByRole("listitem")).toHaveCount(5);
			await expect(page.getByRole("table")).toHaveCount(0);
			await expect(page.getByRole("combobox", { name: "Instalação", exact: true })).toBeHidden();
			await expect(cards.getByRole("link")).toHaveCount(5);
			await page.evaluate(() => document.fonts.ready);
			const caixa = await cards.getByRole("listitem").first().boundingBox();
			expect(caixa).toMatchObject({ x: 16, y: 204, width: largura - 32, height: 120 });
			const botao = await cards.getByRole("link").first().boundingBox();
			expect(botao?.width).toBe(180);
			expect(botao?.height).toBe(32);
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			await page.screenshot({ path: `../.capturas-mcp/analises-mobile/local-${largura}-${tema}.png`, fullPage: true });
		});
	}

test("mobile preserva filtros, snapshot, teclado e retorno do detalhe", async ({ page }) => {
	await simular(page);
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/analises?installation_id=42&repositorio=equipe%2Fprojeto&nivel=alto&ordem=antigas");
	await page.getByRole("button", { name: "Próxima", exact: true }).click();
	await expect(page.getByRole("heading", { name: "equipe/projeto #14", exact: true })).toBeVisible();
	const destino = new URL(page.url());
	const id14 = "00000000-0000-4000-8000-000000000005";
	await page.route(`**/api/analises/${id14}?*`, (r) =>
		r.fulfill({ json: { ...detalhe, analise: { ...analise, id: id14, numero_pr: 14 } } }),
	);
	await page.getByText("Filtros e instalação", { exact: true }).focus();
	await page.keyboard.press("Enter");
	await expect(page.getByRole("combobox", { name: "Nível", exact: true })).toBeVisible();
	await page.getByLabel("Buscar PR", { exact: true }).fill("14");
	await page.getByText("Filtros e instalação", { exact: true }).focus();
	await page.keyboard.press("Enter");
	await expect(page.getByRole("combobox", { name: "Nível", exact: true })).toBeHidden();
	await page.getByText("Filtros e instalação", { exact: true }).click();
	await expect(page.getByLabel("Buscar PR", { exact: true })).toHaveValue("14");
	await page.getByRole("link", { name: "Abrir análise equipe/projeto #14", exact: true }).focus();
	await page.keyboard.press("Enter");
	await expect(page.getByRole("heading", { name: "equipe/projeto / PR #14", exact: true })).toBeVisible();
	await page.reload();
	await page.getByRole("link", { name: "Voltar às análises", exact: true }).click();
	expect(new URL(page.url()).search).toBe(destino.search);
	await expect(page.getByRole("heading", { name: "equipe/projeto #14", exact: true })).toBeVisible();
	await page.getByText("Filtros e instalação", { exact: true }).click();
	await page.getByRole("combobox", { name: "Nível", exact: true }).click();
	await page.getByRole("option", { name: "Baixo", exact: true }).click();
	await expect(page).toHaveURL(/nivel=baixo/);
	expect(new URL(page.url()).searchParams.has("snapshot")).toBeFalsy();
	expect(new URL(page.url()).searchParams.has("pagina")).toBeFalsy();
});

test("redimensionar mantém uma apresentação acessível sem refazer a consulta", async ({ page }) => {
	await simular(page);
	let consultas = 0;
	page.on("request", (r) => {
		if (new URL(r.url()).pathname === "/api/analises") consultas++;
	});
	await page.goto("/analises?installation_id=42");
	await expect(page.getByRole("table")).toBeVisible();
	await expect(page.getByRole("combobox", { name: "Instalação", exact: true })).toBeVisible();
	const consultasIniciais = consultas;
	await page.setViewportSize({ width: 390, height: 844 });
	await expect(page.getByRole("list", { name: "Análises autorizadas" })).toBeVisible();
	await expect(page.getByRole("combobox", { name: "Instalação", exact: true })).toBeHidden();
	await page.getByText("Filtros e instalação", { exact: true }).click();
	await expect(page.getByRole("combobox", { name: "Instalação", exact: true })).toBeVisible();
	await page.setViewportSize({ width: 1440, height: 900 });
	await expect(page.getByRole("table")).toBeVisible();
	await expect(page.getByRole("list", { name: "Análises autorizadas" })).toHaveCount(0);
	await expect(page.getByRole("combobox", { name: "Instalação", exact: true })).toBeVisible();
	expect(consultas).toBe(consultasIniciais);
});

for (const cenario of ["vazio", "erro", "sem-acesso"] as const)
	test(`mobile com ${cenario} não mostra cards ou demonstrações`, async ({ page }) => {
		await simular(page);
		await page.setViewportSize({ width: 320, height: 844 });
		await page.route("**/api/analises?*", (r) =>
			cenario === "vazio"
				? r.fulfill({ json: { installation_id: 42, snapshot: instante, pagina: 1, limite: 2, total: 0, tem_mais: false, itens: [] } })
				: r.fulfill({ status: cenario === "erro" ? 503 : 403, json: { status: "erro" } }),
		);
		await page.goto("/analises?installation_id=42");
		await expect(
			page.getByRole("heading", {
				name: cenario === "vazio" ? "Nenhuma análise encontrada" : "Não foi possível consultar análises",
				exact: true,
			}),
		).toBeVisible();
		await expect(page.getByRole("list", { name: "Análises autorizadas" })).toHaveCount(0);
		await expect(page.getByRole("link", { name: /Abrir análise/ })).toHaveCount(0);
	});
