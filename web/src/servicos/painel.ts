import type { Escopo } from "./fonte";
import { consultarJSON, ErroHTTP } from "./http";
import { objeto } from "./recibos";

export function invalido(): never {
	throw new ErroHTTP("resposta", "Resposta do painel fora do contrato esperado.");
}
export function registro(v: unknown) {
	if (!objeto(v)) return invalido();
	return v;
}
export function texto(v: unknown) {
	if (typeof v !== "string") return invalido();
	return v;
}
export function inteiro(v: unknown) {
	if (typeof v !== "number" || !Number.isSafeInteger(v) || v < 0) return invalido();
	return v;
}
export function data(v: unknown) {
	const s = texto(v);
	if (!Number.isFinite(Date.parse(s))) return invalido();
	return s;
}
export function lista<T>(v: unknown, validar: (item: Record<string, unknown>) => T) {
	if (!Array.isArray(v)) return invalido();
	return v.map((item) => validar(registro(item)));
}
export async function lerPainel(caminho: string, escopo: Escopo, filtros: URLSearchParams, signal: AbortSignal) {
	const q = new URLSearchParams(filtros);
	q.set("installation_id", escopo.instalacao);
	if (escopo.repositorio) q.set("repositorio", escopo.repositorio);
	const v = registro(await consultarJSON(`${caminho}?${q}`, { privada: true, signal }));
	const snapshot = data(v.snapshot);
	if (
		v.installation_id !== Number(escopo.instalacao) ||
		v.repositorio !== escopo.repositorio ||
		v.pagina !== Number(q.get("pagina") ?? 1) ||
		v.limite !== Number(q.get("limite") ?? 20) ||
		(q.has("snapshot") && snapshot !== q.get("snapshot"))
	)
		return invalido();
	const inicio = data(v.inicio),
		fim = data(v.fim);
	const dias = Number(q.get("periodo") ?? 30);
	if (
		Date.parse(fim) - Date.parse(inicio) !== dias * 86400000 ||
		Date.parse(fim) % 86400000 !== 0 ||
		Date.parse(snapshot) >= Date.parse(fim) ||
		Date.parse(snapshot) < Date.parse(fim) - 86400000
	)
		return invalido();
	return { dados: registro(v.dados), inicio, fim, snapshot, pagina: inteiro(v.pagina), limite: inteiro(v.limite) };
}
export function filtrosPeriodo(f: URLSearchParams, paginado = false) {
	const q = new URLSearchParams({ periodo: f.get("periodo") ?? "30" });
	if (paginado)
		for (const k of ["pagina", "snapshot"]) {
			const v = f.get(k);
			if (v) q.set(k, v);
		}
	return q;
}
