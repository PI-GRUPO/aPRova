import { Link } from "react-router";
import { useSessao } from "@/app/sessao";
import { Ativo } from "@/componentes/Ativo";
import { ConsultaPrivada, useEscopo } from "@/componentes/ConsultaPrivada";
import { FilaDePendencias } from "@/componentes/FilaDePendencias";
import { MapaDeAnalises as Mapa } from "@/componentes/MapaDeAnalises";
import { PainelDeEstado } from "@/componentes/Primitivos";
import { consultarVisaoGeral } from "@/servicos/visao-geral";
import { Protegida } from "./Acesso";
import "@/estilos/visao-geral.css";

function Distribuicao({ titulo, apoio, itens }: { titulo: string; apoio: string; itens: { nome: string; quantidade: number }[] }) {
	const total = itens.reduce((s, i) => s + i.quantidade, 0);
	return (
		<section className="painel-overview">
			<h2>{titulo}</h2>
			<div className="conteudo-overview">
				<p className="apoio">Dados simulados · {apoio}</p>
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
	const { instalacao } = useEscopo("");
	const { sessao } = useSessao();
	const q = new URLSearchParams({ installation_id: instalacao });
	return (
		<div className="pagina pagina-overview">
			<h1>Visão geral</h1>
			<Protegida>
				{sessao?.autenticado && sessao.instalacoes.some((i) => String(i.id) === instalacao) ? (
					<>
						<ConsultaPrivada
							secao
							titulo="Métricas de 30 dias"
							consultar={consultarVisaoGeral}
							filtrosDaConsulta={[]}
							ajuda={
								<>
									<p>As métricas de 30 dias e o mapa diário estão indisponíveis.</p>
									<Link to={`/analises?${q}&estado=aguardando_aprovacao&periodo=todos`}>Consultar análises aguardando aprovação</Link>
								</>
							}
						>
							{(dados) => (
								<>
									<p className="apoio periodo-overview">
										Métricas simuladas · 30 dias <span className="periodo-desktop">· {dados.periodo} </span>· atualizado às{" "}
										{dados.atualizadoEm}
									</p>
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
								</>
							)}
						</ConsultaPrivada>
						<FilaDePendencias />
						{__MOCKS__ && (
							<ConsultaPrivada secao titulo="Distribuições e mapa" consultar={consultarVisaoGeral} filtrosDaConsulta={[]}>
								{(dados) => (
									<>
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
												Último recibo assinado · simulado · {dados.ultimoRecibo.repositorio} #{dados.ultimoRecibo.pr}
											</h2>
											<div>
												<p className="apoio mono">
													{dados.ultimoRecibo.hash} · Ed25519, encadeado a {dados.ultimoRecibo.anterior} · {dados.ultimoRecibo.data}
												</p>
												<Link to={`/recibos?${q}`}>
													<Ativo nome="external" />
													Ver os recibos
												</Link>
											</div>
										</section>
									</>
								)}
							</ConsultaPrivada>
						)}
						<footer className="rodape-overview apoio">
							<Link to={`/relatorio-semanal?${q}`}>Consultar relatório semanal</Link>
						</footer>
					</>
				) : (
					<PainelDeEstado titulo={instalacao ? "Instalação não autorizada" : "Nenhuma instalação autorizada"}>
						<p>Confira as instalações disponíveis na sua conta.</p>
					</PainelDeEstado>
				)}
			</Protegida>
		</div>
	);
}
