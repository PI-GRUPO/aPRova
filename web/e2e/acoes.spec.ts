import { expect, type Page, test } from "@playwright/test";
import type { ContextoAcao } from "../src/servicos/acoes-analise";
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
async function preparar(page: Page, detalhe = dados, login = "revisor-demo", origem = "") {
	await page.route("**/api/sessao", (r) =>
		r.fulfill({
			json: {
				autenticado: true,
				login_disponivel: true,
				acoes_disponiveis: !!origem,
				usuario: { id: 7, login },
				instalacoes: [{ id: 42, conta: "equipe" }],
				csrf_token: "a".repeat(43),
				expira_em: new Date(Date.now() + 3600000).toISOString(),
			},
		}),
	);
	await page.route("**/api/analises/**", (r) => r.fulfill({ json: detalhe }));
	await page.goto(`${origem}/analises/${id}?installation_id=42`);
	await expect(page.getByRole("heading", { name: "equipe/projeto / PR #1", exact: true })).toBeVisible();
}
test("ações pendentes não registram decisões pelo painel", async ({ page }) => {
	await preparar(page);
	for (const nome of ["Aprovar", "Pedir alterações", "Reexecutar análise"])
		await expect(page.getByRole("button", { name: nome, exact: true })).toBeDisabled();
	await expect(page.getByText("Nenhuma decisão humana registrada para esta análise.")).toBeVisible();
	const contexto: ContextoAcao = {
		instalacao: String(dados.installation_id),
		id,
		repo: dados.analise.repositorio,
		pr: dados.analise.numero_pr,
		sha: dados.analise.sha_head,
		usuario: "revisor-demo",
		nivel: dados.analise.nivel,
		estado: dados.analise.estado,
		atual: dados.atual,
		proprioPR: false,
		falhaInfra: false,
	};
	const mensagem = await page.evaluate(async (contexto) => {
		const adapter = await import("/src/servicos/acoes-analise");
		try {
			await adapter.registrarAcao.executar(contexto, "aprovar", "revisado", "id", "csrf");
			return "aceita";
		} catch (e) {
			return String(e);
		}
	}, contexto);
	expect(mensagem).toContain("indisponível");
});
test("autor e análise substituída permanecem impedidos", async ({ page }) => {
	await preparar(page, dados, "AGENTE[bot]");
	await expect(page.getByRole("button", { name: "Aprovar", exact: true })).toBeDisabled();
	await expect(page.getByText("O autor não pode aprovar o próprio PR.")).toBeVisible();
	await page.unroute("**/api/analises/**");
	await page.route("**/api/analises/**", (r) => r.fulfill({ json: { ...dados, atual: false } }));
	await page.reload();
	for (const nome of ["Aprovar", "Pedir alterações", "Reexecutar análise"])
		await expect(page.getByRole("button", { name: nome, exact: true })).toBeDisabled();
});

for (const { nome, botao, confirmar, tipo } of [
	{ nome: "aprovação", botao: "Aprovar", confirmar: "Confirmar aprovação", tipo: "aprovar" },
	{ nome: "pedido de alterações", botao: "Pedir alterações", confirmar: "Enviar solicitação", tipo: "solicitar_alteracao" },
	{ nome: "reexecução", botao: "Reexecutar análise", confirmar: "Reexecutar análise", tipo: "reexecutar_analise" },
]) {
	test(`painel real envia ${nome} para a sessão e o commit exibidos`, async ({ page }) => {
		const reexecucao = tipo === "reexecutar_analise";
		let numeroDePedidos = 0;
		const justificativa = "Conferi as evidências deste commit.";
		const origem = "http://127.0.0.1:18080";
		const detalhe: Detalhe = !reexecucao
			? dados
			: {
					...dados,
					analise: { ...dados.analise, estado: "falhou", nivel: null },
					gate: null,
					tentativas: [
						{
							tentativa: 3,
							compilou: false,
							testes_passaram: 0,
							testes_falharam: 0,
							saida: "serviço indisponível",
							saida_truncada: false,
							duracao_ms: 1000,
							tipo_falha: "infraestrutura",
							criado_em: instante,
						},
					],
				};
		await page.setViewportSize({ width: reexecucao ? 390 : 1440, height: 900 });
		await page.route(`**/api/web/analises/${id}/acoes?*`, async (r) => {
			numeroDePedidos++;
			const req = r.request();
			expect(req.method()).toBe("POST");
			expect(new URL(req.url()).searchParams.get("installation_id")).toBe("42");
			expect(req.headers()["x-csrf-token"]).toBe("a".repeat(43));
			const corpo = req.postDataJSON();
			expect(corpo).toEqual({
				id_acao: expect.stringMatching(/^[0-9a-f-]{36}$/),
				tipo,
				commit_sha: detalhe.analise.sha_head,
				justificativa: reexecucao ? "" : justificativa,
			});
			await r.fulfill({ status: 202, json: { id_acao: corpo.id_acao, status: "solicitacao_recebida" } });
		});
		await preparar(page, detalhe, "revisor-demo", origem);
		await page.getByRole("button", { name: botao, exact: true }).click();
		const dialogo = page.getByRole("dialog");
		const enviar = dialogo.getByRole("button", { name: confirmar, exact: true });
		if (reexecucao) {
			await expect(dialogo.getByLabel("Justificativa obrigatória")).toHaveCount(0);
		} else {
			await enviar.click();
			await expect(dialogo.getByRole("alert")).toContainText("Informe uma justificativa");
			expect(numeroDePedidos).toBe(0);
			await dialogo.getByLabel("Justificativa obrigatória").fill(justificativa);
		}
		await enviar.click();
		await expect(dialogo.getByRole("status")).toContainText(reexecucao ? "Nova análise recebida pela fila" : "Review enviada ao GitHub");
		expect(numeroDePedidos).toBe(1);
		if (reexecucao) {
			await dialogo.getByRole("link", { name: "Ver análises deste PR" }).click();
			await expect(page).toHaveURL(`${origem}/analises?installation_id=42&repositorio=equipe%2Fprojeto&numero_pr=1`);
		}
	});
}
