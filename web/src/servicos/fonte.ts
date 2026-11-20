export type Escopo = { instalacao: string; repositorio: string };
export type Consulta<T> = (escopo: Escopo, filtros: URLSearchParams, signal: AbortSignal) => Promise<T>;

export class RecursoIndisponivel extends Error {
	constructor() {
		super("Recurso indisponível no momento.");
	}
}
export async function indisponivel(): Promise<never> {
	throw new RecursoIndisponivel();
}

export type Operacao<F> = { disponivel: boolean; executar: F };
export const operacaoPendente = { disponivel: false, executar: indisponivel };
