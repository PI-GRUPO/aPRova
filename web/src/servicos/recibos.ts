import { consultarJSON, ErroHTTP } from "./http";

export type Recibo = {
	id: string;
	task_run_id: string;
	key_id: string;
	hash_sha256: string;
	payload_base64: string;
	assinatura_base64: string;
	recibo_anterior_id: string | null;
	payload: Record<string, unknown> | null;
};
export type Chave = { key_id: string; algoritmo: string; chave_publica_base64: string };
export const etapas = ["hash", "chave", "assinatura", "conteudo", "elo"] as const;
export type Verificacao = {
	id: string;
	valido: boolean;
	falha?: string;
	detalhe?: string;
	hash_sha256: string;
	key_id: string;
	verificacoes: { nome: string; resultado: "ok" | "falhou" | "nao_verificada" }[];
};
export function objeto(v: unknown): v is Record<string, unknown> {
	return v !== null && typeof v === "object" && !Array.isArray(v);
}
function invalida(): never {
	throw new ErroHTTP("resposta", "A resposta de recibos não corresponde ao formato esperado.");
}
export const uuidValido = (v: string) => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(v);
export async function buscarRecibo(id: string, signal: AbortSignal): Promise<Recibo> {
	const v = await consultarJSON(`/receipts/${encodeURIComponent(id)}`, { signal });
	if (
		!objeto(v) ||
		!["id", "task_run_id", "key_id", "hash_sha256", "payload_base64", "assinatura_base64"].every((k) => typeof v[k] === "string") ||
		v.id !== id ||
		!(v.recibo_anterior_id === null || typeof v.recibo_anterior_id === "string") ||
		!(v.payload === null || objeto(v.payload))
	)
		return invalida();
	return v as Recibo;
}
export async function buscarChaves(signal: AbortSignal): Promise<Chave[]> {
	const v = await consultarJSON("/receipts/keys", { signal });
	if (
		!objeto(v) ||
		!Array.isArray(v.chaves) ||
		!v.chaves.every((c) => objeto(c) && ["key_id", "algoritmo", "chave_publica_base64"].every((k) => typeof c[k] === "string"))
	)
		return invalida();
	return v.chaves as Chave[];
}
export async function verificarRecibo(recibo: Recibo, signal: AbortSignal): Promise<Verificacao> {
	const v = await consultarJSON(`/receipts/${encodeURIComponent(recibo.id)}/verify`, { signal });
	if (
		!objeto(v) ||
		v.id !== recibo.id ||
		v.key_id !== recibo.key_id ||
		typeof v.hash_sha256 !== "string" ||
		typeof v.valido !== "boolean" ||
		!Array.isArray(v.verificacoes) ||
		v.verificacoes.length !== 5 ||
		!v.verificacoes.every(
			(c, i) =>
				objeto(c) && c.nome === etapas[i] && typeof c.resultado === "string" && ["ok", "falhou", "nao_verificada"].includes(c.resultado),
		) ||
		(v.falha !== undefined && typeof v.falha !== "string") ||
		(v.detalhe !== undefined && typeof v.detalhe !== "string")
	)
		return invalida();
	if (v.valido && (v.hash_sha256 !== recibo.hash_sha256 || v.verificacoes.some((c) => c.resultado !== "ok"))) return invalida();
	if (!v.valido && !v.verificacoes.some((c) => c.resultado === "falhou")) return invalida();
	return v as Verificacao;
}
export function baixar(nome: string, conteudo: string | Blob, tipo = "application/json") {
	const url = URL.createObjectURL(conteudo instanceof Blob ? conteudo : new Blob([conteudo], { type: tipo }));
	const a = document.createElement("a");
	a.href = url;
	a.download = nome;
	document.body.append(a);
	a.click();
	a.remove();
	setTimeout(() => URL.revokeObjectURL(url), 1000);
}
export function mensagemDeErro(erro: unknown) {
	if (erro instanceof ErroHTTP) return erro.status === 404 ? "Recibo não encontrado. Confira o UUID informado." : erro.message;
	return "Não foi possível consultar o recibo.";
}
