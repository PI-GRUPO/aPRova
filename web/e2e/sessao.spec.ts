import { expect, type Page, test } from "@playwright/test";

const sessao = {
	autenticado: true,
	login_disponivel: true,
	usuario: { id: 7, login: "pessoa-fixture" },
	instalacoes: [
		{ id: 42, conta: "equipe" },
		{ id: 43, conta: "outra-equipe" },
	],
	csrf_token: "a".repeat(43),
	expira_em: new Date(Date.now() + 3600000).toISOString(),
};
const relatorio = {
	installation_id: 42,
	repositorios: ["equipe/projeto"],
	inicio: "2026-11-09T00:00:00Z",
	fim: "2026-11-16T00:00:00Z",
	total_execucoes: 10,
	concluidas: 6,
	rejeitadas: 1,
	falhas: 1,
	em_andamento: 2,
	taxa_sucesso: 0.75,
	exigiram_aprovacao_humana: 3,
	risco_medio: 2,
	revertidas: 1,
};
async function simular(page: Page, estado: unknown = sessao) {
	await page.route("**/api/sessao", (r) => r.fulfill({ json: estado }));
	await page.route("**/api/web/relatorio-semanal?*", (r) =>
		r.fulfill({ json: { ...relatorio, installation_id: Number(new URL(r.request().url()).searchParams.get("installation_id")) } }),
	);
}
for (const largura of [1440, 390, 360])
	test(`login, conta e relatório semanal nos dois temas em ${largura}px`, async ({ page }) => {
		await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
		await simular(page, { autenticado: false, login_disponivel: true });
		await page.goto("/entrar");
		await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toHaveAttribute("href", "/auth/github");
		for (const tema of ["escuro", "claro"]) {
			if (tema === "claro") {
				await page.evaluate(() => {
					localStorage.setItem("aprova-tema", "claro");
				});
				await page.reload();
			}
			await page.evaluate(() => document.fonts.ready);
			await page.screenshot({ path: `../.capturas-mcp/f3/local-login-${largura}-${tema}.png`, fullPage: true });
		}
		await page.unroute("**/api/sessao");
		await page.route("**/api/sessao", (r) => r.fulfill({ json: sessao }));
		await page.goto("/relatorio-semanal");
		await expect(page.getByText("75%", { exact: true })).toBeVisible();
		await expect(page.getByText("6 concluídas de 8 finalizadas", { exact: true })).toBeVisible();
		for (const tema of ["claro", "escuro"]) {
			if (tema === "escuro") await page.getByRole("button", { name: "Usar tema escuro" }).click();
			await page.screenshot({ path: `../.capturas-mcp/f3/local-semanal-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
		}
		await page.goto("/conta");
		await expect(page.locator(".dados-identidade").getByText("@pessoa-fixture", { exact: true })).toBeVisible();
		await expect(page.getByText("equipe · Instalação 42", { exact: true })).toBeVisible();
		await page.screenshot({ path: `../.capturas-mcp/f3/local-conta-${largura}.png`, fullPage: true });
	});
test("troca de escopo descarta dados e URL adulterada não dispara relatório", async ({ page }) => {
	await simular(page);
	await page.goto("/relatorio-semanal");
	await expect(page.getByText("75%", { exact: true })).toBeVisible();
	await page.getByRole("combobox", { name: "Instalação", exact: true }).click();
	await page.getByRole("option", { name: "outra-equipe · 43" }).click();
	await expect(page).toHaveURL(/installation_id=43/);
	await expect(page.getByText("Instalação 43 · 1 repositórios autorizados na consulta")).toBeVisible();
	let consultas = 0;
	await page.route("**/api/web/relatorio-semanal?*", (r) => {
		consultas++;
		return r.abort();
	});
	await page.goto("/relatorio-semanal?installation_id=999");
	await expect(page.getByRole("heading", { name: "Instalação não autorizada" })).toBeVisible();
	expect(consultas).toBe(0);
	await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
});
test("taxa nula preserva ausência de denominador e zero execuções", async ({ page }) => {
	await simular(page);
	await page.route("**/api/web/relatorio-semanal?*", (r) =>
		r.fulfill({ json: { ...relatorio, total_execucoes: 0, concluidas: 0, rejeitadas: 0, falhas: 0, em_andamento: 0, taxa_sucesso: null } }),
	);
	await page.goto("/relatorio-semanal");
	await expect(page.getByText("Sem dados", { exact: true })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Sem análises neste período" })).toBeVisible();
});

test("prazo da sessão leva ao login com o dashboard aberto", async ({ page }) => {
	const agora = new Date();
	await page.clock.install({ time: agora });
	await simular(page, { ...sessao, expira_em: new Date(agora.getTime() + 60000).toISOString() });
	await page.goto("/relatorio-semanal?installation_id=42");
	await expect(page.getByText("75%", { exact: true })).toBeVisible();
	await page.getByRole("combobox", { name: "Instalação", exact: true }).click();
	await page.getByRole("option", { name: "outra-equipe · 43" }).click();
	await expect(page.getByText("Instalação 43 · 1 repositórios autorizados na consulta")).toBeVisible();
	const entradas = await page.evaluate(() => history.length);
	await page.clock.pauseAt(new Date(agora.getTime() + 55000));
	await page.route("**/api/sessao", () => {});
	const consultaPendente = page.waitForRequest("**/api/sessao");
	await page.evaluate(() => window.dispatchEvent(new Event("focus")));
	const consulta = await consultaPendente;
	const consultaCancelada = page.waitForEvent("requestfailed", (r) => r === consulta);
	await page.clock.fastForward("00:05");
	await consultaCancelada;
	await expect(page).toHaveURL(/\/entrar$/);
	await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toBeVisible();
	await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
	expect(await page.evaluate(() => history.length)).toBe(entradas);
	await page.goBack();
	await expect(page).toHaveURL(/\/entrar$/);
	await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toBeVisible();
	const consultasNoLogin = await page.evaluate(() => {
		const fetchOriginal = window.fetch;
		let consultas = 0;
		window.fetch = (...args) => {
			consultas++;
			return fetchOriginal(...args);
		};
		try {
			window.dispatchEvent(new Event("focus"));
			window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));
			return consultas;
		} finally {
			window.fetch = fetchOriginal;
		}
	});
	expect(consultasNoLogin).toBe(0);
});

test("revalidação em segundo plano mantém o relatório e evita consultas duplicadas", async ({ page }) => {
	await simular(page);
	let consultasRelatorio = 0;
	page.on("request", (req) => {
		if (new URL(req.url()).pathname === "/api/web/relatorio-semanal") consultasRelatorio++;
	});
	await page.goto("/relatorio-semanal");
	const taxa = page.getByText("75%", { exact: true });
	await expect(taxa).toBeVisible();
	const consultasIniciais = consultasRelatorio;

	let liberarConsulta!: () => void;
	const pendente = new Promise<void>((resolve) => {
		liberarConsulta = resolve;
	});
	let revalidacoes = 0;
	await page.route("**/api/sessao", async (r) => {
		revalidacoes++;
		await pendente;
		await r.fulfill({ json: { ...sessao, usuario: { ...sessao.usuario, login: "pessoa-atualizada" } } });
	});
	try {
		await page.evaluate(() => {
			window.dispatchEvent(new Event("focus"));
			window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));
		});
		await expect.poll(() => revalidacoes).toBe(1);
		await expect(taxa).toBeVisible();
		await expect(page.getByText("Confirmando sessão…")).toHaveCount(0);
	} finally {
		liberarConsulta();
	}
	await expect(page.locator("footer").getByText("GitHub · @pessoa-atualizada")).toBeVisible();
	await expect(taxa).toBeVisible();
	expect(revalidacoes).toBe(1);
	expect(consultasRelatorio).toBe(consultasIniciais);
});

for (const status of [200, 401, 503])
	test(`revalidação da sessão ao recuperar o foco trata HTTP ${status}`, async ({ page }) => {
		await simular(page);
		await page.goto("/relatorio-semanal");
		await expect(page.getByText("75%", { exact: true })).toBeVisible();
		await page.route("**/api/sessao", (r) => r.fulfill({ status, json: { autenticado: false, login_disponivel: true } }));
		await page.evaluate(() => window.dispatchEvent(new Event("focus")));
		if (status === 503) {
			await expect(page.getByRole("heading", { name: "Não foi possível confirmar sua sessão" })).toBeVisible();
			await expect(page).toHaveURL(/\/relatorio-semanal$/);
		} else {
			await expect(page).toHaveURL(/\/entrar$/);
			await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toBeVisible();
		}
		await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
	});

for (const status of [401, 403, 503])
	test(`relatório ${status} limpa acesso ou informa falha`, async ({ page }) => {
		await simular(page);
		await page.route("**/api/web/relatorio-semanal?*", (r) => r.fulfill({ status, json: { status: "falha" } }));
		await page.goto("/relatorio-semanal");
		if (status === 401) {
			await expect(page).toHaveURL(/\/entrar$/);
			await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toBeVisible();
		} else {
			await expect(page.getByRole("heading", { name: "Relatório indisponível" })).toBeVisible();
		}
		await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
	});
test("sessão sem instalação não consulta relatório", async ({ page }) => {
	await simular(page, { ...sessao, instalacoes: [] });
	let consultas = 0;
	await page.route("**/api/web/relatorio-semanal?*", (r) => {
		consultas++;
		return r.abort();
	});
	await page.goto("/relatorio-semanal");
	await expect(page.getByRole("heading", { name: "Nenhuma instalação autorizada" })).toBeVisible();
	expect(consultas).toBe(0);
});
test("fluxo HTTP com handlers reais de login, CSRF, relatório e logout", async ({ page, request }) => {
	await page.goto("http://127.0.0.1:18080/entrar");
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await expect(page).toHaveURL("http://127.0.0.1:18080/");
	await expect(page.getByRole("region", { name: "Métricas dos últimos 30 dias" })).toBeVisible();
	await expect(page.getByText("0 nesta página · 0 no total · página 1")).toBeVisible();
	for (const rota of ["configuracao", "sugestoes", "sla"]) {
		await page.goto(`http://127.0.0.1:18080/${rota}`);
		await expect(page.getByRole("heading", { name: "Recurso indisponível" })).toBeVisible();
	}
	await page.goto("http://127.0.0.1:18080/analises/demonstracao/123");
	await expect(page.getByRole("heading", { name: "Página não encontrada" })).toBeVisible();

	await page.goto("http://127.0.0.1:18080/relatorio-semanal");
	await expect(page.getByText("75%", { exact: true })).toBeVisible();
	const cookies = await page.context().cookies();
	const cookie = cookies.find((c) => c.name === "aprova_sessao");
	expect(cookie?.httpOnly).toBeTruthy();
	expect(cookie?.expires).toBeGreaterThan(Date.now() / 1000);
	await page.reload();
	await expect(page.getByText("75%", { exact: true })).toBeVisible();
	const cabecalho = cookies.map((c) => `${c.name}=${c.value}`).join("; ");
	const negado = await request.get("http://127.0.0.1:18080/api/web/relatorio-semanal?installation_id=999", {
		headers: { Cookie: cabecalho },
	});
	expect(negado.status()).toBe(403);
	const csrf = await request.post("http://127.0.0.1:18080/api/sessao/sair", {
		headers: { Cookie: cabecalho, Origin: "http://invasor.test" },
	});
	expect(csrf.status()).toBe(403);
	const outraAba = await page.context().newPage();
	await outraAba.goto("http://127.0.0.1:18080/relatorio-semanal");
	await expect(outraAba.getByText("75%", { exact: true })).toBeVisible();
	await page.getByRole("button", { name: "Sair", exact: true }).click();
	for (const aba of [page, outraAba]) {
		await expect(aba).toHaveURL("http://127.0.0.1:18080/entrar");
		await expect(aba.getByRole("link", { name: "Continuar com GitHub" })).toBeVisible();
	}
	await outraAba.close();
	await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
	const revogada = await request.get("http://127.0.0.1:18080/api/web/relatorio-semanal?installation_id=42", {
		headers: { Cookie: cabecalho },
	});
	expect(revogada.status()).toBe(401);
	await page.reload();
	await expect(page.getByRole("link", { name: "Continuar com GitHub" })).toBeVisible();
	await expect(page.getByRole("link", { name: "Continuar para o dashboard" })).toHaveCount(0);
	await page.getByRole("link", { name: "Continuar com GitHub" }).click();
	await expect(page).toHaveURL("http://127.0.0.1:18080/");
	await expect(page.getByRole("region", { name: "Métricas dos últimos 30 dias" })).toBeVisible();
});

test("relatório exige o repositório solicitado e logout exige confirmação", async ({ page }) => {
	await simular(page);
	await page.goto("/relatorio-semanal?installation_id=42&repositorio=equipe%2Foutro");
	await expect(page.getByRole("heading", { name: "Relatório indisponível" })).toBeVisible();
	await expect(page.getByText("75%", { exact: true })).toHaveCount(0);
	await page.route("**/api/sessao/sair", (r) => r.fulfill({ json: {} }));
	await page.getByRole("button", { name: "Sair", exact: true }).click();
	await expect(page.locator("footer").getByRole("alert")).toContainText("O serviço não confirmou a operação.");
	await expect(page.getByRole("button", { name: "Sair", exact: true })).toBeEnabled();
});
