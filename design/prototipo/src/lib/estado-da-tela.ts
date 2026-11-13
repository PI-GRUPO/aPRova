import { useSearchParams } from "react-router";

export type EstadoDaTela = "dados" | "vazio" | "carregando" | "erro";

const estadosSimulados: EstadoDaTela[] = ["vazio", "carregando", "erro"];

export function useEstadoDaTela(): EstadoDaTela {
	const [parametros] = useSearchParams();
	const pedido = parametros.get("estado") as EstadoDaTela | null;
	return pedido && estadosSimulados.includes(pedido) ? pedido : "dados";
}
