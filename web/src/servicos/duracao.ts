export function formatarEspera(segundos: number) {
	const totalEmMinutos = Math.floor(segundos / 60);
	const dias = Math.floor(totalEmMinutos / (24 * 60));
	const horas = Math.floor(totalEmMinutos / 60) % 24;
	const minutos = totalEmMinutos % 60;
	if (dias > 0) return `${dias}d ${horas}h ${minutos}m`;
	if (horas > 0) return `${horas}h ${minutos}m`;
	return `${minutos}m`;
}
