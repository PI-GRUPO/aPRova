function emNanosegundos(valor: string): bigint | undefined {
	const partes = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:[.,](\d+))?(Z|[+-]\d{2}:\d{2})$/.exec(valor);
	if (!partes) return;
	const [, segundos, fracao = "", fuso] = partes;
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
