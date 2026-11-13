import { eachDayOfInterval, isSameDay, startOfDay, subDays } from "date-fns";
import { agora, analises, inicioDaEspera, instalacao } from "./dados";
import type { Analise, Nivel, TipoDeFalha } from "./tipos";

export interface DiaDoPeriodo {
	dia: Date;
	baixo: number;
	medio: number;
	alto: number;
}

const tiposDeFalha: Exclude<TipoDeFalha, "nenhuma">[] = ["teste", "compilacao", "infraestrutura"];

export function filaDeAprovacao() {
	return analises
		.filter((analise) => analise.state === "aguardando_aprovacao")
		.map((analise) => ({ analise, desde: inicioDaEspera(analise) }))
		.sort((a, b) => a.desde.getTime() - b.desde.getTime());
}

export function analisesPorDia(): DiaDoPeriodo[] {
	const hoje = startOfDay(agora);
	const dias = eachDayOfInterval({
		start: subDays(hoje, instalacao.periodo_dias - 1),
		end: hoje,
	});

	return dias.map((dia) => {
		const doDia = analises.filter((analise) => analise.risk_gate && isSameDay(new Date(analise.received_at), dia));
		const contar = (nivel: Nivel) => doDia.filter((analise) => analise.risk_gate?.level === nivel).length;
		return {
			dia,
			baixo: contar("baixo"),
			medio: contar("medio"),
			alto: contar("alto"),
		};
	});
}

export function totaisPorNivel() {
	const comNivel = analises.filter(
		(analise): analise is Analise & { risk_gate: NonNullable<Analise["risk_gate"]> } => analise.risk_gate !== null,
	);
	const contar = (nivel: Nivel) => comNivel.filter((analise) => analise.risk_gate.level === nivel).length;
	return {
		baixo: contar("baixo"),
		medio: contar("medio"),
		alto: contar("alto"),
		comNivel: comNivel.length,
		semNivel: analises.length - comNivel.length,
		total: analises.length,
	};
}

export function bloqueio() {
	const avaliadas = analises.filter((analise) => analise.risk_gate !== null);
	const bloqueadas = avaliadas.filter((analise) => analise.risk_gate?.level === "alto");
	return {
		avaliadas: avaliadas.length,
		bloqueadas: bloqueadas.length,
		aguardando: bloqueadas.filter((analise) => analise.state === "aguardando_aprovacao").length,
		aprovadas: bloqueadas.filter((analise) => analise.approval?.decision === "aprovada").length,
		comAlteracoes: bloqueadas.filter((analise) => analise.approval?.decision === "alteracoes_solicitadas").length,
	};
}

export function falhasDoSandbox() {
	const execucoes = analises.flatMap((analise) => analise.sandbox_results);
	const porTipo = tiposDeFalha.map((tipo) => ({
		tipo,
		quantas: execucoes.filter((execucao) => execucao.failure_type === tipo).length,
	}));
	return {
		execucoes: execucoes.length,
		falhas: porTipo.reduce((soma, item) => soma + item.quantas, 0),
		porTipo: porTipo.sort((a, b) => b.quantas - a.quantas),
	};
}

export type ItemDaFila = ReturnType<typeof filaDeAprovacao>[number];

export const todosOsRepositorios = "todos";

export function repositoriosDaFila(fila: ItemDaFila[]) {
	return [...new Set(fila.map(({ analise }) => analise.repository))].sort();
}

function textoPesquisavel({ analise }: ItemDaFila) {
	const motivos = analise.risk_gate?.rules_hit.map((motivo) => motivo.id) ?? [];
	return [analise.repository, `#${analise.pr_number}`, analise.head_branch, analise.head_sha, ...motivos].join(" ").toLowerCase();
}

export function filtrarFila(fila: ItemDaFila[], busca: string, repositorio: string) {
	const termo = busca.trim().toLowerCase();
	return fila.filter(
		(item) => (repositorio === todosOsRepositorios || item.analise.repository === repositorio) && textoPesquisavel(item).includes(termo),
	);
}

export function ordenarPorEspera(fila: ItemDaFila[], maiorEsperaPrimeiro: boolean) {
	const sinal = maiorEsperaPrimeiro ? 1 : -1;
	return [...fila].sort((a, b) => sinal * (a.desde.getTime() - b.desde.getTime()));
}
