import { type Consulta, indisponivel } from "./fonte";
export type VisaoGeralDados = {
	origem: "mock" | "api";
	atualizadoEm: string;
	periodo: string;
	metricas: { rotulo: string; valor: string; unidade: string; contexto: string; contextoMobile?: string; proporcao?: number }[];
	bloqueadas: { nome: string; quantidade: number }[];
	falhas: { nome: string; quantidade: number }[];
	ultimoRecibo: { repositorio: string; pr: number; hash: string; anterior: string; data: string };
	mapa: { data: string; alto: number; medio: number; baixo: number }[];
};

// A API de métricas de 30 dias ainda não existe. O agregado semanal não
// corresponde a este contrato: manter a simulação explícita e substituível.
export const consultarVisaoGeral: Consulta<VisaoGeralDados> = __MOCKS__
	? async (_escopo, _filtros, signal) => {
			const { visaoGeralMock } = await import("@/mocks/visao-geral");
			signal.throwIfAborted();
			return structuredClone(visaoGeralMock);
		}
	: indisponivel;

export function destinoAnalise(id: string, origem: "mock" | "api", instalacao: string, retorno: URLSearchParams) {
	const caminho = origem === "mock" ? "/analises/demonstracao" : "/analises";
	return `${caminho}/${encodeURIComponent(id)}?${new URLSearchParams({ installation_id: instalacao, retorno: retorno.toString() })}`;
}
