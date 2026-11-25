import { type Operacao, operacaoPendente } from "./fonte";
import { consultarJSON } from "./http";
import { invalido, registro } from "./painel";
export async function consultarVinculo(usuario: number, signal: AbortSignal) {
	const v = registro(await consultarJSON("vinculoSlack", { privada: true, signal }));
	if (v.usuario_id !== usuario || typeof v.vinculado !== "boolean") return invalido();
	return v.vinculado;
}
export const alterarVinculo: Operacao<(acao: "conectar" | "desvincular") => Promise<void>> = operacaoPendente;
