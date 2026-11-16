import { mensagensAcaoMock } from "@/mocks/acoes";
// Adapter exclusivamente local. Não chama /acoes-remotas nem resolve identidade Slack.
export type TipoAcao = keyof typeof mensagensAcaoMock;
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
	origem: "mock";
	idAcao: string;
	contexto: ContextoAcao;
	tipo: TipoAcao;
	justificativa: string;
	estado: "solicitacao_recebida";
	mensagem: string;
};
const resultados = new Map<string, ResultadoAcao>();
function escopo(c: ContextoAcao) {
	return JSON.stringify([c.usuario, c.instalacao, c.repo, c.pr, c.id, c.sha]);
}
export function indisponibilidadeSimulada(c: ContextoAcao, tipo: TipoAcao) {
	if (!c.atual) return "Esta análise foi substituída. Consulte a análise atual.";
	if (tipo === "reexecutar_analise")
		return c.estado === "falhou" && c.falhaInfra ? "" : "Reexecução disponível na demonstração somente para falha de infraestrutura.";
	if (c.proprioPR && tipo === "aprovar") return "O autor não pode aprovar o próprio PR.";
	return c.estado === "aguardando_aprovacao" ? "" : "Esta análise não está aguardando decisão humana.";
}
export function consultarAcaoSimulada(c: ContextoAcao) {
	const resultado = resultados.get(escopo(c));
	return resultado ? structuredClone(resultado) : undefined;
}
export async function registrarAcaoSimulada(
	c: ContextoAcao,
	tipo: TipoAcao,
	justificativa: string,
	idAcao: string,
): Promise<ResultadoAcao> {
	const motivo = indisponibilidadeSimulada(c, tipo);
	if (motivo) throw new Error(motivo);
	const texto = justificativa.trim();
	if (tipo !== "reexecutar_analise" && (!texto || Array.from(texto).length > 4000))
		throw new Error("Informe uma justificativa com até 4000 caracteres.");
	if (!idAcao || !c.instalacao || !c.id || !c.repo || !c.sha || !c.usuario)
		throw new Error("O contexto da análise está incompleto. Reabra a análise.");
	const anterior = consultarAcaoSimulada(c);
	if (anterior) {
		if (anterior.idAcao === idAcao && anterior.tipo === tipo && anterior.justificativa === texto) return structuredClone(anterior);
		throw new Error("Já existe uma solicitação simulada para esta análise e commit. Consulte o resultado registrado.");
	}
	const resultado: ResultadoAcao = {
		origem: "mock",
		idAcao,
		contexto: { ...c },
		tipo,
		justificativa: texto,
		estado: "solicitacao_recebida",
		mensagem: mensagensAcaoMock[tipo],
	};
	resultados.set(escopo(c), resultado);
	return structuredClone(resultado);
}
