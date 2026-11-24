import { indisponivel } from "./fonte";
import { consultarJSON, ErroHTTP } from "./http";
import { invalido, registro } from "./painel";

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

const mensagensDeErro = {
	autor_do_proprio_pr: "O autor não pode aprovar o próprio PR.",
	analise_incompativel: "Esta análise já não está apta para a ação. Atualize a página e confira o estado do PR.",
	corpo_invalido: "Confira os dados e a justificativa da ação.",
};

export const registrarAcao = {
	disponivel: !__MOCKS__,
	async executar(contexto: ContextoAcao, tipo: TipoAcao, justificativa: string, idAcao: string, csrf: string): Promise<ResultadoAcao> {
		if (!registrarAcao.disponivel || !csrf) return indisponivel();
		const q = new URLSearchParams({ installation_id: contexto.instalacao });
		let repetida = false;
		try {
			const r = registro(
				await consultarJSON(`/api/web/analises/${encodeURIComponent(contexto.id)}/acoes?${q.toString()}`, {
					metodo: "POST",
					privada: true,
					csrf,
					timeout: 30000,
					statusEsperado: 202,
					lerCodigo: true,
					corpo: { id_acao: idAcao, tipo, commit_sha: contexto.sha, justificativa },
				}),
			);
			if (r.id_acao !== idAcao || r.status !== "solicitacao_recebida") return invalido();
		} catch (e) {
			if (!(e instanceof ErroHTTP)) throw e;
			if (e.codigo !== "acao_repetida") {
				const codigo = e.codigo;
				if (codigo && Object.hasOwn(mensagensDeErro, codigo)) throw new Error(mensagensDeErro[codigo as keyof typeof mensagensDeErro]);
				throw e;
			}
			repetida = true;
		}
		return {
			idAcao,
			contexto,
			tipo,
			justificativa,
			estado: "solicitacao_recebida",
			mensagem: repetida
				? "Já existe uma solicitação com esse identificador. Consulte a análise para acompanhar."
				: tipo === "reexecutar_analise"
					? "Nova análise recebida pela fila. A execução será atualizada quando o processamento começar."
					: "Review enviada ao GitHub. A análise será atualizada após o retorno do GitHub.",
		};
	},
};
export function motivoIndisponivel(c: ContextoAcao, tipo: TipoAcao) {
	if (!c.atual) return "Esta análise foi substituída.";
	if (c.proprioPR && tipo === "aprovar") return "O autor não pode aprovar o próprio PR.";
	if (!registrarAcao.disponivel) return "Ação indisponível no painel.";
	if (tipo === "reexecutar_analise") return c.estado === "falhou" && c.falhaInfra ? "" : "Reexecução indisponível para esta análise.";
	return c.estado === "aguardando_aprovacao" && c.nivel === "alto" ? "" : "Esta análise não aguarda decisão humana.";
}
