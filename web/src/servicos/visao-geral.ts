export type ItemDaFila = {
	id: string;
	repositorio: string;
	pr: number;
	branch: string;
	espera: string;
	motivo: string;
	evidencia: string;
};
export type VisaoGeralDados = {
	origem: "mock";
	atualizadoEm: string;
	periodo: string;
	fila: ItemDaFila[];
	metricas: { rotulo: string; valor: string; unidade: string; contexto: string; contextoMobile?: string; proporcao?: number }[];
	bloqueadas: { nome: string; quantidade: number }[];
	falhas: { nome: string; quantidade: number }[];
	ultimoRecibo: { repositorio: string; pr: number; hash: string; anterior: string; data: string };
	mapa: { data: string; alto: number; medio: number; baixo: number }[];
};

// A API de métricas de 30 dias ainda não existe. O agregado semanal não
// corresponde a este contrato: manter a simulação explícita e substituível.
export async function consultarVisaoGeral(signal: AbortSignal): Promise<VisaoGeralDados> {
	const { visaoGeralMock } = await import("@/mocks/visao-geral");
	signal.throwIfAborted();
	return structuredClone(visaoGeralMock);
}
