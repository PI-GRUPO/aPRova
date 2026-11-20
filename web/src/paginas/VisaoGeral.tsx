import { Link, useSearchParams } from "react-router";
import { Ativo } from "@/componentes/Ativo";
import { ConsultaPrivada, useEscopo } from "@/componentes/ConsultaPrivada";
import { MapaDeAnalises as Mapa } from "@/componentes/MapaDeAnalises";
import { Campo } from "@/componentes/Primitivos";
import { Selecao } from "@/componentes/Selecao";
import { consultarVisaoGeral, destinoAnalise, type ItemDaFila, type VisaoGeralDados } from "@/servicos/visao-geral";
import "@/estilos/visao-geral.css";

function DetalheDaFila({ item, origem }: { item: ItemDaFila; origem: VisaoGeralDados["origem"] }) {
	const [q] = useSearchParams();
	const { instalacao } = useEscopo("");
	const retorno = new URLSearchParams();
	for (const nome of ["installation_id", "fila", "repositorio"]) {
		const valor = q.get(nome);
		if (valor) retorno.set(nome, valor);
	}
	return (
		<Link className="link-fila" to={destinoAnalise(item.id, origem, instalacao, retorno)}>
			{item.repositorio} #{item.pr}
		</Link>
	);
}

function Distribuicao({ titulo, apoio, itens }: { titulo: string; apoio: string; itens: { nome: string; quantidade: number }[] }) {
	const total = itens.reduce((s, i) => s + i.quantidade, 0);
	return (
		<section className="painel-overview">
			<h2>{titulo}</h2>
			<div className="conteudo-overview">
				<p className="apoio">{apoio}</p>
				{itens.map((i) => (
					<div className="linha-distribuicao" key={i.nome}>
						<span>{i.nome}</span>
						<div className="barra-overview">
							<span style={{ width: `${total ? (i.quantidade / total) * 100 : 0}%` }} />
						</div>
						<span className="apoio">
							{i.quantidade} de {total}
						</span>
					</div>
				))}
			</div>
		</section>
	);
}

export function VisaoGeral() {
	const escopo = useEscopo("");
	const q = new URLSearchParams({ installation_id: escopo.instalacao });
	return (
		<ConsultaPrivada
			titulo="Visão geral"
			consultar={consultarVisaoGeral}
			filtrosLocais
			ajuda={
				<>
					<p>As métricas de 30 dias e o mapa de risco ainda estão indisponíveis.</p>
					<Link to={`/analises?${q}&estado=aguardando_aprovacao&periodo=todos`}>Consultar análises aguardando aprovação</Link>
					<br />
					<Link to={`/relatorio-semanal?${q}`}>Consultar relatório semanal</Link>
				</>
			}
		>
			{(dados) => <Conteudo dados={dados} />}
		</ConsultaPrivada>
	);
}
function Conteudo({ dados }: { dados: VisaoGeralDados }) {
	const [q, setQ] = useSearchParams();
	const busca = q.get("fila") ?? "";
	const repo = q.get("repositorio") ?? "";
	function filtrar(nome: string, valor: string) {
		setQ(
			(anterior) => {
				const novo = new URLSearchParams(anterior);
				if (valor) novo.set(nome, valor);
				else novo.delete(nome);
				return novo;
			},
			{ replace: true },
		);
	}
	const fila =
		dados?.fila.filter(
			(i) =>
				(!repo || i.repositorio === repo) &&
				`${i.repositorio} #${i.pr} ${i.branch} ${i.motivo} ${i.evidencia}`
					.toLocaleLowerCase("pt-BR")
					.includes(busca.toLocaleLowerCase("pt-BR")),
		) ?? [];
	return (
		<div className="pagina pagina-overview">
			<div className="titulo-overview">
				<h1>Visão geral</h1>
				<p className="apoio">
					Últimos 30 dias <span className="periodo-desktop">· {dados?.periodo ?? "—"} </span>· atualizado às {dados?.atualizadoEm ?? "—"}
				</p>
			</div>

			<section className="faixa-overview" aria-label="Métricas dos últimos 30 dias">
				{dados.metricas.map((m) => (
					<section className="metrica-overview" key={m.rotulo}>
						<h2>{m.rotulo}</h2>
						<p className="valor-overview">{m.valor}</p>
						<p className="apoio">{m.unidade}</p>
						<div className="espaco-proporcao">
							{m.proporcao !== undefined && (
								<div className="barra-overview">
									<span style={{ width: `${m.proporcao * 100}%` }} />
								</div>
							)}
						</div>
						<p className="apoio contexto-metrica-desktop">{m.contexto}</p>
						<p className="apoio contexto-metrica-mobile">{m.contextoMobile ?? m.contexto}</p>
					</section>
				))}
			</section>
			<section className="fila-overview">
				<h2>Aguardando aprovação · risco alto</h2>
				<div className="filtros-fila">
					<Campo
						type="search"
						rotulo="Buscar na fila"
						placeholder="PR, branch, commit, motivo ou evidência"
						value={busca}
						onChange={(e) => filtrar("fila", e.target.value)}
					/>
					<div className="campo">
						<span className="rotulo-campo">Repositório</span>
						<Selecao
							rotulo="Repositório"
							valor={repo || "todos"}
							aoMudar={(v) => filtrar("repositorio", v === "todos" ? "" : v)}
							opcoes={[
								{ valor: "todos", rotulo: "Todos os repositórios" },
								...Array.from(new Set(dados.fila.map((i) => i.repositorio))).map((r) => ({ valor: r, rotulo: r })),
							]}
						/>
					</div>
					<p className="apoio contagem-fila" role="status">
						{fila.length} de {dados.fila.length} análises
					</p>
				</div>
				<div className="tabela-fila-viewport">
					<table className="tabela-fila">
						<thead>
							<tr>
								{["Espera", "Pull request", "Branch", "Motivos", "Evidências"].map((c) => (
									<th key={c}>{c}</th>
								))}
							</tr>
						</thead>
						<tbody>
							{fila.map((i) => (
								<tr key={i.id}>
									<td>{i.espera}</td>
									<td>
										<DetalheDaFila item={i} origem={dados.origem} />
									</td>
									<td className="mono">{i.branch}</td>
									<td className="mono">{i.motivo}</td>
									<td className={i.evidencia.includes("testes") ? "" : "mono"}>{i.evidencia}</td>
								</tr>
							))}
						</tbody>
					</table>
				</div>
				<div className="cards-fila">
					{fila.map((i) => (
						<article key={i.id}>
							<DetalheDaFila item={i} origem={dados.origem} />
							<p className="mono">{i.motivo.split(" +")[0]}</p>
							<p className="apoio">{i.evidencia.split(" · ")[0]}</p>
						</article>
					))}
				</div>
				{!fila.length && <p className="apoio vazio-fila">Nenhuma análise encontrada com esses filtros.</p>}
			</section>
			<div className="distribuicoes-overview">
				<Distribuicao
					titulo="Bloqueadas"
					apoio={`${dados.bloqueadas.reduce((s, i) => s + i.quantidade, 0)} no período, por desfecho`}
					itens={dados.bloqueadas}
				/>
				<Distribuicao
					titulo="Falhas por tipo"
					apoio={`das ${dados.falhas.reduce((s, i) => s + i.quantidade, 0)} falhas`}
					itens={dados.falhas}
				/>
			</div>
			<Mapa dias={dados.mapa} />
			<section className="ultimo-recibo-overview">
				<h2>
					Último recibo assinado · {dados.ultimoRecibo.repositorio} #{dados.ultimoRecibo.pr}
				</h2>
				<div>
					<p className="apoio mono">
						{dados.ultimoRecibo.hash} · Ed25519, encadeado a {dados.ultimoRecibo.anterior} · {dados.ultimoRecibo.data}
					</p>
					<Link to="/recibos">
						<Ativo nome="external" />
						Ver os recibos
					</Link>
				</div>
			</section>
			<footer className="rodape-overview apoio">
				<Link to="/relatorio-semanal">Consultar relatório semanal</Link>
			</footer>
		</div>
	);
}
