import { consultarJSON, ErroHTTP } from "./http";
import { objeto } from "./recibos";
export type Sessao =
	| { autenticado: false; login_disponivel: boolean }
	| {
			autenticado: true;
			login_disponivel: boolean;
			usuario: { id: number; login: string };
			instalacoes: { id: number; conta: string }[];
			csrf_token: string;
			expira_em: string;
	  };
export async function consultarSessao(signal: AbortSignal): Promise<Sessao> {
	const v = await consultarJSON("/api/sessao", { signal });
	if (!objeto(v) || typeof v.autenticado !== "boolean" || typeof v.login_disponivel !== "boolean")
		throw new ErroHTTP("resposta", "Resposta de sessão inválida.");
	if (
		v.autenticado &&
		(!objeto(v.usuario) ||
			typeof v.usuario.id !== "number" ||
			typeof v.usuario.login !== "string" ||
			!Array.isArray(v.instalacoes) ||
			!v.instalacoes.every((i) => objeto(i) && typeof i.id === "number" && typeof i.conta === "string") ||
			typeof v.csrf_token !== "string" ||
			v.csrf_token.length < 32 ||
			typeof v.expira_em !== "string" ||
			Number.isNaN(Date.parse(v.expira_em)))
	)
		throw new ErroHTTP("resposta", "Resposta de sessão inválida.");
	return v as Sessao;
}
export async function encerrarSessao(csrf: string) {
	await consultarJSON("/api/sessao/sair", { metodo: "POST", csrf });
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
export async function consultarRelatorio(id: string, inicio: string, signal: AbortSignal): Promise<Relatorio> {
	const q = new URLSearchParams({ installation_id: id });
	if (inicio) q.set("inicio", inicio);
	const v = await consultarJSON(`/api/web/relatorio-semanal?${q}`, { signal, privada: true });
	if (
		!objeto(v) ||
		v.installation_id !== Number(id) ||
		!Array.isArray(v.repositorios) ||
		!v.repositorios.every((r) => typeof r === "string") ||
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
