import { type Consulta, type Escopo, indisponivel, type Operacao, operacaoPendente } from "./fonte";
export type Configuracao = {
	limite: number;
	caminhos: string[];
	versao: number;
	historico: { versao: number; pessoa: string; data: string }[];
};
export type Sugestao = {
	id: string;
	repo: string;
	titulo: string;
	origem: string;
	estado: "pendente" | "em_revisao" | "recusada" | "aplicada";
	resumo: string;
	evidencia: string;
	limiteProposto: number | null;
};
export type Gestao = {
	repositorios: string[];
	configuracoes: Record<string, Configuracao>;
	sla: Record<string, { primeiro: number; segundo: number }>;
	sugestoes: Sugestao[];
	regras: { id: string; descricao: string; nivel: string; acionamentos: number; aprovadas: number }[];
	calibracoes: { id: string; nome: string; risco: string; resultado: string }[];
	auditoria: { id: string; data: string; pessoa: string; acao: string; pr: number; origem: string; decisao: string; motivo: string }[];
};
const leituraGestao: Consulta<Gestao> = __MOCKS__
	? async (escopo, filtros, signal) => {
			const { lerGestaoMock } = await import("@/mocks/gestao");
			signal.throwIfAborted();
			return lerGestaoMock(escopo.repositorio, filtros);
		}
	: indisponivel;
export const consultasGestao = {
	"/regras": leituraGestao,
	"/auditoria": leituraGestao,
	"/configuracao": leituraGestao,
	"/sugestoes": leituraGestao,
	"/sla": leituraGestao,
} satisfies Record<string, Consulta<Gestao>>;
export const salvarConfiguracao: Operacao<
	(escopo: Escopo, limite: number, caminhos: string[], versao: number, proposta?: string) => Promise<Configuracao>
> = operacaoPendente;
export const salvarSla: Operacao<(escopo: Escopo, primeiro: number, segundo: number) => Promise<void>> = operacaoPendente;
export const revisarSugestao: Operacao<(escopo: Escopo, id: string, acao: "recusada" | "em_revisao") => Promise<void>> = operacaoPendente;
