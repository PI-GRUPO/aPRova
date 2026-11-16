import type { Detalhe } from "./analises";
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
	identidade: { instalacao: string; usuario: string };
	complemento?: ComplementoDemonstracao;
};
export async function consultarAnaliseDemonstracao(id: string, signal: AbortSignal): Promise<AnaliseDemonstracao | null> {
	signal.throwIfAborted();
	const { analisesDemonstracao, identidadeDemonstracao, complementosDemonstracao } = await import("@/mocks/analises-demonstracao");
	signal.throwIfAborted();
	const dados = analisesDemonstracao.find((a) => a.analise.id === id);
	return dados
		? structuredClone({ origem: "mock", dados, identidade: identidadeDemonstracao, complemento: complementosDemonstracao[id] })
		: null;
}
// A volta sempre aponta à visão geral; nunca aceita um destino arbitrário.
export function retornoDaFila(valor: string | null) {
	const origem = new URLSearchParams(valor ?? "");
	const retorno = new URLSearchParams();
	for (const nome of ["fila", "repositorio"]) {
		const v = origem.get(nome);
		if (v) retorno.set(nome, v);
	}
	return `/${retorno.size ? `?${retorno}` : ""}`;
}
