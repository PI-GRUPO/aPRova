import { uuidValido } from "./uuid";

export type TipoDeErro = "http" | "rede" | "timeout" | "cancelado" | "resposta" | "requisicao";
export class ErroHTTP extends Error {
	constructor(
		public tipo: TipoDeErro,
		mensagem: string,
		public status?: number,
		public codigo?: string,
	) {
		super(mensagem);
		this.name = "ErroHTTP";
	}
}
function origemPublica(valor: string | undefined) {
	if (!valor) return "";
	const url = new URL(valor);
	if (!["http:", "https:"].includes(url.protocol) || url.username || url.password || url.pathname !== "/" || url.search || url.hash)
		throw new Error("A origem pública deve ser uma origem HTTP sem credenciais ou caminho.");
	return url.origin;
}
const rotas = {
	acoesAnalise: "/api/web/analises/:id/acoes",
	analise: "/api/analises/:id",
	analises: "/api/analises",
	auditoria: "/api/web/auditoria",
	cadeiaRecibos: "/api/web/recibos/cadeia",
	chavesRecibos: "/receipts/keys",
	metricas: "/api/metricas",
	prontidao: "/ready",
	recibo: "/receipts/:id",
	recibos: "/api/web/recibos",
	regras: "/api/web/regras",
	relatorioSemanal: "/api/web/relatorio-semanal",
	repositorios: "/api/repositorios",
	sair: "/api/sessao/sair",
	saude: "/health",
	sessao: "/api/sessao",
	verificarRecibo: "/receipts/:id/verify",
	vinculoSlack: "/api/web/vinculo-slack",
} as const;

export async function consultarJSON(
	rota: keyof typeof rotas,
	opcoes: {
		id?: string;
		parametros?: URLSearchParams;
		signal?: AbortSignal;
		timeout?: number;
		privada?: boolean;
		metodo?: "POST";
		csrf?: string;
		statusEsperado?: number;
		corpo?: unknown;
		lerCodigo?: boolean;
	} = {},
): Promise<unknown> {
	const controlador = new AbortController();
	let iniciouEnvio = false;
	let timer: ReturnType<typeof setTimeout> | undefined;
	const cancelar = () => controlador.abort(new ErroHTTP("cancelado", "Consulta cancelada."));
	try {
		if (!Object.hasOwn(rotas, rota)) throw new ErroHTTP("requisicao", "Rota de API não permitida.");
		let caminho: string = rotas[rota];
		if (caminho.includes(":id")) {
			const id = opcoes.id;
			if (!id || !uuidValido(id)) throw new ErroHTTP("http", "Identificador inválido.", 404);
			caminho = caminho.replace(":id", id);
		}
		const origem = origemPublica(import.meta.env.VITE_API_ORIGEM);
		const destino = new URL(caminho, origem || window.location.origin);
		if (opcoes.parametros) destino.search = opcoes.parametros.toString();
		const corpo = opcoes.corpo === undefined ? undefined : JSON.stringify(opcoes.corpo);
		if (opcoes.signal?.aborted) cancelar();
		controlador.signal.throwIfAborted();
		opcoes.signal?.addEventListener("abort", cancelar, { once: true });
		timer = setTimeout(() => controlador.abort(new ErroHTTP("timeout", "A consulta demorou além do limite.")), opcoes.timeout ?? 10000);
		iniciouEnvio = true;
		const resposta = await fetch(destino, {
			method: opcoes.metodo ?? "GET",
			headers: {
				Accept: "application/json",
				...(opcoes.csrf ? { "X-CSRF-Token": opcoes.csrf } : {}),
				...(opcoes.corpo === undefined ? {} : { "Content-Type": "application/json" }),
			},
			body: corpo,
			credentials: "include",
			cache: "no-store",
			signal: controlador.signal,
			redirect: "error",
		});
		if (!resposta.ok) {
			if (resposta.status === 401 && opcoes.privada && !controlador.signal.aborted) {
				window.dispatchEvent(new Event("aprova:sessao-expirada"));
			}
			let codigo: string | undefined;
			if (opcoes.lerCodigo && resposta.headers.get("content-type")?.includes("application/json")) {
				const erro: { codigo?: unknown } | null = await resposta.json().catch(() => null);
				if (typeof erro?.codigo === "string") codigo = erro.codigo;
				controlador.signal.throwIfAborted();
			}
			throw new ErroHTTP(
				"http",
				resposta.status === 403
					? "Você não tem permissão para esta operação."
					: resposta.status === 503
						? "Serviço temporariamente indisponível."
						: "Não foi possível consultar o serviço.",
				resposta.status,
				codigo,
			);
		}
		if (opcoes.statusEsperado !== undefined && resposta.status !== opcoes.statusEsperado)
			throw new ErroHTTP("resposta", "O serviço não confirmou a operação.");
		if (resposta.status === 204 && opcoes.metodo === "POST") return null;
		if (!resposta.headers.get("content-type")?.includes("application/json")) throw new ErroHTTP("resposta", "O serviço não retornou JSON.");
		try {
			return await resposta.json();
		} catch {
			controlador.signal.throwIfAborted();
			throw new ErroHTTP("resposta", "A resposta do serviço não é válida.");
		}
	} catch (erro) {
		if (erro instanceof ErroHTTP) throw erro;
		controlador.signal.throwIfAborted();
		if (!iniciouEnvio) throw new ErroHTTP("requisicao", "Não foi possível preparar a requisição.");
		throw new ErroHTTP("rede", "Não foi possível conectar ao serviço.");
	} finally {
		clearTimeout(timer);
		opcoes.signal?.removeEventListener("abort", cancelar);
	}
}
