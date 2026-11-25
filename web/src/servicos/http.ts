export type TipoDeErro = "http" | "rede" | "timeout" | "cancelado" | "resposta";
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
const origem = origemPublica(import.meta.env.VITE_API_ORIGEM);
export async function consultarJSON(
	caminho: string,
	opcoes: {
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
	if (!caminho.startsWith("/") || caminho.startsWith("//") || caminho.includes("\\")) throw new Error("Caminho de API inválido.");
	const destino = new URL(caminho, origem || window.location.origin);
	if (destino.origin !== (origem || window.location.origin)) throw new Error("Origem de API inválida.");
	const controlador = new AbortController();
	let expirou = false;
	const cancelar = () => controlador.abort();
	if (opcoes.signal?.aborted) controlador.abort();
	opcoes.signal?.addEventListener("abort", cancelar, { once: true });
	const timer = setTimeout(() => {
		expirou = true;
		controlador.abort();
	}, opcoes.timeout ?? 10000);
	try {
		const resposta = await fetch(destino, {
			method: opcoes.metodo ?? "GET",
			headers: {
				Accept: "application/json",
				...(opcoes.csrf ? { "X-CSRF-Token": opcoes.csrf } : {}),
				...(opcoes.corpo === undefined ? {} : { "Content-Type": "application/json" }),
			},
			body: opcoes.corpo === undefined ? undefined : JSON.stringify(opcoes.corpo),
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
		} catch (erro) {
			if (controlador.signal.aborted) throw erro;
			throw new ErroHTTP("resposta", "A resposta do serviço não é válida.");
		}
	} catch (erro) {
		if (erro instanceof ErroHTTP) throw erro;
		if (controlador.signal.aborted)
			throw new ErroHTTP(expirou ? "timeout" : "cancelado", expirou ? "A consulta demorou além do limite." : "Consulta cancelada.");
		throw new ErroHTTP("rede", "Não foi possível conectar ao serviço.");
	} finally {
		clearTimeout(timer);
		opcoes.signal?.removeEventListener("abort", cancelar);
	}
}
