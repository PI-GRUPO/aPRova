import type { VisaoGeralDados } from "@/servicos/visao-geral";

// Fixtures visuais do Figma; nunca utilizadas como fallback de erro da API.
const contagensAlto = new Map([
	[5, 1],
	[10, 1],
	[15, 1],
	[18, 1],
	[22, 1],
	[26, 2],
	[28, 3],
	[29, 3],
]);
export const visaoGeralMock: VisaoGeralDados = {
	origem: "mock",
	atualizadoEm: "14:00",
	periodo: "13 out a 11 nov 2026",
	metricas: [
		{ rotulo: "Espera mais antiga", valor: "2d 1h 0m", unidade: "aguardando aprovação", contexto: "aPRova-TESTES #1 · 1 de 5 na fila" },
		{
			rotulo: "Taxa de bloqueio",
			valor: "47%",
			unidade: "das avaliadas",
			contexto: "15 de 32 análises em 30 dias",
			contextoMobile: "15 de 32 em 30 dias",
			proporcao: 15 / 32,
		},
		{
			rotulo: "Falhas no sandbox",
			valor: "31%",
			unidade: "das execuções",
			contexto: "13 de 42 execuções em 30 dias; cada tentativa conta",
			contextoMobile: "13 de 42 em 30 dias",
			proporcao: 13 / 42,
		},
		{
			rotulo: "Análises recebidas",
			valor: "41",
			unidade: "em 30 dias",
			contexto: "32 passaram pelo gate · 9 em andamento ou paradas antes do gate",
			contextoMobile: "32 no gate · 9 antes",
		},
	],
	bloqueadas: [
		{ nome: "Aguardando", quantidade: 5 },
		{ nome: "Aprovadas", quantidade: 7 },
		{ nome: "Alterações pedidas", quantidade: 3 },
	],
	falhas: [
		{ nome: "Infraestrutura", quantidade: 9 },
		{ nome: "Teste", quantidade: 3 },
		{ nome: "Compilação", quantidade: 1 },
	],
	ultimoRecibo: { repositorio: "cobranca-api", pr: 218, hash: "f27414ff01e9ac4c", anterior: "9666d567", data: "11 nov · 13:38" },
	mapa: Array.from({ length: 30 }, (_, i) => ({
		data: new Date(Date.UTC(2026, 9, 13 + i)).toISOString().slice(0, 10),
		alto: contagensAlto.get(i) ?? 0,
		medio: [2, 7, 12, 17, 20, 23, 27].includes(i) ? 1 : 0,
		baixo: [3, 6, 8, 12, 15, 16, 18, 21, 24, 27, 28].includes(i) ? 1 : 0,
	})),
};
