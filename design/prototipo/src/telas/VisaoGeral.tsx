import { Esqueleto, EsqueletoDeTabela } from "@/componentes/Esqueleto";
import { EsqueletoDaFaixa, FaixaDeMetricas, Metrica } from "@/componentes/FaixaDeMetricas";
import { EscalaDoMapa, MapaDeCalor } from "@/componentes/MapaDeCalor";
import { Pagina } from "@/componentes/Pagina";
import { LinhaDeContagem, Painel, TabelaDeContagem } from "@/componentes/Painel";
import { UltimoRecibo } from "@/componentes/UltimoRecibo";
import { agora, analises, instalacao, nomeCurtoDoRepositorio } from "@/lib/dados";
import { type EstadoDaTela, useEstadoDaTela } from "@/lib/estado-da-tela";
import { espera, horaDoDia, periodo, porcentagem } from "@/lib/formatos";
import type { TipoDeFalha } from "@/lib/tipos";
import { analisesPorDia, bloqueio, falhasDoSandbox, filaDeAprovacao, totaisPorNivel } from "@/lib/visao-geral";
import { FilaDeAprovacao } from "@/telas/FilaDeAprovacao";

const nomeDaFalha: Record<Exclude<TipoDeFalha, "nenhuma">, string> = {
	teste: "Teste",
	compilacao: "Compilação",
	infraestrutura: "Infraestrutura",
};

function MetricaDaEsperaMaisAntiga() {
	const fila = filaDeAprovacao();
	const maisAntiga = fila[0];

	if (!maisAntiga) {
		return <Metrica rotulo="Espera mais antiga" valor="nenhuma" apoio="ninguém espera aprovação agora" />;
	}

	return (
		<Metrica
			rotulo="Espera mais antiga"
			valor={espera(maisAntiga.desde, agora)}
			apoio={`${nomeCurtoDoRepositorio(maisAntiga.analise.repository)} #${maisAntiga.analise.pr_number} · 1 de ${fila.length} na fila`}
		/>
	);
}

function MetricasDoPeriodo() {
	const taxa = bloqueio();
	const falhas = falhasDoSandbox();
	const totais = totaisPorNivel();
	const dias = instalacao.periodo_dias;

	return (
		<FaixaDeMetricas rotulo={`Números dos últimos ${dias} dias`}>
			<MetricaDaEsperaMaisAntiga />
			<Metrica
				rotulo="Taxa de bloqueio"
				valor={porcentagem(taxa.bloqueadas, taxa.avaliadas)}
				unidade="das avaliadas"
				fracao={{ parte: taxa.bloqueadas, todo: taxa.avaliadas }}
				apoio={`${taxa.bloqueadas} de ${taxa.avaliadas} análises em ${dias} dias`}
			/>
			<Metrica
				rotulo="Falhas no sandbox"
				valor={porcentagem(falhas.falhas, falhas.execucoes)}
				unidade="das execuções"
				fracao={{ parte: falhas.falhas, todo: falhas.execucoes }}
				apoio={`${falhas.falhas} de ${falhas.execucoes} execuções em ${dias} dias; cada tentativa conta`}
			/>
			<Metrica
				rotulo="Análises recebidas"
				valor={String(analises.length)}
				unidade={`em ${dias} dias`}
				apoio={`${totais.comNivel} passaram pelo gate · ${totais.semNivel} em andamento ou paradas antes do gate`}
			/>
		</FaixaDeMetricas>
	);
}

function PaineisDoPeriodo() {
	const taxa = bloqueio();
	const falhas = falhasDoSandbox();
	const dias = analisesPorDia();

	return (
		<div className="grid grid-cols-[minmax(0,1fr)] gap-4 md:grid-cols-2">
			<Painel titulo="Bloqueadas" resumo={`${taxa.bloqueadas} no período, por desfecho`}>
				<TabelaDeContagem legenda="Desfecho das análises bloqueadas, em número de análises">
					<LinhaDeContagem rotulo="Aguardando" parte={taxa.aguardando} todo={taxa.bloqueadas} />
					<LinhaDeContagem rotulo="Aprovadas" parte={taxa.aprovadas} todo={taxa.bloqueadas} />
					<LinhaDeContagem rotulo="Alterações pedidas" parte={taxa.comAlteracoes} todo={taxa.bloqueadas} />
				</TabelaDeContagem>
			</Painel>
			<Painel titulo="Falhas por tipo" resumo={`das ${falhas.falhas} falhas`}>
				<TabelaDeContagem legenda="Falhas do sandbox por tipo, em número de execuções">
					{falhas.porTipo.map(({ tipo, quantas }) => (
						<LinhaDeContagem key={tipo} rotulo={nomeDaFalha[tipo]} parte={quantas} todo={falhas.falhas} />
					))}
				</TabelaDeContagem>
			</Painel>
			<Painel titulo="Análises por dia e nível" resumo={<EscalaDoMapa />} className="md:col-span-2">
				<MapaDeCalor dias={dias} legenda="Análises por dia de chegada e por nível de risco" className="flex-1 px-3 pb-2" />
			</Painel>
		</div>
	);
}

function EsqueletoDaVisaoGeral() {
	return (
		<div className="flex flex-col gap-4">
			<EsqueletoDaFaixa />
			<EsqueletoDeTabela linhas={5} />
			<div className="grid grid-cols-[minmax(0,1fr)] gap-4 md:grid-cols-2">
				<Esqueleto className="h-[145px] rounded-painel" />
				<Esqueleto className="h-[145px] rounded-painel" />
				<Esqueleto className="h-[180px] rounded-painel md:col-span-2" />
			</div>
		</div>
	);
}

function contextoDoEstado(estado: EstadoDaTela) {
	const periodoDaTela = `Últimos ${instalacao.periodo_dias} dias · ${periodo(agora, instalacao.periodo_dias)}`;
	const hora = horaDoDia(agora);
	if (estado === "carregando") return `${periodoDaTela} · Atualizando…`;
	if (estado === "erro") return `${periodoDaTela} · Última atualização às ${hora}`;
	return `${periodoDaTela} · atualizado às ${hora}`;
}

export function VisaoGeral() {
	const estado = useEstadoDaTela();

	return (
		<Pagina
			titulo="Visão geral"
			tituloDoDocumento="Visão geral"
			contexto={contextoDoEstado(estado)}
			estado={estado}
			oQueCarrega="a visão geral"
			vazio={{
				titulo: `Nenhuma análise nos últimos ${instalacao.periodo_dias} dias`,
				texto:
					"O aPRova analisa cada pull request aberto ou atualizado nos repositórios desta instalação. Quando o primeiro chegar, a fila de aprovação e os números do período aparecem aqui.",
			}}
			esqueleto={<EsqueletoDaVisaoGeral />}
		>
			<div className="flex flex-col gap-4">
				<MetricasDoPeriodo />
				<FilaDeAprovacao />
				<PaineisDoPeriodo />
				<UltimoRecibo />
			</div>
		</Pagina>
	);
}
