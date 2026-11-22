import type { Consulta } from "./fonte";
import { data, inteiro, invalido, lerPainel, lista, registro, texto } from "./painel";
import { uuidValido } from "./recibos";
import type { VisaoGeralDados } from "./visao-geral";

export const metricasReais: Consulta<VisaoGeralDados> = async (escopo, _filtros, signal) => {
	const { dados, inicio, fim, snapshot } = await lerPainel("/api/metricas", escopo, new URLSearchParams(), signal);
	const recebidas = inteiro(dados.recebidas),
		avaliadas = inteiro(dados.avaliadas),
		alto = inteiro(dados.alto);
	const f = registro(dados.falhas);
	const falhas = Object.entries({ infraestrutura: "Infraestrutura", teste: "Teste", compilacao: "Compilação" }).map(([k, nome]) => ({
		nome,
		quantidade: inteiro(f[k]),
	}));
	const bloqueios = registro(dados.bloqueios);
	const totalFalhas = falhas.reduce((s, f) => s + f.quantidade, 0),
		tentativas = inteiro(dados.tentativas);
	if (alto > avaliadas || avaliadas > recebidas || totalFalhas > tentativas) return invalido();
	const pendentes = inteiro(dados.pendentes),
		semInicio = inteiro(dados.pendentes_sem_inicio);
	const desde = dados.desde === null ? null : data(dados.desde);
	const media = dados.tempo_medio_segundos;
	if (media !== null && (typeof media !== "number" || media < 0 || !Number.isFinite(media))) return invalido();
	const dias = lista(dados.mapa, (d) => ({
		data: data(d.data),
		alto: inteiro(d.alto),
		medio: inteiro(d.medio),
		baixo: inteiro(d.baixo),
		naoAvaliado: inteiro(d.nao_avaliado),
	}));
	const porData = new Map(dias.map((d) => [d.data, d]));
	if (
		porData.size !== dias.length ||
		dias.some((d) => !/^\d{4}-\d{2}-\d{2}$/.test(d.data) || d.data < inicio.slice(0, 10) || d.data >= fim.slice(0, 10)) ||
		dias.reduce((s, d) => s + d.alto + d.medio + d.baixo + d.naoAvaliado, 0) !== recebidas ||
		dias.reduce((s, d) => s + d.alto, 0) !== alto ||
		dias.reduce((s, d) => s + d.naoAvaliado, 0) !== recebidas - avaliadas ||
		semInicio > pendentes
	)
		return invalido();
	const mapa = [];
	for (let i = 0; i < 30; i++) {
		const dia = new Date(Date.parse(inicio) + 86400000 * i).toISOString().slice(0, 10);
		mapa.push(porData.get(dia) ?? { data: dia, alto: 0, medio: 0, baixo: 0 });
	}
	let ultimoRecibo = null;
	if (dados.ultimo_recibo !== null) {
		const r = registro(dados.ultimo_recibo);
		const id = texto(r.id),
			repo = texto(r.repo);
		if (!/^[0-9a-f]{64}$/.test(texto(r.hash)) || !uuidValido(id) || (escopo.repositorio && repo !== escopo.repositorio)) return invalido();
		ultimoRecibo = { id, repositorio: repo, pr: inteiro(r.pr), hash: texto(r.hash), data: data(r.emissao) };
	}
	const percentual = (n: number, total: number) => (total ? `${Math.round((n * 100) / total)}%` : "Sem dados");
	return {
		origem: "api",
		atualizadoEm: `${new Date(snapshot).toLocaleTimeString("pt-BR", { timeZone: "UTC" })} UTC`,
		periodo: `${inicio.slice(0, 10)} a ${new Date(Date.parse(fim) - 1).toISOString().slice(0, 10)}`,
		mapa,
		metricas: [
			{
				rotulo: "Espera mais antiga",
				valor: !pendentes
					? "Sem pendências"
					: semInicio || !desde
						? "Indisponível"
						: `${Math.floor((Date.parse(snapshot) - Date.parse(desde)) / 60000)} min`,
				unidade: "aguardando aprovação",
				contexto: `${pendentes} pendências atuais · ${semInicio} sem transição registrada`,
			},
			{
				rotulo: "Taxa de bloqueio",
				valor: percentual(alto, avaliadas),
				unidade: "das avaliadas com risco alto",
				contexto: `${alto} de ${avaliadas} avaliadas`,
				proporcao: avaliadas ? alto / avaliadas : undefined,
			},
			{
				rotulo: "Falhas no sandbox",
				valor: percentual(totalFalhas, tentativas),
				unidade: "das tentativas",
				contexto: `${totalFalhas} de ${tentativas} tentativas`,
				proporcao: tentativas ? totalFalhas / tentativas : undefined,
			},
			{
				rotulo: "Análises recebidas",
				valor: String(recebidas),
				unidade: "no período",
				contexto: `${recebidas - avaliadas} sem avaliação · ${media === null ? "Duração média indisponível" : `Tempo médio até o estado final: ${Math.round(media)} s`}`,
			},
		],
		bloqueadas: Object.entries({
			aguardando: "Aguardando",
			aprovadas: "Aprovadas",
			alteracoes: "Alterações pedidas",
			sem_decisao: "Sem decisão para o commit",
		}).map(([k, nome]) => ({ nome, quantidade: inteiro(bloqueios[k]) })),
		falhas,
		ultimoRecibo,
	};
};
