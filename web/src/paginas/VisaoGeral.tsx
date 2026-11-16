import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { Ativo } from "@/componentes/Ativo";
import { Botao, Campo, Modal, PainelDeEstado } from "@/componentes/Primitivos";
import { Selecao } from "@/componentes/Selecao";
import { consultarVisaoGeral, type ItemDaFila, type VisaoGeralDados } from "@/servicos/visao-geral";
import "@/estilos/visao-geral.css";

function DetalheDaFila({ item }: { item: ItemDaFila }) {
	const [q] = useSearchParams();
	const retorno = new URLSearchParams();
	for (const nome of ["fila", "repositorio"]) {
		const valor = q.get(nome);
		if (valor) retorno.set(nome, valor);
	}
	return (
		<Link className="link-fila" to={`/analises/demonstracao/${item.id}?${new URLSearchParams({ retorno: retorno.toString() })}`}>
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
function Mapa({ dias }: { dias: VisaoGeralDados["mapa"] }) {
	return (
		<section className="painel-overview">
			<h2>Análises por dia e nível</h2>
			<div className="conteudo-overview">
				<div className="legenda-mapa apoio">
					Por dia{" "}
					{["0", "1", "2", "3+"].map((v, i) => (
						<span key={v}>
							<i style={{ opacity: i ? 0.5 + i / 6 : 0.12 }} />
							{v}
						</span>
					))}
				</div>
				<div className="mapa-com-rotulos">
					<div className="rotulos-mapa">
						{["Alto", "Médio", "Baixo"].map((n) => (
							<span key={n}>{n}</span>
						))}
					</div>
					<section className="viewport-mapa" aria-label="Mapa de análises de 30 dias, rolagem horizontal">
						{(["alto", "medio", "baixo"] as const).map((nivel) => (
							<div className="linha-mapa" key={nivel}>
								{dias.map((dia) => {
									const n = dia[nivel];
									const data = new Date(`${dia.data}T12:00:00Z`).toLocaleDateString("pt-BR", {
										day: "numeric",
										month: "short",
										timeZone: "UTC",
									});
									const rotulo = `${data} · risco ${nivel === "medio" ? "médio" : nivel}: ${n} análise(s)`;
									return (
										<Modal
											key={dia.data}
											titulo={rotulo}
											descricao="Contagem simulada para validação do dashboard"
											gatilho={
												<button type="button" className={`celula-mapa intensidade-${Math.min(n, 3)} nivel-${nivel}`} aria-label={rotulo}>
													{n >= 2 ? n : ""}
												</button>
											}
										>
											<p>
												{n === 0 ? "Nenhuma análise neste recorte simulado." : `${n} análise(s) neste dia e nível no conjunto simulado.`}
											</p>
										</Modal>
									);
								})}
							</div>
						))}
						<div className="datas-mapa apoio">
							<span>13 out</span>
							<span>20 out</span>
							<span>27 out</span>
							<span>3 nov</span>
							<span>11 nov</span>
						</div>
					</section>
				</div>
				<p className="apoio nota-mapa">
					Cada célula é um dia. Clique para ver as análises. Zero é neutro; a intensidade indica a quantidade.
				</p>
			</div>
		</section>
	);
}
export function VisaoGeral() {
	const [dados, setDados] = useState<VisaoGeralDados>();
	const [erro, setErro] = useState(false);

	const [q, setQ] = useSearchParams();
	useEffect(() => {
		const c = new AbortController();
		setErro(false);
		setDados(undefined);
		void consultarVisaoGeral(c.signal)
			.then((v) => {
				if (!c.signal.aborted) setDados(v);
			})
			.catch(() => {
				if (!c.signal.aborted) setErro(true);
			});
		return () => c.abort();
	}, []);
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
			{erro ? (
				<PainelDeEstado titulo="Não foi possível carregar a visão geral" erro>
					<Botao onClick={() => window.location.reload()}>Tentar novamente</Botao>
				</PainelDeEstado>
			) : !dados ? (
				<p role="status">Carregando visão geral…</p>
			) : (
				<>
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
												<DetalheDaFila item={i} />
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
									<DetalheDaFila item={i} />
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
						<span>Dados simulados · métricas de 30 dias e fila de referência</span>
						<Link to="/relatorio-semanal">Consultar relatório semanal da API</Link>
					</footer>
				</>
			)}
		</div>
	);
}
