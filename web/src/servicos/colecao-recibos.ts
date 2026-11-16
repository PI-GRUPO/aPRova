import { referenciaMock } from "@/mocks/gestao";
import { cadeiaDemonstracao, recibosDemonstracao } from "@/mocks/recibos";
export function listarRecibosSimulados(repo: string, dias: number) {
	const inicio = referenciaMock.getTime() - dias * 86400000;
	return recibosDemonstracao
		.filter((r) => (!repo || r.repo === repo) && Date.parse(r.emissao) > inicio)
		.map((r) => ({ ...r, origem: "mock" as const }));
}
export async function verificarCadeiaSimulada() {
	return { ...cadeiaDemonstracao, origem: "mock" as const };
}
export function exportarColecaoSimulada(repo: string, dias: number, formato: "json" | "csv") {
	const registros = listarRecibosSimulados(repo, dias);
	if (formato === "json")
		return JSON.stringify({ origem: "mock", aviso: "Demonstração sem assinaturas ou valor de prova", registros }, null, 2);
	const celula = (v: unknown) => `"${String(v).replaceAll('"', '""')}"`;
	return [
		["origem", "recibo", "repositorio", "pr", "commit", "emissao"],
		...registros.map((r) => ["mock", r.id, r.repo, r.pr, r.sha, r.emissao]),
	]
		.map((l) => l.map(celula).join(","))
		.join("\r\n");
}
