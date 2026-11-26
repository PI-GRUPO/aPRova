function emNanosegundos(valor: string): bigint | undefined {
	const segundos = valor.slice(0, 19);
	const inicioFuso = valor.length - (valor.endsWith("Z") ? 1 : 6);
	if (inicioFuso < 19 || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$/.test(segundos)) return;
	const fuso = valor.slice(inicioFuso);
	if (fuso !== "Z" && !/^[+-]\d{2}:\d{2}$/.test(fuso)) return;
	const decimal = valor.slice(19, inicioFuso);
	const fracao = decimal.slice(1);
	if (decimal && (!/^[.,]/.test(decimal) || !fracao || /\D/.test(fracao))) return;
	const dataLocal = new Date(`${segundos}Z`);
	const milissegundos = Date.parse(`${segundos}${fuso}`);
	if (!Number.isFinite(milissegundos) || !Number.isFinite(dataLocal.valueOf()) || dataLocal.toISOString().slice(0, 19) !== segundos) return;
	const nanos = fracao.slice(0, 9).padEnd(9, "0");
	return BigInt(milissegundos) * 1_000_000n + BigInt(nanos);
}

export function mesmoInstante(a: string, b: string) {
	const instante = emNanosegundos(a);
	return instante !== undefined && instante === emNanosegundos(b);
}
