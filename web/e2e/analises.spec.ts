import { expect, type Page, test } from "@playwright/test";

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
	tentativas: [
		{
			tentativa: 1,
			compilou: true,
			testes_passaram: 10,
			testes_falharam: 1,
			saida: "<script>window.invasao=true</script>\nFAIL: teste de referência",
			saida_truncada: true,
			duracao_ms: 42459,
			tipo_falha: "teste",
			criado_em: instante,
		},
	],
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
		await expect(page.getByText("Não avaliado", { exact: true })).toBeVisible();
		for (const tema of ["escuro", "claro"]) {
			if (tema === "claro") await page.getByRole("button", { name: "Usar tema claro" }).click();
			await page.evaluate(() => document.fonts.ready);
			await page.screenshot({ path: `../.capturas-mcp/f4a/local-lista-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
		}
		await page.getByRole("link", { name: "equipe/projeto #12", exact: true }).click();
		await expect(page.getByRole("heading", { name: "equipe/projeto / PR #12", exact: true })).toBeVisible();
		await expect(page.getByText("Não informada nesta análise", { exact: true })).toBeVisible();
		await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
		for (const tema of ["claro", "escuro"]) {
			if (tema === "escuro") await page.getByRole("button", { name: "Usar tema escuro" }).click();
			await page.screenshot({ path: `../.capturas-mcp/f4a/local-detalhe-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			expect(
				await page.locator("img:visible").evaluateAll((imgs) =>
					imgs.every((el) => {
						const i = el as HTMLImageElement;
						return i.complete && i.naturalWidth > 0 && i.width === i.naturalWidth && i.height === i.naturalHeight;
					}),
				),
			).toBeTruthy();
		}
		await page.getByText("Saída da tentativa 1", { exact: true }).click();
		await expect(page.locator("pre.saida-analise")).toContainText("<script>");
		expect(await page.evaluate(() => "invasao" in window)).toBeFalsy();
		await expect(page.getByText(/Registro de outro commit/)).toBeVisible();
		await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
		await expect(page.getByText("Recibo válido.", { exact: true })).toBeVisible();
		await page.getByRole("button", { name: "Testando sandbox", exact: true }).click();
		await expect(page.getByRole("dialog")).toContainText("Início da execução isolada.");
		await page.keyboard.press("Escape");
		await page.getByRole("link", { name: "Voltar às análises", exact: true }).click();
		await expect(page).toHaveURL(/\/analises\?installation_id=42$/);
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
