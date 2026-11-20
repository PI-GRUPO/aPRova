import { type Operacao, operacaoPendente } from "./fonte";

type VinculoSlack = { workspace: string; usuarioSlack: string };
export const consultarVinculo: Operacao<(signal: AbortSignal) => Promise<VinculoSlack | null>> = operacaoPendente;
export const alterarVinculo: Operacao<(acao: "conectar" | "desvincular") => Promise<void>> = operacaoPendente;
