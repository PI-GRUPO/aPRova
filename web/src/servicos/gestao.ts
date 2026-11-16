import { useSyncExternalStore } from "react";
import {
	auditoriaMock,
	calibracoesMock,
	configuracaoInicialMock,
	ocorrenciasRegrasMock,
	referenciaMock,
	regrasMock,
	repositoriosMock,
	slaInicialMock,
	sugestoesMock,
} from "@/mocks/gestao";
import { listarRepositorios } from "./analises";

export type Configuracao = typeof configuracaoInicialMock;
export type Sugestao = Omit<(typeof sugestoesMock)[number], "estado"> & { estado: "pendente" | "em_revisao" | "recusada" | "aplicada" };
type EstadoGestao = {
	configuracoes: Record<string, Configuracao>;
	sla: Record<string, typeof slaInicialMock>;
	sugestoes: Sugestao[];
};
const inicial: EstadoGestao = {
	configuracoes: Object.fromEntries(repositoriosMock.map((r) => [r.nome, structuredClone(configuracaoInicialMock)])),
	sla: Object.fromEntries(repositoriosMock.map((r) => [r.nome, { ...slaInicialMock }])),
	sugestoes: structuredClone(sugestoesMock),
};
let estado = inicial;
const ouvintes = new Set<() => void>();
function assinar(fn: () => void) {
	ouvintes.add(fn);
	return () => {
		ouvintes.delete(fn);
	};
}
function atualizar(novo: EstadoGestao) {
	estado = novo;
	for (const fn of ouvintes) fn();
}
export function useGestaoSimulada() {
	return useSyncExternalStore(
		assinar,
		() => estado,
		() => inicial,
	);
}
export const repositoriosDemonstracao = repositoriosMock;
export function estatisticasRegras(repo: string, dias: number) {
	const inicio = referenciaMock.getTime() - dias * 86400000;
	return regrasMock.map((regra) => {
		const ocorrencias = ocorrenciasRegrasMock.filter(
			(o) => o.regra === regra.id && (!repo || o.repo === repo) && Date.parse(o.data) > inicio,
		);
		return { ...regra, acionamentos: ocorrencias.length, aprovadas: ocorrencias.filter((o) => o.aprovada).length };
	});
}
export function consultarCalibracoes(repo: string) {
	return calibracoesMock.filter((c) => !repo || c.repo === repo);
}
export function consultarAuditoria(filtros: { repo: string; dias: number; origem: string; decisao: string; recusadas: boolean }) {
	return auditoriaMock.filter(
		(a) =>
			(!filtros.repo || a.repo === filtros.repo) &&
			(!filtros.origem || a.origem === filtros.origem) &&
			(!filtros.decisao || a.decisao === filtros.decisao) &&
			(!filtros.recusadas || a.decisao === "Recusada") &&
			Date.parse(a.data) > referenciaMock.getTime() - filtros.dias * 86400000,
	);
}
export async function consultarSaudeRepositorios(instalacao: string | undefined, signal: AbortSignal) {
	if (instalacao) {
		const nomes = await listarRepositorios(instalacao, signal);
		return {
			origem: "api" as const,
			itens: nomes.map((nome) => ({
				id: nome,
				nome: nome.split("/").at(-1) ?? nome,
				completo: nome,
				check: "Não confirmado",
				ultima: "Não disponível",
				permissoes: "Não confirmadas",
			})),
		};
	}
	signal.throwIfAborted();
	return { origem: "mock" as const, itens: structuredClone(repositoriosMock) };
}
export async function salvarConfiguracaoSimulada(repo: string, limite: number, caminhos: string[], versao: number, proposta?: string) {
	const atual = estado.configuracoes[repo];
	if (!atual || atual.versao !== versao) throw new Error("A versão mudou. Reabra a configuração antes de salvar.");
	if (
		!Number.isSafeInteger(limite) ||
		limite < 1 ||
		caminhos.some((c) => !c.trim() || c.length > 200) ||
		new Set(caminhos).size !== caminhos.length
	)
		throw new Error("Confira o limite e os caminhos informados.");
	const nova = {
		limite,
		caminhos: [...caminhos],
		versao: versao + 1,
		historico: [{ versao: versao + 1, pessoa: "@demo", data: new Date().toLocaleString("pt-BR") }, ...atual.historico],
	};
	atualizar({
		...estado,
		configuracoes: { ...estado.configuracoes, [repo]: nova },
		sugestoes: estado.sugestoes.map((s) => (s.id === proposta && s.repo === repo ? { ...s, estado: "aplicada" } : s)),
	});
	return nova;
}
export async function salvarSlaSimulado(repo: string, primeiro: number, segundo: number) {
	if (!(repo in estado.sla) || !Number.isSafeInteger(primeiro) || !Number.isSafeInteger(segundo) || primeiro < 1 || segundo <= primeiro)
		throw new Error("O primeiro prazo deve ser positivo e o escalonamento deve ocorrer depois dele.");
	atualizar({ ...estado, sla: { ...estado.sla, [repo]: { primeiro, segundo } } });
}
export async function revisarSugestaoSimulada(id: string, acao: "recusada" | "em_revisao") {
	if (!estado.sugestoes.some((s) => s.id === id && s.estado === "pendente")) throw new Error("Esta proposta já foi revisada.");
	atualizar({ ...estado, sugestoes: estado.sugestoes.map((s) => (s.id === id ? { ...s, estado: acao } : s)) });
}
