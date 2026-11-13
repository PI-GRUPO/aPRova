import analisesBrutas from "@/dados/analises.json";
import chavesBrutas from "@/dados/chaves.json";
import instalacaoBruta from "@/dados/instalacao.json";
import recibosBrutos from "@/dados/recibos.json";
import regrasBrutas from "@/dados/regras.json";
import type { Analise, Instalacao, Recibo, Regra } from "./tipos";

export const instalacao = instalacaoBruta as Instalacao;
export const analises = analisesBrutas as Analise[];
export const recibos = recibosBrutos as Recibo[];
export const regras = regrasBrutas as Regra[];
export const chaves = chavesBrutas.chaves;

export const agora = new Date(instalacao.gerado_em);

export function nomeCurtoDoRepositorio(repositorio: string) {
	return repositorio.split("/")[1] ?? repositorio;
}

export function inicioDaEspera(analise: Analise) {
	const etapa = analise.steps.find((passo) => passo.to_state === "aguardando_aprovacao");
	return new Date(etapa ? etapa.created_at : analise.updated_at);
}
