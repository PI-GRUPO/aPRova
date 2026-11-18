export type NivelMapa = "alto" | "medio" | "baixo";
export type ItemMapa = { id: string; repositorio: string; pr: number; avaliadoEm: string };
export type RecorteMapa = { origem: "mock"; data: string; nivel: NivelMapa; itens: ItemMapa[] };
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
export async function consultarRecorteMapa(data: string, nivel: NivelMapa, signal: AbortSignal): Promise<RecorteMapa> {
	signal.throwIfAborted();
	if (!dataMapaValida(data) || !nivelMapaValido(nivel)) throw new Error("Recorte de demonstração inválido.");
	const { referenciasMapaMock, diasMapaMock } = await import("@/mocks/mapa");
	signal.throwIfAborted();
	if (!diasMapaMock.includes(data)) throw new Error("Dia fora da janela de demonstração.");
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
