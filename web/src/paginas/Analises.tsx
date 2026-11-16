import { useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useSessao } from "@/app/sessao";
import { dataUTC, Estado, Risco } from "@/componentes/Analise";
import { Botao, Campo, PainelDeEstado } from "@/componentes/Primitivos";
import { Selecao } from "@/componentes/Selecao";
import { CabecalhoDaTabela, Celula, Coluna, ColunaOrdenavel, CorpoDaTabela, LinhaDaTabela, Tabela } from "@/componentes/Tabela";
import { erroDeAnalise, estados, listarAnalises, listarRepositorios, type PaginaAnalises } from "@/servicos/analises";
import { Protegida } from "./Acesso";
import "@/estilos/analises.css";
export function Analises() {
	const { sessao } = useSessao();
	const [q] = useSearchParams();
	const id = q.get("installation_id") ?? (sessao?.autenticado ? String(sessao.instalacoes[0]?.id ?? "") : "");
	return (
		<div className="pagina pagina-analises">
			<div className="titulo-contexto">
				<h1>Análises</h1>
				<p className="apoio">Histórico autorizado · filtros aplicados a toda a consulta · horários em UTC</p>
			</div>
			<Protegida>
				{sessao?.autenticado && sessao.instalacoes.length > 0 ? (
					<ListaDaInstalacao key={`${sessao.usuario.id}:${id}`} id={id} />
				) : (
					<PainelDeEstado titulo="Nenhuma instalação autorizada">
						<p>Não há instalações disponíveis para consultar análises.</p>
					</PainelDeEstado>
				)}
			</Protegida>
		</div>
	);
}
function consultaDaLista(q: URLSearchParams, id: string) {
	const c = new URLSearchParams({ installation_id: id });
	for (const nome of ["repositorio", "numero_pr", "nivel", "estado", "pagina", "limite", "ordem", "snapshot"]) {
		const v = q.get(nome);
		if (v) c.set(nome, v);
	}
	const periodo = q.get("periodo") ?? "30";
	const fim = new Date();
	fim.setUTCHours(0, 0, 0, 0);
	fim.setUTCDate(fim.getUTCDate() + 1);
	const inicio = new Date(fim);
	if (periodo === "todos") inicio.setTime(Date.UTC(1970, 0, 1));
	else if (["7", "30", "90"].includes(periodo)) inicio.setUTCDate(inicio.getUTCDate() - Number(periodo));
	else throw new Error("Período inválido.");
	c.set("inicio", inicio.toISOString().slice(0, 10));
	c.set("fim", fim.toISOString().slice(0, 10));
	return c;
}
function ListaDaInstalacao({ id }: { id: string }) {
	const { sessao } = useSessao();
	const [q, setQ] = useSearchParams();
	const [repos, setRepos] = useState<string[]>();
	const [erroRepos, setErroRepos] = useState("");
	const [atualizacao, setAtualizacao] = useState(0);
	const [busca, setBusca] = useState(q.get("numero_pr") ?? "");
	const [erroBusca, setErroBusca] = useState("");
	const numeroPR = q.get("numero_pr") ?? "";
	useEffect(() => setBusca(numeroPR), [numeroPR]);
	const autorizado = !!sessao?.autenticado && sessao.instalacoes.some((i) => String(i.id) === id);
	useEffect(() => {
		if (!autorizado) return;
		const c = new AbortController();
		listarRepositorios(id, c.signal)
			.then((v) => {
				if (!c.signal.aborted) setRepos(v);
			})
			.catch((e) => {
				if (!c.signal.aborted) setErroRepos(erroDeAnalise(e));
			});
		return () => c.abort();
	}, [id, autorizado]);
	function filtro(nome: string, valor: string) {
		setQ(() => {
			const nova = new URLSearchParams(window.location.search);
			nova.set("installation_id", id);
			if (valor) nova.set(nome, valor);
			else nova.delete(nome);
			nova.delete("pagina");
			nova.delete("snapshot");
			if (nome === "installation_id") nova.delete("repositorio");
			return nova;
		});
	}
	function limpar() {
		setBusca("");
		setErroBusca("");
		setQ(new URLSearchParams({ installation_id: id }));
	}
	return (
		<>
			<div className="controles">
				<div className="campo">
					<span className="rotulo-campo">Instalação</span>
					<Selecao
						rotulo="Instalação"
						valor={id}
						aoMudar={(v) => filtro("installation_id", v)}
						opcoes={sessao?.autenticado ? sessao.instalacoes.map((i) => ({ valor: String(i.id), rotulo: `${i.conta} · ${i.id}` })) : []}
					/>
				</div>
				<Botao onClick={limpar}>Limpar filtros</Botao>
				<Botao
					onClick={() => {
						setAtualizacao((v) => v + 1);
						setQ(() => {
							const nova = new URLSearchParams(window.location.search);
							nova.delete("pagina");
							nova.delete("snapshot");
							return nova;
						});
					}}
				>
					Atualizar resultados
				</Botao>
			</div>
			{!autorizado ? (
				<PainelDeEstado titulo="Instalação não autorizada" erro>
					<p>Selecione uma instalação disponível para sua conta.</p>
				</PainelDeEstado>
			) : (
				<>
					<div className="filtros-analises">
						<form
							onSubmit={(e) => {
								e.preventDefault();
								const v = busca.trim();
								if (v && !/^[1-9]\d*$/.test(v)) {
									setErroBusca("Informe apenas o número positivo do PR.");
									return;
								}
								setErroBusca("");
								filtro("numero_pr", v);
							}}
						>
							<Campo
								rotulo="Buscar PR"
								placeholder="Número do PR"
								inputMode="numeric"
								value={busca}
								onChange={(e) => setBusca(e.target.value)}
								erro={erroBusca}
							/>
							<Botao type="submit">Buscar</Botao>
						</form>
						<Filtro
							rotulo="Repositório"
							valor={q.get("repositorio") ?? "todos"}
							aoMudar={(v) => filtro("repositorio", v === "todos" ? "" : v)}
							opcoes={[{ valor: "todos", rotulo: "Todos" }, ...(repos ?? []).map((r) => ({ valor: r, rotulo: r }))]}
						/>
						<Filtro
							rotulo="Nível"
							valor={q.get("nivel") ?? "todos"}
							aoMudar={(v) => filtro("nivel", v)}
							opcoes={[
								{ valor: "todos", rotulo: "Todos" },
								{ valor: "alto", rotulo: "Alto" },
								{ valor: "medio", rotulo: "Médio" },
								{ valor: "baixo", rotulo: "Baixo" },
								{ valor: "nao_avaliado", rotulo: "Não avaliado" },
							]}
						/>
						<Filtro
							rotulo="Estado"
							valor={q.get("estado") ?? "todos"}
							aoMudar={(v) => filtro("estado", v)}
							opcoes={[{ valor: "todos", rotulo: "Todos" }, ...Object.entries(estados).map(([valor, rotulo]) => ({ valor, rotulo }))]}
						/>
						<Filtro
							rotulo="Período"
							valor={q.get("periodo") ?? "30"}
							aoMudar={(v) => filtro("periodo", v)}
							opcoes={[
								{ valor: "7", rotulo: "Últimos 7 dias" },
								{ valor: "30", rotulo: "Últimos 30 dias" },
								{ valor: "90", rotulo: "Últimos 90 dias" },
								{ valor: "todos", rotulo: "Todo o histórico" },
							]}
						/>
					</div>
					{erroRepos && <p role="alert">{erroRepos}</p>}
					<Resultados
						key={`${q.toString()}:${atualizacao}`}
						id={id}
						q={q}
						aoPaginar={(pagina, snapshot) =>
							setQ(() => {
								const nova = new URLSearchParams(window.location.search);
								nova.set("installation_id", id);
								nova.set("pagina", String(pagina));
								nova.set("snapshot", snapshot);
								return nova;
							})
						}
						aoOrdenar={() => filtro("ordem", q.get("ordem") === "antigas" ? "recentes" : "antigas")}
					/>
				</>
			)}
		</>
	);
}
function Filtro({
	rotulo,
	valor,
	aoMudar,
	opcoes,
}: {
	rotulo: string;
	valor: string;
	aoMudar: (v: string) => void;
	opcoes: { valor: string; rotulo: string }[];
}) {
	return (
		<div className="campo">
			<span className="rotulo-campo">{rotulo}</span>
			<Selecao rotulo={rotulo} valor={valor} aoMudar={aoMudar} opcoes={opcoes} />
		</div>
	);
}
function Resultados({
	id,
	q,
	aoPaginar,
	aoOrdenar,
}: {
	id: string;
	q: URLSearchParams;
	aoPaginar: (pagina: number, snapshot: string) => void;
	aoOrdenar: () => void;
}) {
	const navegar = useNavigate();
	const [dados, setDados] = useState<PaginaAnalises>();
	const [erro, setErro] = useState("");
	const [tentativa, setTentativa] = useState(0);
	const chave = q.toString();
	// biome-ignore lint/correctness/useExhaustiveDependencies: chave representa os filtros e tentativa permite repetição explícita.
	useEffect(() => {
		const c = new AbortController();
		setDados(undefined);
		setErro("");
		try {
			listarAnalises(consultaDaLista(new URLSearchParams(chave), id), c.signal)
				.then((v) => {
					if (!c.signal.aborted) setDados(v);
				})
				.catch((e) => {
					if (!c.signal.aborted) setErro(erroDeAnalise(e));
				});
		} catch (e) {
			setErro(e instanceof Error ? e.message : "Filtros inválidos.");
		}
		return () => c.abort();
	}, [chave, id, tentativa]);
	if (erro)
		return (
			<PainelDeEstado titulo="Não foi possível consultar análises" erro>
				<p>{erro}</p>
				<Botao onClick={() => setTentativa((v) => v + 1)}>Tentar novamente</Botao>
			</PainelDeEstado>
		);
	if (!dados) return <p role="status">Carregando análises…</p>;
	const inicio = dados.itens.length ? (dados.pagina - 1) * dados.limite + 1 : 0;
	const voltar = new URLSearchParams(q);
	voltar.set("installation_id", id);
	return (
		<section className="resultados-analises">
			<h2>Resultados</h2>
			{dados.itens.length === 0 ? (
				<PainelDeEstado titulo="Nenhuma análise encontrada">
					<p>Não há resultados nesta página para os filtros e o escopo selecionados.</p>
				</PainelDeEstado>
			) : (
				// biome-ignore lint/a11y/noNoninteractiveTabindex: a região rolável precisa receber foco para rolagem por teclado.
				<section className="tabela-rolavel" aria-label="Resultados de análises, tabela rolável" tabIndex={0}>
					<Tabela legenda="Análises autorizadas">
						<CabecalhoDaTabela>
							<Coluna>Repositório / PR</Coluna>
							<Coluna>Risco</Coluna>
							<Coluna>Estado</Coluna>
							<ColunaOrdenavel sentido={q.get("ordem") === "antigas" ? "crescente" : "decrescente"} aoOrdenar={aoOrdenar}>
								Recebida em
							</ColunaOrdenavel>
							<Coluna>Commit</Coluna>
						</CabecalhoDaTabela>
						<CorpoDaTabela>
							{dados.itens.map((a) => (
								<LinhaDaTabela key={a.id}>
									<Celula>
										<Link
											to={`/analises/${a.id}?${new URLSearchParams({ installation_id: id, retorno: voltar.toString() })}`}
											onClick={(e) => {
												if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
												e.preventDefault();
												const retorno = new URLSearchParams(window.location.search);
												retorno.set("installation_id", id);
												navegar(`/analises/${a.id}?${new URLSearchParams({ installation_id: id, retorno: retorno.toString() })}`);
											}}
										>
											{a.repositorio} #{a.numero_pr}
										</Link>
									</Celula>
									<Celula>
										<Risco nivel={a.nivel} />
									</Celula>
									<Celula>
										<Estado estado={a.estado} />
									</Celula>
									<Celula>{dataUTC(a.recebido_em)}</Celula>
									<Celula>
										<code title={a.sha_head}>{a.sha_head.slice(0, 7)}</code>
									</Celula>
								</LinhaDaTabela>
							))}
						</CorpoDaTabela>
					</Tabela>
				</section>
			)}
			<div className="paginacao-analises">
				<p className="apoio">
					{inicio} a {inicio ? inicio + dados.itens.length - 1 : 0} de {dados.total} análises · página {dados.pagina}
				</p>
				<Botao disabled={dados.pagina <= 1} onClick={() => aoPaginar(dados.pagina - 1, dados.snapshot)}>
					Anterior
				</Botao>
				<Botao disabled={!dados.tem_mais || dados.pagina >= 201} onClick={() => aoPaginar(dados.pagina + 1, dados.snapshot)}>
					Próxima
				</Botao>
			</div>
			<p className="apoio">
				Recorte de criação: {dataUTC(dados.snapshot)}. Alterações de estado podem mudar os resultados; atualize para iniciar nova consulta.
			</p>
		</section>
	);
}
