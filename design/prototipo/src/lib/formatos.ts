import { format, subDays } from "date-fns";
import { ptBR } from "date-fns/locale";
import type { Nivel } from "./tipos";

const minuto = 60_000;
const hora = 60 * minuto;
const dia = 24 * hora;

export const nomeDoNivel: Record<Nivel, string> = {
	baixo: "baixo",
	medio: "médio",
	alto: "alto",
};

function partesDaEspera(desde: Date, ate: Date) {
	const decorrido = Math.max(0, ate.getTime() - desde.getTime());
	return {
		dias: Math.floor(decorrido / dia),
		horas: Math.floor((decorrido % dia) / hora),
		minutos: Math.floor((decorrido % hora) / minuto),
	};
}

export function espera(desde: Date, ate: Date) {
	const { dias, horas, minutos } = partesDaEspera(desde, ate);

	if (dias > 0) return `${dias} d ${horas} h`;
	if (horas > 0) return `${horas} h ${minutos} min`;
	return `${Math.max(1, minutos)} min`;
}

export function esperaPorExtenso(desde: Date, ate: Date) {
	const { dias, horas, minutos } = partesDaEspera(desde, ate);

	if (dias > 0) return `${plural(dias, "dia", "dias")} e ${plural(horas, "hora", "horas")}`;
	if (horas > 0) return `${plural(horas, "hora", "horas")} e ${plural(minutos, "minuto", "minutos")}`;
	return plural(Math.max(1, minutos), "minuto", "minutos");
}

export function diaEMes(data: Date) {
	return format(data, "d MMM", { locale: ptBR }).replace(".", "");
}

export function dataEHora(data: Date) {
	return format(data, "d MMM yyyy, HH:mm", { locale: ptBR }).replace(".", "");
}

export function periodo(ate: Date, dias: number) {
	return `${diaEMes(subDays(ate, dias - 1))} a ${format(ate, "d MMM yyyy", { locale: ptBR }).replace(".", "")}`;
}

export function porcentagem(parte: number, todo: number) {
	if (todo === 0) return "0%";
	return `${Math.round((parte / todo) * 100)}%`;
}

export function plural(quantidade: number, singular: string, variasFormas: string) {
	return `${quantidade} ${quantidade === 1 ? singular : variasFormas}`;
}
