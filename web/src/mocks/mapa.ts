import type { Detalhe } from "@/servicos/analises";
import type { NivelMapa } from "@/servicos/mapa";
import { analisesDemonstracao } from "./analises-demonstracao";
import { visaoGeralMock } from "./visao-geral";

type Referencia = { repositorio: string; pr: number; hora: string };
// Referências literais dos seletores do Figma. Os demais recortes têm exemplos
// públicos fictícios para desenvolver o fluxo, sem representar dados do núcleo.
const referenciasFigma: Record<string, Referencia[]> = {
	"2026-11-08:alto": [
		{ repositorio: "cobranca-api", pr: 215, hora: "13:00" },
		{ repositorio: "painel-entregas", pr: 96, hora: "10:00" },
	],
	"2026-11-11:alto": [
		{ repositorio: "painel-entregas", pr: 97, hora: "06:40" },
		{ repositorio: "cobranca-api", pr: 217, hora: "10:55" },
		{ repositorio: "painel-entregas", pr: 99, hora: "13:12" },
	],
};
const niveis: NivelMapa[] = ["alto", "medio", "baixo"];
export const referenciasMapaMock = visaoGeralMock.mapa.flatMap((dia, indiceDia) =>
	niveis.flatMap((nivel, indiceNivel) =>
		Array.from({ length: dia[nivel] }, (_, indice) => {
			const ref = referenciasFigma[`${dia.data}:${nivel}`]?.[indice] ?? {
				repositorio: "repositorio-demonstracao",
				pr: 1000 + indiceDia * 9 + indiceNivel * 3 + indice,
				hora: `${String(10 + indice).padStart(2, "0")}:00`,
			};
			const id = `mapa-${dia.data}-${nivel}-${indice + 1}`;
			const momento = `${dia.data}T${ref.hora}:00Z`;
			const base = analisesDemonstracao.find((a) => a.analise.repositorio === ref.repositorio && a.analise.numero_pr === ref.pr);
			const detalhe: Detalhe = {
				installation_id: 0,
				analise: {
					id,
					repositorio: ref.repositorio,
					numero_pr: ref.pr,
					sha_head: `${dia.data.replaceAll("-", "")}${indiceNivel + 1}${indice + 1}`.padEnd(40, "0"),
					autor: "agente-demo[bot]",
					estado: nivel === "alto" ? "aguardando_aprovacao" : "concluido",
					nivel,
					recebido_em: momento,
					criado_em: momento,
					atualizado_em: momento,
				},
				branch: base?.branch ?? "demonstracao/mapa",
				url_pr: "",
				url_check: null,
				atual: true,
				gate: {
					nivel_deterministico: nivel,
					nivel,
					motivo: base?.gate?.motivo ?? "Classificação apresentada no cenário simulado do mapa.",
					motivos: base?.gate?.motivos ?? [
						{
							id: "cenario_demonstracao",
							nivel,
							descricao: "Referência fictícia para desenvolver o painel.",
							evidencias: ["Este cenário não representa uma avaliação executada no núcleo."],
						},
					],
					calibracao_estado: "nao_informada",
					calibracao_razao: "Histórico de calibração não incluído nesta referência simulada.",
					vizinhos: [],
				},
				etapas: [],
				tentativas: [],
				aprovacoes: [],
				recibo_id: null,
				testes_estruturados_disponiveis: false,
				rede_isolada: null,
			};
			return { data: dia.data, nivel, avaliadoEm: momento, detalhe };
		}),
	),
);
export const diasMapaMock = visaoGeralMock.mapa.map((d) => d.data);
