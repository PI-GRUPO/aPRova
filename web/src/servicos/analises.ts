import type { Consulta, Escopo } from "./fonte";
import { consultarJSON, ErroHTTP } from "./http";
import { mesmoInstante } from "./instante";
import { objeto, uuidValido } from "./recibos";
export const estados = new Map<string, string>([
	["pendente", "Pendente"],
	["mapeando_contexto", "Mapeando contexto"],
	["gerando", "Gerando"],
	["testando_sandbox", "Testando sandbox"],
	["avaliando_risco", "Avaliando risco"],
	["aguardando_aprovacao", "Aguardando aprovação"],
	["aplicando", "Aplicando"],
	["concluido", "Concluído"],
	["rejeitado", "Rejeitado"],
	["falhou", "Falhou"],
]);
export const niveis = new Map<string, string>([
	["baixo", "risco baixo"],
	["medio", "risco médio"],
	["alto", "risco alto"],
	["nao_avaliado", "Não avaliado"],
]);
export type Analise = {
	espera_segundos?: number;
	id: string;
	repositorio: string;
	numero_pr: number;
	sha_head: string;
	autor: string;
	estado: string;
	nivel: string | null;
	recebido_em: string;
	criado_em: string;
	atualizado_em: string;
};
export type PaginaAnalises = {
	installation_id: number;
	snapshot: string;
	pagina: number;
	limite: number;
	total: number;
	tem_mais: boolean;
	itens: Analise[];
};
export type Motivo = { id: string; nivel: string; descricao: string; evidencias: string[] };
export type Vizinho = { task_run_id: string; repositorio: string; numero_pr: number; similaridade: number; desfecho: string };
export type Gate = {
	nivel_deterministico: string;
	nivel: string;
	motivo: string;
	motivos: Motivo[];
	calibracao_estado: string;
	calibracao_razao: string;
	vizinhos: Vizinho[];
};
export type Etapa = { id: number; de: string; para: string; detalhe: string; criado_em: string };
export type Tentativa = {
	tentativa: number;
	compilou: boolean;
	testes_passaram: number;
	testes_falharam: number;
	saida: string;
	saida_truncada: boolean;
	duracao_ms: number;
	tipo_falha: string;
	criado_em: string;
};
export type Aprovacao = {
	id: number;
	decisao: string;
	revisor: string;
	origem: string;
	justificativa: string | null;
	commit_sha: string | null;
	aplicavel_ao_commit: boolean;
	decidido_em: string;
};
export type Detalhe = {
	installation_id: number;
	analise: Analise;
	branch: string | null;
	url_pr: string;
	url_check: string | null;
	atual: boolean;
	gate: Gate | null;
	etapas: Etapa[];
	tentativas: Tentativa[];
	aprovacoes: Aprovacao[];
	recibo_id: string | null;
	testes_estruturados_disponiveis: boolean;
	rede_isolada: boolean | null;
};
const data = (v: unknown) => typeof v === "string" && !Number.isNaN(Date.parse(v));
const numero = (v: unknown): v is number => typeof v === "number" && Number.isSafeInteger(v) && v >= 0;
const textos = (v: unknown) => Array.isArray(v) && v.every((x) => typeof x === "string");
const anulavel = (v: unknown) => v === null || typeof v === "string";
function invalida(): never {
	throw new ErroHTTP("resposta", "A resposta de análises não corresponde ao contrato esperado.");
}
function analise(v: unknown): v is Analise {
	return (
		objeto(v) &&
		["id", "repositorio", "sha_head", "autor"].every((k) => typeof v[k] === "string") &&
		numero(v.numero_pr) &&
		(v.espera_segundos === undefined || numero(v.espera_segundos)) &&
		typeof v.estado === "string" &&
		estados.has(v.estado) &&
		(v.nivel === null || (typeof v.nivel === "string" && ["baixo", "medio", "alto"].includes(v.nivel))) &&
		["recebido_em", "criado_em", "atualizado_em"].every((k) => data(v[k]))
	);
}
export async function listarAnalises(q: URLSearchParams, signal: AbortSignal): Promise<PaginaAnalises> {
	const v = await consultarJSON(`/api/analises?${q}`, { signal, privada: true });
	if (
		!objeto(v) ||
		v.installation_id !== Number(q.get("installation_id")) ||
		!data(v.snapshot) ||
		!["pagina", "limite", "total"].every((k) => numero(v[k])) ||
		typeof v.tem_mais !== "boolean" ||
		!Array.isArray(v.itens) ||
		!v.itens.every(analise) ||
		v.pagina !== Number(q.get("pagina") ?? 1) ||
		!numero(v.limite) ||
		v.limite < 1 ||
		v.limite > 50 ||
		v.itens.length > v.limite ||
		(q.has("repositorio") && v.itens.some((i) => i.repositorio !== q.get("repositorio")))
	)
		return invalida();
	return v as PaginaAnalises;
}
function gateValido(v: unknown) {
	return (
		v === null ||
		(objeto(v) &&
			["nivel_deterministico", "nivel", "motivo", "calibracao_estado", "calibracao_razao"].every((k) => typeof v[k] === "string") &&
			Array.isArray(v.motivos) &&
			v.motivos.every((m) => objeto(m) && ["id", "nivel", "descricao"].every((k) => typeof m[k] === "string") && textos(m.evidencias)) &&
			Array.isArray(v.vizinhos) &&
			v.vizinhos.every(
				(m) =>
					objeto(m) &&
					["task_run_id", "repositorio", "desfecho"].every((k) => typeof m[k] === "string") &&
					numero(m.numero_pr) &&
					typeof m.similaridade === "number",
			))
	);
}
function agregadosValidos(v: Record<string, unknown>) {
	return (
		Array.isArray(v.etapas) &&
		v.etapas.every(
			(e) => objeto(e) && numero(e.id) && ["de", "para", "detalhe"].every((k) => typeof e[k] === "string") && data(e.criado_em),
		) &&
		Array.isArray(v.tentativas) &&
		v.tentativas.every(
			(e) =>
				objeto(e) &&
				["tentativa", "testes_passaram", "testes_falharam", "duracao_ms"].every((k) => numero(e[k])) &&
				typeof e.compilou === "boolean" &&
				typeof e.saida_truncada === "boolean" &&
				typeof e.saida === "string" &&
				typeof e.tipo_falha === "string" &&
				data(e.criado_em),
		) &&
		Array.isArray(v.aprovacoes) &&
		v.aprovacoes.every(
			(e) =>
				objeto(e) &&
				numero(e.id) &&
				["decisao", "revisor", "origem"].every((k) => typeof e[k] === "string") &&
				anulavel(e.justificativa) &&
				anulavel(e.commit_sha) &&
				typeof e.aplicavel_ao_commit === "boolean" &&
				data(e.decidido_em),
		)
	);
}
export function urlGitHub(v: string) {
	try {
		const u = new URL(v);
		return u.origin === "https://github.com" && !u.username && !u.password ? u.href : undefined;
	} catch {
		return undefined;
	}
}
export async function detalharAnalise(id: string, instalacao: string, signal: AbortSignal, repositorio = ""): Promise<Detalhe> {
	const q = new URLSearchParams({ installation_id: instalacao });
	if (repositorio) q.set("repositorio", repositorio);
	const v = await consultarJSON(`/api/analises/${encodeURIComponent(id)}?${q}`, { signal, privada: true });
	if (
		!objeto(v) ||
		v.installation_id !== Number(instalacao) ||
		!analise(v.analise) ||
		v.analise.id !== id ||
		(!!repositorio && v.analise.repositorio !== repositorio) ||
		!anulavel(v.branch) ||
		typeof v.url_pr !== "string" ||
		!urlGitHub(v.url_pr) ||
		!anulavel(v.url_check) ||
		(typeof v.url_check === "string" && !urlGitHub(v.url_check)) ||
		typeof v.atual !== "boolean" ||
		!gateValido(v.gate) ||
		!agregadosValidos(v) ||
		!anulavel(v.recibo_id) ||
		typeof v.testes_estruturados_disponiveis !== "boolean" ||
		!(v.rede_isolada === null || typeof v.rede_isolada === "boolean")
	)
		return invalida();
	return v as Detalhe;
}
export async function listarRepositorios(id: string, signal: AbortSignal): Promise<string[]> {
	const v = await consultarJSON(`/api/repositorios?installation_id=${encodeURIComponent(id)}`, { signal, privada: true });
	if (!objeto(v) || v.installation_id !== Number(id) || !textos(v.repositorios)) return invalida();
	return v.repositorios as string[];
}
export function erroDeAnalise(e: unknown) {
	if (e instanceof ErroHTTP) {
		if (e.status === 404) return "Análise não encontrada neste escopo.";
		if (e.status === 403) return "Você não tem acesso a esta instalação ou repositório.";
		if (e.status === 400) return "Os filtros ou o identificador não são válidos.";
		return e.message;
	}
	return "Não foi possível consultar as análises.";
}

export const consultarRepos: Consulta<string[]> = ({ instalacao }, _, signal) => listarRepositorios(instalacao, signal);
export const consultarFila: Consulta<PaginaAnalises> = async ({ instalacao }, filtros, signal) => {
	const q = new URLSearchParams(filtros);
	q.set("installation_id", instalacao);
	q.set("estado", "aguardando_aprovacao");
	q.set("nivel", "alto");
	q.set("inicio", "1970-01-01");
	q.set("ordem", "antigas");
	const dados = await listarAnalises(q, signal);
	const corteSolicitado = q.get("snapshot");
	if (
		dados.itens.some((i) => !uuidValido(i.id) || i.estado !== "aguardando_aprovacao" || i.nivel !== "alto") ||
		(corteSolicitado !== null && !mesmoInstante(dados.snapshot, corteSolicitado))
	)
		throw new ErroHTTP("resposta", "A resposta não corresponde aos filtros da fila.");
	return dados;
};

export async function saudeRepositorios({ instalacao }: Escopo, filtros: URLSearchParams, signal: AbortSignal) {
	const pagina = Number(filtros.get("pagina") ?? 1);
	const q = new URLSearchParams({ installation_id: instalacao, detalhes: "1", pagina: String(pagina) });
	const v = await consultarJSON(`/api/repositorios?${q}`, { privada: true, signal });
	if (
		!objeto(v) ||
		v.installation_id !== Number(instalacao) ||
		v.pagina !== pagina ||
		v.limite !== 10 ||
		!numero(v.total) ||
		!objeto(v.permissoes) ||
		!Array.isArray(v.itens) ||
		v.itens.length !== Math.max(0, Math.min(10, v.total - (pagina - 1) * 10))
	)
		return invalida();
	const permissoesConhecidas = [
		["checks", v.permissoes.checks],
		["contents", v.permissoes.contents],
		["pull_requests", v.permissoes.pull_requests],
	] as const;
	const acessos = permissoesConhecidas.flatMap(([chave, nivel]) => {
		if (nivel === undefined) return [];
		if (typeof nivel !== "string" || !["read", "write", "admin", "none"].includes(nivel)) return invalida();
		return [`${chave}: ${nivel}`];
	});
	const itens = v.itens.map((r) => {
		if (
			!objeto(r) ||
			typeof r.nome !== "string" ||
			typeof r.branch !== "string" ||
			!(r.ultima_analise === null || (typeof r.ultima_analise === "string" && data(r.ultima_analise))) ||
			!(r.check_obrigatorio === null || typeof r.check_obrigatorio === "boolean")
		)
			return invalida();
		return { nome: r.nome, branch: r.branch, ultima: r.ultima_analise, check: r.check_obrigatorio };
	});
	return { itens, permissoes: acessos.join(" · "), total: v.total };
}
