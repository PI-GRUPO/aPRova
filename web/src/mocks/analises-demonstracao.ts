import type { Detalhe } from "@/servicos/analises";
import type { ComplementoDemonstracao } from "@/servicos/analises-demonstracao";

const referencias = [
	{
		id: "00000000-0000-4000-8000-000000000001",
		repositorio: "aPRova-TESTES",
		pr: 1,
		branch: "teste-peso-zero",
		espera: "2 d 1 h",
		motivo: "falha_sandbox",
		evidencia: "1 de 11 testes falharam",
	},
	{
		id: "00000000-0000-4000-8000-000000000002",
		repositorio: "cobranca-api",
		pr: 214,
		branch: "agente/juros-compostos",
		espera: "1 d 1 h",
		motivo: "dependencia_nova",
		evidencia: "go.mod",
	},
	{
		id: "00000000-0000-4000-8000-000000000003",
		repositorio: "painel-entregas",
		pr: 97,
		branch: "agente/exportar-planilha",
		espera: "7 h 19 min",
		motivo: "dependencia_nova +1",
		evidencia: "requirements.txt · planilha.py",
	},
	{
		id: "00000000-0000-4000-8000-000000000004",
		repositorio: "cobranca-api",
		pr: 217,
		branch: "agente/parcelamento",
		espera: "3 h 3 min",
		motivo: "area_sensivel +1",
		evidencia: "000031_parcelas.down.sql +2",
	},
	{
		id: "00000000-0000-4000-8000-000000000005",
		repositorio: "painel-entregas",
		pr: 99,
		branch: "agente/reorganiza-rotas",
		espera: "47 min",
		motivo: "muitos_arquivos +1",
		evidencia: "painel/rotas/rota_01.py +32",
	},
];

// Dados públicos do protótipo, sem relação com uma instalação autenticada.
const commits = ["6be930afc83bb7213d3d4e111f2c6eb7fd008a42", "958be56", "30f2971", "ab21f04", "11645da"];
export const analisesDemonstracao: Detalhe[] = referencias.map((item, indice) => {
	const commit = commits.at(indice);
	if (!commit) throw new Error("Commit ausente para a análise de demonstração.");
	const primeiro = indice === 0;
	const recebido = "2026-11-09T11:00:00Z";
	return {
		installation_id: 0, // Sem tenant real: o adapter fornece o escopo explícito de demonstração.
		analise: {
			id: item.id,
			repositorio: item.repositorio,
			numero_pr: item.pr,
			sha_head: commit.padEnd(40, "0"),
			autor: "agente-frete[bot]",
			estado: "aguardando_aprovacao",
			nivel: "alto",
			recebido_em: recebido,
			criado_em: recebido,
			atualizado_em: "2026-11-09T11:00:47Z",
		},
		branch: item.branch,
		url_pr: "",
		url_check: null,
		atual: true,
		gate: {
			nivel_deterministico: "alto",
			nivel: "alto",
			motivo: primeiro ? "Os testes ou a compilação falharam." : `Referência visual: ${item.motivo}.`,
			motivos: [
				{
					id: item.motivo.split(" ")[0],
					nivel: "alto",
					descricao: primeiro ? "Os testes ou a compilação falharam." : "Motivo apresentado na fila.",
					evidencias: [item.evidencia],
				},
			],
			calibracao_estado: primeiro ? "motivo_nao_calibravel" : "nao_informada",
			calibracao_razao: primeiro ? "Redução bloqueada por falha de teste" : "Histórico de calibração não incluído nesta análise.",
			vizinhos: [],
		},
		etapas: primeiro
			? [
					{
						id: 1,
						de: "pendente",
						para: "mapeando_contexto",
						detalhe: "Recebida · contexto obtido.",
						criado_em: recebido,
					},
					{
						id: 2,
						de: "mapeando_contexto",
						para: "testando_sandbox",
						detalhe: "Contexto do PR mapeado.",
						criado_em: "2026-11-09T11:00:01Z",
					},
					{
						id: 3,
						de: "testando_sandbox",
						para: "avaliando_risco",
						detalhe: "Testes executados: 10 passaram e 1 falhou.",
						criado_em: "2026-11-09T11:00:46Z",
					},
					{
						id: 4,
						de: "avaliando_risco",
						para: "aguardando_aprovacao",
						detalhe: "Risco alto; aguardando revisão humana.",
						criado_em: "2026-11-09T11:00:47Z",
					},
				]
			: [],
		tentativas: primeiro
			? [
					{
						tentativa: 1,
						compilou: true,
						testes_passaram: 10,
						testes_falharam: 1,
						saida: "--- FAIL: TestFretePesoZero\nFAIL · 1 de 11 testes falhou",
						saida_truncada: false,
						duracao_ms: 42459,
						tipo_falha: "teste",
						criado_em: "2026-11-09T11:00:04Z",
					},
				]
			: [],
		aprovacoes: [],
		recibo_id: primeiro ? "6d290152-af24-4462-a525-72adf3b22d6c" : null,
		testes_estruturados_disponiveis: false,
		rede_isolada: null,
	};
});

export const complementosDemonstracao: Record<string, ComplementoDemonstracao> = {
	[referencias[0].id]: {
		contextoCalibracao: "1 teste falhou nesta execução.",
		ajudaCalibracao: "O histórico não reduz o risco de uma execução com falha nos testes.",
		etapas: [
			{
				id: 1,
				titulo: "Recebida",
				estado: "concluida",
				momento: "2026-11-09T11:00:00Z",
				detalhe: "Análise recebida.",
			},
			{
				id: 2,
				titulo: "Contexto mapeado",
				estado: "concluida",
				momento: "2026-11-09T11:00:01Z",
				detalhe: "Código e contexto obtidos.",
			},
			{
				id: 3,
				titulo: "Testes executados",
				estado: "falhou",
				momento: "2026-11-09T11:00:04Z",
				detalhe: "10 testes passaram; TestFretePesoZero falhou.",
			},
			{
				id: 4,
				titulo: "Risco avaliado",
				estado: "concluida",
				momento: "2026-11-09T11:00:46Z",
				detalhe: "Risco alto devido à falha nos testes.",
			},
			{
				id: 5,
				titulo: "Aguardando aprovação",
				estado: "aguardando",
				momento: "2026-11-09T11:00:47Z",
				detalhe: "Aguardando revisão humana.",
			},
		],
	},
};
