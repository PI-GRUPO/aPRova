import { consultarJSON, ErroHTTP } from "./http";
import { objeto } from "./recibos";

type Instalacao = { id: number; conta: string };

export type Sessao =
	| { autenticado: false; login_disponivel: boolean }
	| {
			autenticado: true;
			login_disponivel: boolean;
			acoes_disponiveis?: boolean;
			usuario: { id: number; login: string };
			instalacoes: Instalacao[];
			csrf: string;
			expira_em: string;
	  };

function instalacaoValida(v: unknown): v is Instalacao {
	return objeto(v) && typeof v.id === "number" && Number.isSafeInteger(v.id) && v.id > 0 && typeof v.conta === "string";
}

export async function consultarSessao(signal: AbortSignal): Promise<Sessao> {
	const v = await consultarJSON("sessao", { signal });
	if (!objeto(v) || typeof v.autenticado !== "boolean" || typeof v.login_disponivel !== "boolean")
		throw new ErroHTTP("resposta", "Resposta de sessão inválida.");
	if (!v.autenticado) return { autenticado: false, login_disponivel: v.login_disponivel };

	const { usuario, instalacoes, acoes_disponiveis, expira_em, csrf_token: csrf } = v;
	if (
		!objeto(usuario) ||
		typeof usuario.id !== "number" ||
		!Number.isSafeInteger(usuario.id) ||
		usuario.id <= 0 ||
		typeof usuario.login !== "string" ||
		(acoes_disponiveis !== undefined && typeof acoes_disponiveis !== "boolean") ||
		!Array.isArray(instalacoes) ||
		!instalacoes.every(instalacaoValida) ||
		typeof csrf !== "string" ||
		csrf.length < 32 ||
		typeof expira_em !== "string" ||
		Number.isNaN(Date.parse(expira_em))
	)
		throw new ErroHTTP("resposta", "Resposta de sessão inválida.");
	return {
		autenticado: true,
		login_disponivel: v.login_disponivel,
		acoes_disponiveis,
		usuario: { id: usuario.id, login: usuario.login },
		instalacoes,
		csrf,
		expira_em,
	};
}
export async function encerrarSessao(csrf: string) {
	await consultarJSON("sair", { metodo: "POST", csrf, statusEsperado: 204 });
}
export type Relatorio = {
	installation_id: number;
	repositorios: string[];
	inicio: string;
	fim: string;
	total_execucoes: number;
	concluidas: number;
	rejeitadas: number;
	falhas: number;
	em_andamento: number;
	taxa_sucesso: number | null;
	exigiram_aprovacao_humana: number;
	risco_medio: number;
	revertidas: number;
};
export async function consultarRelatorio(id: string, inicio: string, signal: AbortSignal, repositorio = ""): Promise<Relatorio> {
	const q = new URLSearchParams({ installation_id: id });
	if (repositorio) q.set("repositorio", repositorio);
	if (inicio) q.set("inicio", inicio);
	const v = await consultarJSON("relatorioSemanal", { parametros: q, signal, privada: true });
	if (
		!objeto(v) ||
		v.installation_id !== Number(id) ||
		!Array.isArray(v.repositorios) ||
		!v.repositorios.every((r) => typeof r === "string") ||
		(!!repositorio && (v.repositorios.length !== 1 || v.repositorios[0] !== repositorio)) ||
		![
			"total_execucoes",
			"concluidas",
			"rejeitadas",
			"falhas",
			"em_andamento",
			"exigiram_aprovacao_humana",
			"risco_medio",
			"revertidas",
		].every((k) => typeof v[k] === "number" && Number.isSafeInteger(v[k]) && v[k] >= 0) ||
		typeof v.inicio !== "string" ||
		typeof v.fim !== "string" ||
		Number.isNaN(Date.parse(v.inicio)) ||
		Number.isNaN(Date.parse(v.fim)) ||
		!(v.taxa_sucesso === null || (typeof v.taxa_sucesso === "number" && v.taxa_sucesso >= 0 && v.taxa_sucesso <= 1))
	)
		throw new ErroHTTP("resposta", "Resposta de relatório inválida.");
	return v as Relatorio;
}
