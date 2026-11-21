import { type Operacao, operacaoPendente } from "./fonte";

export type TipoAcao = "aprovar" | "solicitar_alteracao" | "reexecutar_analise";
export type ContextoAcao = {
	instalacao: string;
	id: string;
	repo: string;
	pr: number;
	sha: string;
	usuario: string;
	nivel: string | null;
	estado: string;
	atual: boolean;
	proprioPR: boolean;
	falhaInfra: boolean;
};
export type ResultadoAcao = {
	idAcao: string;
	contexto: ContextoAcao;
	tipo: TipoAcao;
	justificativa: string;
	estado: "solicitacao_recebida";
	mensagem: string;
};
export const registrarAcao: Operacao<(c: ContextoAcao, tipo: TipoAcao, justificativa: string, idAcao: string) => Promise<ResultadoAcao>> =
	operacaoPendente;
export function motivoIndisponivel(c: ContextoAcao, tipo: TipoAcao) {
	if (!c.atual) return "Esta análise foi substituída.";
	if (c.proprioPR && tipo === "aprovar") return "O autor não pode aprovar o próprio PR.";
	if (!registrarAcao.disponivel) return "Ação indisponível no painel.";
	if (tipo === "reexecutar_analise") return c.estado === "falhou" && c.falhaInfra ? "" : "Reexecução indisponível para esta análise.";
	return c.estado === "aguardando_aprovacao" ? "" : "Esta análise não aguarda decisão humana.";
}
