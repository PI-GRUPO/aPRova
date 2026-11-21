import type { Detalhe } from "./analises";
import { indisponivel } from "./fonte";
export type EtapaDemonstracao = {
	id: number;
	titulo: string;
	estado: "concluida" | "falhou" | "aguardando";
	momento: string;
	detalhe: string;
};
export type ComplementoDemonstracao = { etapas: EtapaDemonstracao[]; contextoCalibracao: string; ajudaCalibracao: string };
export type AnaliseDemonstracao = {
	origem: "mock";
	dados: Detalhe;

	complemento?: ComplementoDemonstracao;
};
export async function consultarAnaliseDemonstracao(id: string, signal: AbortSignal): Promise<AnaliseDemonstracao | null> {
	if (!__MOCKS__) return indisponivel();
	signal.throwIfAborted();
	const { analisesDemonstracao, complementosDemonstracao } = await import("@/mocks/analises-demonstracao");
	signal.throwIfAborted();
	let dados = analisesDemonstracao.find((a) => a.analise.id === id);
	if (!dados && /^mapa-\d{4}-\d{2}-\d{2}-(alto|medio|baixo)-[1-9]\d*$/.test(id)) {
		const { referenciasMapaMock } = await import("@/mocks/mapa");
		signal.throwIfAborted();
		dados = referenciasMapaMock.find((r) => r.detalhe.analise.id === id)?.detalhe;
	}
	return dados ? structuredClone({ origem: "mock", dados, complemento: complementosDemonstracao[id] }) : null;
}
// A volta sempre aponta à visão geral; nunca aceita um destino arbitrário.
export function retornoDaFila(valor: string | null) {
	const origem = new URLSearchParams(valor ?? "");
	const retorno = new URLSearchParams();
	for (const nome of ["installation_id", "fila", "repositorio", "mapa_dia", "mapa_nivel", "pagina", "snapshot"]) {
		const v = origem.get(nome);
		if (v) retorno.set(nome, v);
	}
	return `/${retorno.size ? `?${retorno}` : ""}`;
}
