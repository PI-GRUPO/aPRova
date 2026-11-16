import { expect, type Page, test } from "@playwright/test";
import type { Detalhe } from "../src/servicos/analises";

const id = "00000000-0000-4000-8000-000000000074";
const instante = "2026-11-16T12:00:00Z";
const dados: Detalhe = {
	installation_id: 42,
	analise: {
		id,
		repositorio: "equipe/projeto",
		numero_pr: 1,
		sha_head: "6be930a".padEnd(40, "0"),
		autor: "agente[bot]",
		estado: "aguardando_aprovacao",
		nivel: "alto",
		recebido_em: instante,
		criado_em: instante,
		atualizado_em: instante,
	},
	branch: "teste-peso-zero",
	url_pr: "https://github.com/equipe/projeto/pull/1",
	url_check: null,
	atual: true,
	gate: {
		nivel_deterministico: "alto",
		nivel: "alto",
		motivo: "Os testes falharam.",
		motivos: [{ id: "falha_sandbox", nivel: "alto", descricao: "Teste falhou", evidencias: ["1 de 11 testes falharam"] }],
		calibracao_estado: "motivo_nao_calibravel",
		calibracao_razao: "Falha de teste",
		vizinhos: [],
	},
	etapas: [],
	tentativas: [],
	aprovacoes: [],
	recibo_id: null,
	testes_estruturados_disponiveis: false,
	rede_isolada: null,
};
async function preparar(page: Page, detalhe = dados, login = "revisor-demo") {
	await page.route("**/api/sessao", (r) =>
		r.fulfill({
			json: {
				autenticado: true,
				login_disponivel: true,
				usuario: { id: 7, login },
				instalacoes: [{ id: 42, conta: "equipe" }],
				csrf_token: "a".repeat(43),
				expira_em: new Date(Date.now() + 3600000).toISOString(),
			},
		}),
	);
	await page.route("**/api/analises/**", (r) => r.fulfill({ json: detalhe }));
	await page.goto(`/analises/${id}?installation_id=42`);
	await expect(page.getByRole("heading", { name: "equipe/projeto / PR #1", exact: true })).toBeVisible();
}
for (const largura of [1440, 390, 360])
	for (const tema of ["escuro", "claro"]) {
		test(`modais de decisão em ${largura}px no tema ${tema}`, async ({ page }) => {
			await page.setViewportSize({ width: largura, height: largura === 1440 ? 900 : 844 });
			await page.addInitScript((t) => localStorage.setItem("aprova-tema", t), tema);
			const escritas: string[] = [];
			page.on("request", (r) => {
				if (r.method() !== "GET") escritas.push(r.url());
			});
			await preparar(page);
			for (const [botao, texto, confirmar] of [
				["Aprovar", "Revisei os testes e os impactos apresentados.", "Confirmar aprovação"],
				["Pedir alterações", "Corrigir o cálculo para peso zero e incluir a validação desse cenário.", "Enviar solicitação"],
			]) {
				await page.getByRole("button", { name: botao, exact: true }).click();
				await expect(page.getByRole("dialog")).toContainText("Demonstração · ações locais, sem envio ao núcleo.");
				await page.getByRole("textbox", { name: "Justificativa obrigatória" }).fill(texto);
				await page.evaluate(() => document.fonts.ready);
				await page
					.getByRole("dialog")
					.screenshot({ path: `../.capturas-mcp/acoes/local-${botao === "Aprovar" ? "aprovar" : "alteracao"}-${largura}-${tema}.png` });
				expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
				await expect(page.getByRole("button", { name: confirmar, exact: true })).toBeVisible();
				await page.keyboard.press("Escape");
				await expect(page.getByRole("button", { name: botao, exact: true })).toBeFocused();
			}
			await page.getByRole("button", { name: "Aprovar", exact: true }).click();
			await expect(page.getByRole("textbox")).toHaveValue("Revisei os testes e os impactos apresentados.");
			await page.getByRole("button", { name: "Confirmar aprovação" }).click();
			await expect(page.getByRole("status")).toContainText("Nenhum review ou merge foi executado");
			await page.getByRole("dialog").screenshot({ path: `../.capturas-mcp/acoes/local-recebida-${largura}-${tema}.png` });
			await page.getByRole("button", { name: "Voltar à análise", exact: true }).click();
			await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeFocused();
			await expect(page.getByRole("button", { name: "Pedir alterações", exact: true })).toBeDisabled();
			await expect(page.getByText("Nenhuma decisão humana registrada para esta análise.")).toBeVisible();
			await expect(page.getByText("Aguardando aprovação", { exact: true })).toBeVisible();
			await expect(page.getByText("Recibo ainda não emitido para esta análise.")).toBeVisible();
			await preparar(page);
			await page.getByRole("button", { name: "Pedir alterações", exact: true }).click();
			await page.getByRole("textbox").fill("Corrigir o cenário antes de solicitar nova revisão.");
			await page.getByRole("button", { name: "Enviar solicitação", exact: true }).click();
			await expect(page.getByRole("status")).toContainText("Nenhum review foi publicado");
			await page.getByRole("dialog").screenshot({ path: `../.capturas-mcp/acoes/local-alteracao-registrada-${largura}-${tema}.png` });
			const infra: Detalhe = {
				...dados,
				analise: { ...dados.analise, estado: "falhou", nivel: null },
				gate: null,
				tentativas: [
					{
						tentativa: 1,
						compilou: false,
						testes_passaram: 0,
						testes_falharam: 0,
						saida: "Executor indisponível",
						saida_truncada: false,
						duracao_ms: 1000,
						tipo_falha: "infraestrutura",
						criado_em: instante,
					},
				],
			};
			await preparar(page, infra);
			await page.getByRole("button", { name: "Reexecutar análise", exact: true }).click();
			await expect(page.getByRole("dialog")).toContainText("Falha de infraestrutura recuperável");
			await page.getByRole("dialog").screenshot({ path: `../.capturas-mcp/acoes/local-reexecutar-${largura}-${tema}.png` });
			await page.getByRole("dialog").getByRole("button", { name: "Reexecutar análise", exact: true }).click();
			await expect(page.getByRole("status")).toContainText("Nenhuma tentativa foi enfileirada");
			await page.getByRole("dialog").screenshot({ path: `../.capturas-mcp/acoes/local-reexecucao-solicitada-${largura}-${tema}.png` });
			expect(escritas).toEqual([]);
		});
	}
test("justificativa vazia é rejeitada e rascunho sobrevive ao cancelamento", async ({ page }) => {
	await preparar(page);
	await page.getByRole("button", { name: "Pedir alterações", exact: true }).click();
	await page.getByRole("textbox").fill("   ");
	await page.getByRole("button", { name: "Enviar solicitação" }).click();
	await expect(page.getByRole("alert")).toContainText("Informe uma justificativa");
	await page.getByRole("textbox").fill("Falta o teste do caso de timeout.");
	await page.getByRole("button", { name: "Cancelar", exact: true }).click();
	await page.getByRole("button", { name: "Pedir alterações", exact: true }).click();
	await expect(page.getByRole("textbox")).toHaveValue("Falta o teste do caso de timeout.");
	await page.getByRole("button", { name: "Enviar solicitação" }).click();
	await expect(page.getByRole("status")).toContainText("Nenhum review foi publicado");
});
test("autor e análise substituída permanecem impedidos na demonstração", async ({ page }) => {
	await preparar(page, dados, "AGENTE[bot]");
	await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
	await expect(page.getByText("O autor não pode aprovar o próprio PR.")).toBeVisible();
	await page.unroute("**/api/analises/**");
	await page.route("**/api/analises/**", (r) => r.fulfill({ json: { ...dados, atual: false } }));
	await page.reload();
	for (const nome of ["Aprovar", "Pedir alterações", "Reexecutar análise"])
		await expect(page.getByRole("button", { name: nome, exact: true })).toBeDisabled();
});
for (const falha of ["infraestrutura", "teste", "compilacao"]) {
	test(`reexecução com falha de ${falha}`, async ({ page }) => {
		const detalhe = {
			...dados,
			analise: { ...dados.analise, estado: "falhou" },
			gate: null,
			tentativas: [
				{
					tentativa: 1,
					compilou: false,
					testes_passaram: 0,
					testes_falharam: 0,
					saida: "Executor indisponível",
					saida_truncada: false,
					duracao_ms: 1000,
					tipo_falha: falha,
					criado_em: instante,
				},
			],
		};
		await preparar(page, detalhe);
		const botao = page.getByRole("button", { name: "Reexecutar análise", exact: true });
		if (falha !== "infraestrutura") {
			await expect(botao).toBeDisabled();
			return;
		}
		await botao.click();
		await expect(page.getByRole("dialog")).toContainText("Falha de infraestrutura recuperável");
		await page.getByRole("dialog").getByRole("button", { name: "Reexecutar análise", exact: true }).click();
		await expect(page.getByRole("status")).toContainText("Nenhuma tentativa foi enfileirada");
		await page.getByRole("button", { name: "Voltar à análise", exact: true }).click();
		await expect(page.getByText("Falhou", { exact: true })).toBeVisible();
	});
}
test("adapter repete a mesma solicitação sem duplicar e isola análise, SHA, ator e instalação", async ({ page }) => {
	await preparar(page);
	const resultado = await page.evaluate(async () => {
		const modulo = await import(`${location.origin}/src/servicos/acoes-simuladas.ts`);
		const contexto = {
			instalacao: "42",
			id: "analise-mock",
			repo: "equipe/projeto",
			pr: 1,
			sha: "sha1",
			usuario: "7",
			nivel: "alto",
			estado: "aguardando_aprovacao",
			atual: true,
			proprioPR: false,
			falhaInfra: false,
		};
		const a = await modulo.registrarAcaoSimulada(contexto, "aprovar", "Justificativa de teste", "id-estavel");
		const b = await modulo.registrarAcaoSimulada(contexto, "aprovar", "Justificativa de teste", "id-estavel");
		const erros: string[] = [];
		for (const [texto, id] of [
			["Outra justificativa", "id-estavel"],
			["Justificativa de teste", "novo-id"],
			["a".repeat(4001), "longo"],
		]) {
			try {
				await modulo.registrarAcaoSimulada(contexto, "aprovar", texto, id);
			} catch (e) {
				erros.push(String(e));
			}
		}
		const outros = await Promise.all(
			[{ sha: "sha2" }, { instalacao: "43" }, { usuario: "8" }, { id: "outra" }].map((c) =>
				modulo.registrarAcaoSimulada({ ...contexto, ...c }, "aprovar", "Outra ação", "id-estavel"),
			),
		);
		return { a, b, erros, outros };
	});
	expect(resultado.a).toEqual(resultado.b);
	expect(resultado.erros).toHaveLength(3);
	expect(resultado.outros).toHaveLength(4);
	expect(resultado.a.origem).toBe("mock");
	expect(resultado.a.contexto.sha).toBe("sha1");
});

test("clique duplo mantém uma solicitação e outro SHA exige nova confirmação", async ({ page }) => {
	await preparar(page);
	await page.getByRole("button", { name: "Aprovar", exact: true }).click();
	await page.getByRole("textbox").fill("Revisei as evidências deste commit.");
	await page.getByRole("button", { name: "Confirmar aprovação" }).evaluate((el: HTMLButtonElement) => {
		el.click();
		el.click();
	});
	await expect(page.getByRole("status")).toContainText("Nenhum review ou merge foi executado");
	await page.getByRole("button", { name: "Voltar à análise", exact: true }).click();
	await page.getByRole("button", { name: "Aprovar", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("Nenhum review ou merge foi executado");
	await expect(page.getByRole("button", { name: "Confirmar aprovação" })).toHaveCount(0);
	await page.unroute("**/api/analises/**");
	await page.route("**/api/analises/**", (r) => r.fulfill({ json: { ...dados, analise: { ...dados.analise, sha_head: "b".repeat(40) } } }));
	await page.reload();
	await expect(page.getByRole("dialog")).toHaveCount(0);
	await page.getByRole("button", { name: "Aprovar", exact: true }).click();
	await expect(page.getByRole("textbox")).toHaveValue("");
	await expect(page.getByRole("dialog")).toContainText("bbbbbbb");
});

test("conflito mantém a justificativa e não mostra sucesso de uma nova ação", async ({ page }) => {
	await preparar(page);
	await page.getByRole("button", { name: "Aprovar", exact: true }).click();
	await page.getByRole("textbox").fill("Texto que precisa ser preservado.");
	await page.evaluate(async ({ analise }) => {
		const m = await import(`${location.origin}/src/servicos/acoes-simuladas.ts`);
		await m.registrarAcaoSimulada(
			{
				instalacao: "42",
				id: analise.id,
				repo: analise.repositorio,
				pr: analise.numero_pr,
				sha: analise.sha_head,
				usuario: "7",
				nivel: "alto",
				estado: "aguardando_aprovacao",
				atual: true,
				proprioPR: false,
				falhaInfra: false,
			},
			"aprovar",
			"Ação já recebida",
			"outra-submissao",
		);
	}, dados);
	await page.getByRole("button", { name: "Confirmar aprovação" }).click();
	await expect(page.getByRole("alert")).toContainText("Já existe uma solicitação simulada");
	await expect(page.getByRole("textbox")).toHaveValue("Texto que precisa ser preservado.");
	await expect(page.getByRole("status")).toHaveCount(0);
	await page.getByRole("button", { name: "Consultar solicitação registrada", exact: true }).click();
	await expect(page.getByRole("status")).toContainText("Solicitação de aprovação recebida na simulação");
	await page.getByRole("button", { name: "Voltar à análise", exact: true }).click();
	await page.reload();
	// Reload restaura o ambiente de demonstração, sem consultar um comando real.
	await page.getByRole("button", { name: "Aprovar", exact: true }).click();
	await expect(page.getByRole("textbox")).toHaveValue("");
});
