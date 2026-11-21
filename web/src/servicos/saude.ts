import { consultarJSON, ErroHTTP } from "./http";
export interface Saude {
	status: "ok";
}
export interface Prontidao {
	status: "ok" | "indisponivel";
	dependencias: Record<string, "ok" | "indisponivel">;
}
export async function consultarSaude(signal?: AbortSignal): Promise<Saude> {
	const dado = await consultarJSON("/health", { signal });
	if (!dado || typeof dado !== "object" || !("status" in dado) || dado.status !== "ok")
		throw new ErroHTTP("resposta", "Resposta de saúde inválida.");
	return { status: "ok" };
}
export async function consultarProntidao(signal?: AbortSignal): Promise<Prontidao> {
	const dado = await consultarJSON("/ready", { signal });
	if (
		!dado ||
		typeof dado !== "object" ||
		!("status" in dado) ||
		!["ok", "indisponivel"].includes(String(dado.status)) ||
		!("dependencias" in dado) ||
		!dado.dependencias ||
		typeof dado.dependencias !== "object" ||
		Array.isArray(dado.dependencias) ||
		!Object.values(dado.dependencias).every((v) => v === "ok" || v === "indisponivel")
	)
		throw new ErroHTTP("resposta", "Resposta de prontidão inválida.");
	return dado as Prontidao;
}
