import type { Consulta } from "./fonte";
import { metricasReais } from "./metricas";
export type VisaoGeralDados = {
	origem: "mock" | "api";
	atualizadoEm: string;
	periodo: string;
	metricas: { rotulo: string; valor: string; unidade: string; contexto: string; contextoMobile?: string; proporcao?: number }[];
	bloqueadas: { nome: string; quantidade: number }[];
	falhas: { nome: string; quantidade: number }[];
	ultimoRecibo: { repositorio: string; pr: number; hash: string; anterior?: string; id?: string; data: string } | null;
	mapa: { data: string; alto: number; medio: number; baixo: number }[];
};

export const consultarVisaoGeral: Consulta<VisaoGeralDados> = __MOCKS__
	? async (_escopo, _filtros, signal) => {
			const { visaoGeralMock } = await import("@/mocks/visao-geral");
			signal.throwIfAborted();
			return structuredClone(visaoGeralMock);
		}
	: metricasReais;

export function destinoAnalise(id: string, origem: "mock" | "api", instalacao: string, retorno: URLSearchParams) {
	const caminho = origem === "mock" ? "/analises/demonstracao" : "/analises";
	return `${caminho}/${encodeURIComponent(id)}?${new URLSearchParams({ installation_id: instalacao, retorno: retorno.toString() })}`;
}
