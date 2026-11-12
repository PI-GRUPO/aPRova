import { useState } from "react";
import { Link } from "react-router";
import { CampoDeBusca } from "@/componentes/CampoDeBusca";
import { Identificador } from "@/componentes/Identificador";
import { MotivosDaAnalise } from "@/componentes/LinkDoMotivo";
import { Selecao } from "@/componentes/Selecao";
import { SeloDeRisco } from "@/componentes/SeloDeRisco";
import { CabecalhoDaTabela, Celula, Coluna, ColunaOrdenavel, CorpoDaTabela, LinhaDaTabela, Tabela } from "@/componentes/Tabela";
import { cn } from "@/lib/classes";
import { agora, nomeCurtoDoRepositorio } from "@/lib/dados";
import { espera, plural } from "@/lib/formatos";
import type { Analise } from "@/lib/tipos";
import {
	filaDeAprovacao,
	filtrarFila,
	type ItemDaFila,
	ordenarPorEspera,
	repositoriosDaFila,
	todosOsRepositorios,
} from "@/lib/visao-geral";

function evidenciasDaAnalise(analise: Analise) {
	return [...new Set(analise.risk_gate?.rules_hit.flatMap((motivo) => motivo.evidencias) ?? [])];
}

function EvidenciaPrincipal({ analise, quebraLinha = false, className }: { analise: Analise; quebraLinha?: boolean; className?: string }) {
	const [primeira, ...demais] = evidenciasDaAnalise(analise);
	if (!primeira) return null;
	const ehCaminho = !primeira.includes(" ");

	return (
		<span className={cn("flex items-baseline gap-x-2", className)}>
			<span className={cn(quebraLinha ? "break-all whitespace-normal" : "truncate", ehCaminho && "font-mono")} title={primeira}>
				{primeira}
			</span>
			{demais.length > 0 ? <span className="shrink-0 text-12 text-secundario">e mais {demais.length}</span> : null}
		</span>
	);
}

function LinhaDaFila({ analise, desde }: ItemDaFila) {
	return (
		<LinhaDaTabela>
			<Celula className="text-right font-medium">{espera(desde, agora)}</Celula>
			<Celula className="max-md:whitespace-normal">
				<Link to={`/analises/${analise.id}`} className="after:absolute after:inset-0">
					{nomeCurtoDoRepositorio(analise.repository)} <Identificador>#{analise.pr_number}</Identificador>
				</Link>
				<MotivosDaAnalise analise={analise} className="flex-wrap leading-6 md:hidden" />
				<EvidenciaPrincipal analise={analise} className="relative z-10 flex-wrap md:hidden" quebraLinha />
			</Celula>
			<Celula className="relative z-10 max-2xl:hidden">
				<Identificador>{analise.head_branch}</Identificador>
			</Celula>
			<Celula className="relative z-10 text-secundario max-2xl:hidden">
				<Identificador>{analise.head_sha.slice(0, 7)}</Identificador>
			</Celula>
			<Celula className="max-md:hidden">
				<MotivosDaAnalise analise={analise} className="md:max-lg:flex-col" />
			</Celula>
			<Celula className="relative z-10 w-full max-w-0 max-md:hidden">
				<EvidenciaPrincipal analise={analise} />
			</Celula>
		</LinhaDaTabela>
	);
}

export function FilaDeAprovacao() {
	const [busca, setBusca] = useState("");
	const [repositorio, setRepositorio] = useState(todosOsRepositorios);
	const [maiorEsperaPrimeiro, setMaiorEsperaPrimeiro] = useState(true);

	const todaAFila = filaDeAprovacao();
	const fila = ordenarPorEspera(filtrarFila(todaAFila, busca, repositorio), maiorEsperaPrimeiro);
	const filtrada = busca !== "" || repositorio !== todosOsRepositorios;
	const opcoes = [
		{ valor: todosOsRepositorios, rotulo: "Todos os repositórios" },
		...repositoriosDaFila(todaAFila).map((nome) => ({ valor: nome, rotulo: nomeCurtoDoRepositorio(nome) })),
	];

	function limparFiltros() {
		setBusca("");
		setRepositorio(todosOsRepositorios);
	}

	return (
		<section aria-labelledby="titulo-da-fila" className="overflow-hidden rounded-controle border border-borda bg-superficie">
			<div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-borda px-3 py-2">
				<h2 id="titulo-da-fila" className="text-14 font-semibold">
					Aguardando aprovação
				</h2>
				<SeloDeRisco nivel="alto" variante="ponto" comPrefixo />
				<search aria-label="Filtros da fila" className="flex flex-wrap items-center gap-x-3 gap-y-2 max-md:w-full md:ml-auto">
					<CampoDeBusca
						rotulo="Buscar na fila"
						valor={busca}
						aoMudar={setBusca}
						exemplo="PR, branch ou motivo"
						className="max-md:w-full md:w-56"
					/>
					<Selecao
						rotulo="Repositório"
						valor={repositorio}
						aoMudar={setRepositorio}
						opcoes={opcoes}
						className="max-md:min-w-0 max-md:flex-1 md:w-52"
					/>
					<p role="status" className="text-12 text-secundario">
						{fila.length} de {plural(todaAFila.length, "análise", "análises")}
					</p>
				</search>
			</div>

			{todaAFila.length === 0 ? (
				<p className="px-3 py-6 text-secundario">
					Nenhuma análise espera por aprovação agora. Um PR de risco alto aparece aqui quando o Check Run bloqueia o merge.
				</p>
			) : (
				<div className="overflow-x-auto">
					<Tabela legenda="Análises aguardando aprovação, com tempo de espera e motivos do risco">
						<CabecalhoDaTabela>
							<ColunaOrdenavel
								sentido={maiorEsperaPrimeiro ? "decrescente" : "crescente"}
								aoOrdenar={() => setMaiorEsperaPrimeiro((atual) => !atual)}
								aDireita
								className="w-0"
							>
								Espera
							</ColunaOrdenavel>
							<Coluna>Pull request</Coluna>
							<Coluna className="max-2xl:hidden">Branch</Coluna>
							<Coluna className="max-2xl:hidden">Commit</Coluna>
							<Coluna className="max-md:hidden">Motivos</Coluna>
							<Coluna className="max-md:hidden">Evidência</Coluna>
						</CabecalhoDaTabela>
						<CorpoDaTabela>
							{fila.map((item) => (
								<LinhaDaFila key={item.analise.id} {...item} />
							))}
						</CorpoDaTabela>
					</Tabela>
					{fila.length === 0 ? (
						<p className="px-3 py-6 text-secundario">
							Nenhuma análise da fila corresponde ao filtro.{" "}
							{filtrada ? (
								<button type="button" onClick={limparFiltros} className="cursor-pointer text-link underline underline-offset-[3px]">
									Limpar filtros
								</button>
							) : null}
						</p>
					) : null}
				</div>
			)}
		</section>
	);
}
