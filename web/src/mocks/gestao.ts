// Referências de demonstração extraídas do Figma. Não representam a instalação real.
export const repositoriosMock = [
	{
		id: "aprova",
		nome: "aPRova-TESTES",
		completo: "equipe/aPRova-TESTES",
		check: "Configurado",
		ultima: "há 8 min",
		permissoes: "Completas",
	},
	{
		id: "cobranca",
		nome: "cobranca-api",
		completo: "equipe/cobranca-api",
		check: "Não configurado",
		ultima: "há 2 h",
		permissoes: "Completas",
	},
	{
		id: "painel",
		nome: "painel-entregas",
		completo: "equipe/painel-entregas",
		check: "Não confirmado",
		ultima: "há 2 d",
		permissoes: "Falta permissão de checks",
	},
];
export const referenciaMock = new Date("2026-11-11T14:00:00Z");
export const regrasMock = [
	{ id: "falha_sandbox", descricao: "teste ou compilação falhou", nivel: "alto", acionamentos: 5, aprovadas: 2 },
	{ id: "muitos_arquivos", descricao: "mais de 20 arquivos", nivel: "alto", acionamentos: 3, aprovadas: 1 },
	{ id: "configuracao_ci", descricao: "build ou CI alterado", nivel: "alto", acionamentos: 1, aprovadas: 0 },
	{ id: "dependencia_nova", descricao: "nova dependência", nivel: "alto", acionamentos: 4, aprovadas: 2 },
	{ id: "segredo_versionado", descricao: "arquivo sensível", nivel: "alto", acionamentos: 1, aprovadas: 0 },
	{ id: "area_sensivel", descricao: "caminho sensível", nivel: "alto", acionamentos: 3, aprovadas: 2 },
	{ id: "codigo_sem_teste", descricao: "código sem teste alterado", nivel: "medio", acionamentos: 6, aprovadas: 0 },
];
// Ocorrências para que filtros alterem contagens, em vez de trocar só o rótulo.
export const ocorrenciasRegrasMock = regrasMock.flatMap((r, indice) =>
	Array.from({ length: r.acionamentos }, (_, i) => ({
		id: `${r.id}-${i}`,
		regra: r.id,
		repo: repositoriosMock[(i + indice) % 3].nome,
		data: new Date(Date.UTC(2026, 10, 11 - i * 5)).toISOString(),
		aprovada: i < r.aprovadas,
	})),
);
export const calibracoesMock = [
	{ id: "c1", nome: "painel-entregas #88", repo: "painel-entregas", risco: "Alto → Médio", resultado: "Reduzido · histórico favorável" },
	{ id: "c2", nome: "painel-entregas #97", repo: "painel-entregas", risco: "Alto → Alto", resultado: "Mantido · histórico insuficiente" },
	{ id: "c3", nome: "aPRova-TESTES #1", repo: "aPRova-TESTES", risco: "Alto → Alto", resultado: "Redução bloqueada · falha de teste" },
];
export const auditoriaMock = [
	{
		id: "a1",
		data: "2026-11-11T13:10:00Z",
		pessoa: "@autor",
		acao: "Aprovar",
		pr: 1,
		repo: "aPRova-TESTES",
		origem: "Painel",
		decisao: "Recusada",
		motivo: "Autor do próprio PR",
	},
	{
		id: "a2",
		data: "2026-11-11T12:50:00Z",
		pessoa: "@revisor",
		acao: "Pedir alteração",
		pr: 214,
		repo: "cobranca-api",
		origem: "Slack",
		decisao: "Aceita",
		motivo: "Permissão confirmada",
	},
	{
		id: "a3",
		data: "2026-11-11T12:42:00Z",
		pessoa: "@convidado",
		acao: "Aprovar",
		pr: 97,
		repo: "painel-entregas",
		origem: "Slack",
		decisao: "Recusada",
		motivo: "Permissão insuficiente",
	},
	{
		id: "a4",
		data: "2026-11-11T12:40:00Z",
		pessoa: "@revisor",
		acao: "Reexecutar",
		pr: 99,
		repo: "painel-entregas",
		origem: "Painel",
		decisao: "Falhou",
		motivo: "Execução indisponível",
	},
];
export const configuracaoInicialMock = {
	limite: 20,
	caminhos: ["auth/", "security/", "payments/", "migrations/"],
	versao: 3,
	historico: [
		{ versao: 3, pessoa: "@admin", data: "10 nov · 11:00" },
		{ versao: 2, pessoa: "@admin", data: "08 nov · 09:30" },
		{ versao: 1, pessoa: "Inicial", data: "01 nov · 08:00" },
	],
};
export const sugestoesMock = [
	{
		id: "arquivos-gerados",
		repo: "cobranca-api",
		titulo: "Revisar limite para arquivos gerados",
		origem: "Histórico e decisões humanas",
		estado: "pendente" as const,
		resumo: "Critério específico para arquivos gerados.",
		evidencia:
			"24 revisões semelhantes · 22 aprovações · 2 ajustes solicitados. As justificativas apontam arquivos gerados como motivo recorrente.",
		limiteProposto: 30,
	},
	{
		id: "agrupamento",
		repo: "cobranca-api",
		titulo: "Ajustar agrupamento de área sensível",
		origem: "Ocorrências e justificativas",
		estado: "pendente" as const,
		resumo: "Revisar o agrupamento de caminhos sensíveis.",
		evidencia:
			"Ocorrências e justificativas recorrentes indicam caminhos relacionados. A revisão humana define quais caminhos continuam protegidos.",
		limiteProposto: null,
	},
];
export const slaInicialMock = { primeiro: 2, segundo: 8 };
