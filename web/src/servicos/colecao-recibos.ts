import { type Consulta, type Escopo, indisponivel, type Operacao, operacaoPendente } from "./fonte";
export type RegistroRecibo = { id: string; repo: string; pr: number; sha: string; emissao: string };
export const consultarColecao: Consulta<{ repositorios: string[]; itens: RegistroRecibo[] }> = __MOCKS__
	? async ({ repositorio }, filtros, signal) => {
			const [{ recibosDemonstracao }, { referenciaMock }] = await Promise.all([import("@/mocks/recibos"), import("@/mocks/gestao")]);
			signal.throwIfAborted();
			const periodo = filtros.get("periodo") ?? "30";
			const inicio = referenciaMock.getTime() - (["7", "30", "90"].includes(periodo) ? Number(periodo) : 30) * 86400000;
			return {
				repositorios: [...new Set(recibosDemonstracao.map((r) => r.repo))],
				itens: recibosDemonstracao.filter((r) => (!repositorio || r.repo === repositorio) && Date.parse(r.emissao) > inicio),
			};
		}
	: indisponivel;
export const verificarCadeia: Operacao<
	(escopo: Escopo, signal: AbortSignal) => Promise<{ valida: boolean; primeiroReciboInvalido: string | null }>
> = operacaoPendente;
export const exportarColecao: Operacao<(escopo: Escopo, dias: number, formato: "json" | "csv", signal: AbortSignal) => Promise<Blob>> =
	operacaoPendente;
