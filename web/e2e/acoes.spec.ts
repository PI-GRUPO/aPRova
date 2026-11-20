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
test("ações pendentes não registram decisões pelo painel", async ({ page }) => {
	await preparar(page);
	for (const nome of ["Aprovar", "Pedir alterações", "Reexecutar análise"])
		await expect(page.getByRole("button", { name: nome, exact: true })).toBeDisabled();
	await expect(page.getByText("Nenhuma decisão humana registrada para esta análise.")).toBeVisible();
	const mensagem = await page.evaluate(async () => {
		const adapter = await import(`${location.origin}/src/servicos/acoes-simuladas.ts`);
		try {
			await adapter.registrarAcao.executar({}, "aprovar", "revisado", "id");
			return "aceita";
		} catch (e) {
			return String(e);
		}
	});
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
