import { Esqueleto, LinhasDeEsqueleto } from "@/componentes/Esqueleto";
import { FaixaDeMetricas, Metrica } from "@/componentes/FaixaDeMetricas";
import { EscalaDoMapa, MapaDeCalor } from "@/componentes/MapaDeCalor";
import { Pagina } from "@/componentes/Pagina";
import { LinhaDeContagem, Painel, TabelaDeContagem } from "@/componentes/Painel";
import { agora, analises, instalacao } from "@/lib/dados";
import { useEstadoDaTela } from "@/lib/estado-da-tela";
import { espera, horaDoDia, periodo, porcentagem } from "@/lib/formatos";
import type { TipoDeFalha } from "@/lib/tipos";
import { analisesPorDia, bloqueio, falhasDoSandbox, filaDeAprovacao, totaisPorNivel } from "@/lib/visao-geral";
import { FilaDeAprovacao } from "@/telas/FilaDeAprovacao";

const nomeDaFalha: Record<Exclude<TipoDeFalha, "nenhuma">, string> = {
	teste: "Teste",
	compilacao: "Compilação",
	infraestrutura: "Infraestrutura",
};

function MetricasDoPeriodo() {
	const fila = filaDeAprovacao();
	const maisAntiga = fila[0];
	const taxa = bloqueio();
	const falhas = falhasDoSandbox();
	const totais = totaisPorNivel();
	const dias = instalacao.periodo_dias;

	return (
		<FaixaDeMetricas rotulo={`Números dos últimos ${dias} dias`}>
			<Metrica
				rotulo="Aguardando aprovação"
				valor={String(fila.length)}
				unidade={fila.length === 1 ? "análise" : "análises"}
				parte={fila.length}
				todo={taxa.bloqueadas}
				apoio={
					maisAntiga
						? `de ${taxa.bloqueadas} bloqueadas, mais antiga há ${espera(maisAntiga.desde, agora)}`
						: `de ${taxa.bloqueadas} bloqueadas, nenhuma à espera`
				}
			/>
			<Metrica
				rotulo="Taxa de bloqueio"
				valor={porcentagem(taxa.bloqueadas, taxa.avaliadas)}
				unidade="das avaliadas"
				parte={taxa.bloqueadas}
				todo={taxa.avaliadas}
				apoio={`${taxa.bloqueadas} de ${taxa.avaliadas} análises em ${dias} dias`}
			/>
			<Metrica
				rotulo="Falhas no sandbox"
				valor={porcentagem(falhas.falhas, falhas.execucoes)}
				unidade="das execuções"
				parte={falhas.falhas}
				todo={falhas.execucoes}
				apoio={`${falhas.falhas} de ${falhas.execucoes} execuções em ${dias} dias`}
			/>
			<Metrica
				rotulo="Análises recebidas"
				valor={String(analises.length)}
				unidade={`em ${dias} dias`}
				parte={totais.comNivel}
				todo={totais.total}
				apoio={`${totais.comNivel} avaliadas pelo gate, ${totais.semNivel} sem nível`}
			/>
		</FaixaDeMetricas>
	);
}

function PaineisDoPeriodo() {
	const taxa = bloqueio();
	const falhas = falhasDoSandbox();
	const dias = analisesPorDia();

	return (
		<div className="grid grid-cols-[minmax(0,1fr)] gap-4 md:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,2fr)]">
			<Painel titulo="Bloqueadas" resumo={`${taxa.bloqueadas} por desfecho`}>
				<TabelaDeContagem legenda="Desfecho das análises bloqueadas, em número de análises">
					<LinhaDeContagem rotulo="Aguardando" parte={taxa.aguardando} todo={taxa.bloqueadas} />
					<LinhaDeContagem rotulo="Aprovadas" parte={taxa.aprovadas} todo={taxa.bloqueadas} />
					<LinhaDeContagem rotulo="Com alterações" parte={taxa.comAlteracoes} todo={taxa.bloqueadas} />
				</TabelaDeContagem>
			</Painel>
			<Painel titulo="Falhas por tipo" resumo={`das ${falhas.falhas} falhas`}>
				<TabelaDeContagem legenda="Falhas do sandbox por tipo, em número de execuções">
					{falhas.porTipo.map(({ tipo, quantas }) => (
						<LinhaDeContagem key={tipo} rotulo={nomeDaFalha[tipo]} parte={quantas} todo={falhas.falhas} />
					))}
				</TabelaDeContagem>
			</Painel>
			<Painel titulo="Análises por dia e nível" resumo={<EscalaDoMapa />} className="md:max-xl:col-span-2">
				<MapaDeCalor dias={dias} legenda="Análises por dia de chegada e por nível de risco" className="flex-1 px-3 py-2" />
			</Painel>
		</div>
	);
}

function EsqueletoDaVisaoGeral() {
	return (
		<div className="flex flex-col gap-4">
			<div className="grid grid-cols-2 gap-4 xl:grid-cols-4">
				<Esqueleto className="h-[118px] rounded-painel" />
				<Esqueleto className="h-[118px] rounded-painel" />
				<Esqueleto className="h-[118px] rounded-painel" />
				<Esqueleto className="h-[118px] rounded-painel" />
			</div>
			<LinhasDeEsqueleto quantas={6} />
			<div className="grid grid-cols-[minmax(0,1fr)] gap-4 md:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,2fr)]">
				<Esqueleto className="h-[132px] rounded-painel" />
				<Esqueleto className="h-[132px] rounded-painel" />
				<Esqueleto className="h-[132px] rounded-painel md:max-xl:col-span-2" />
			</div>
		</div>
	);
}

export function VisaoGeral() {
	const estado = useEstadoDaTela();

	return (
		<Pagina
			titulo="Visão geral"
			tituloDoDocumento="Visão geral"
			contexto={`Últimos ${instalacao.periodo_dias} dias · ${periodo(agora, instalacao.periodo_dias)} · atualizado às ${horaDoDia(agora)}`}
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
			</div>
		</Pagina>
	);
}
