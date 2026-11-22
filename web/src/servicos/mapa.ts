import { listarAnalises } from "./analises";
export type NivelMapa = "alto" | "medio" | "baixo";
export type ItemMapa = { id: string; repositorio: string; pr: number; avaliadoEm: string };
export type RecorteMapa = {
	origem: "mock" | "api";
	data: string;
	nivel: NivelMapa;
	itens: ItemMapa[];
	pagina?: number;
	temMais?: boolean;
	snapshot?: string;
};
export function nivelMapaValido(valor: string | null): valor is NivelMapa {
	return valor === "alto" || valor === "medio" || valor === "baixo";
}
export function dataMapaValida(valor: string | null): valor is string {
	if (!valor || !/^\d{4}-\d{2}-\d{2}$/.test(valor)) return false;
	const data = new Date(`${valor}T00:00:00Z`);
	return !Number.isNaN(data.valueOf()) && data.toISOString().slice(0, 10) === valor;
}
export function rotuloDiaMapa(valor: string) {
	const partes = new Intl.DateTimeFormat("pt-BR", { day: "numeric", month: "short", timeZone: "UTC" }).formatToParts(
		new Date(`${valor}T12:00:00Z`),
	);
	return `${partes.find((p) => p.type === "day")?.value} ${partes.find((p) => p.type === "month")?.value.replace(/\.$/, "")}`;
}
export async function consultarRecorteMapa(
	instalacao: string,
	data: string,
	nivel: NivelMapa,
	signal: AbortSignal,
	repositorio = "",
	pagina = 1,
	snapshot = "",
): Promise<RecorteMapa> {
	signal.throwIfAborted();
	if (!dataMapaValida(data) || !nivelMapaValido(nivel)) throw new Error("Recorte inválido.");
	if (!__MOCKS__) {
		const fim = new Date(Date.parse(data) + 86400000).toISOString().slice(0, 10);
		const q = new URLSearchParams({ installation_id: instalacao, inicio: data, fim, nivel, pagina: String(pagina) });
		if (repositorio) q.set("repositorio", repositorio);
		if (snapshot) q.set("snapshot", snapshot);
		const v = await listarAnalises(q, signal);
		if (
			(snapshot && snapshot !== v.snapshot) ||
			v.pagina !== pagina ||
			v.itens.some((i) => i.nivel !== nivel || new Date(i.recebido_em).toISOString().slice(0, 10) !== data)
		)
			throw new Error("Recorte inválido.");
		return {
			origem: "api",
			data,
			nivel,
			pagina: v.pagina,
			temMais: v.tem_mais,
			snapshot: v.snapshot,
			itens: v.itens.map((i) => ({ id: i.id, repositorio: i.repositorio, pr: i.numero_pr, avaliadoEm: i.recebido_em })),
		};
	}
	const { referenciasMapaMock, diasMapaMock } = await import("@/mocks/mapa");
	signal.throwIfAborted();
	if (!diasMapaMock.includes(data)) throw new Error("Dia fora da janela.");
	const itens = referenciasMapaMock
		.filter((r) => r.data === data && r.nivel === nivel)
		.map((r) => ({
			id: r.detalhe.analise.id,
			repositorio: r.detalhe.analise.repositorio,
			pr: r.detalhe.analise.numero_pr,
			avaliadoEm: r.avaliadoEm,
		}));
	return structuredClone({ origem: "mock", data, nivel, itens });
}
