import { Fragment, useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useMobile } from "@/app/responsivo";
import { useSessao } from "@/app/sessao";
import { dataUTC, Estado, Risco } from "@/componentes/Analise";
import { useLeitura } from "@/componentes/ConsultaPrivada";
import { ControlesDaLista } from "@/componentes/ControlesDaLista";
import { Botao, Campo, PainelDeEstado } from "@/componentes/Primitivos";
import { Selecao } from "@/componentes/Selecao";
import { CabecalhoDaTabela, Celula, Coluna, ColunaOrdenavel, CorpoDaTabela, LinhaDaTabela, Tabela } from "@/componentes/Tabela";
import { consultarRepos, estados, listarAnalises, type PaginaAnalises } from "@/servicos/analises";
import type { Consulta } from "@/servicos/fonte";
import { destinoAnalise } from "@/servicos/visao-geral";
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
				<p className="apoio contexto-analises-desktop">Histórico autorizado · filtros aplicados a toda a consulta · horários em UTC</p>
				<p className="apoio contexto-analises-mobile">Análises da instalação</p>
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
	else return null;
	c.set("inicio", inicio.toISOString().slice(0, 10));
	c.set("fim", fim.toISOString().slice(0, 10));
	return c;
}
const consultarLista: Consulta<PaginaAnalises> = (_escopo, consulta, signal) => listarAnalises(consulta, signal);

function ListaDaInstalacao({ id }: { id: string }) {
	const mobile = useMobile();
	const { sessao } = useSessao();
	const [q, setQ] = useSearchParams();
	const [atualizacao, setAtualizacao] = useState(0);
	const [busca, setBusca] = useState(q.get("numero_pr") ?? "");
	const [erroBusca, setErroBusca] = useState("");
	const numeroPR = q.get("numero_pr") ?? "";
	useEffect(() => {
		setBusca(numeroPR);
	}, [numeroPR]);
	const autorizado = !!sessao?.autenticado && sessao.instalacoes.some((i) => String(i.id) === id);
	const { estado: catalogo } = useLeitura(consultarRepos, id, "", { reutilizar: true, habilitada: autorizado });
	const repos = catalogo && "dados" in catalogo ? catalogo.dados : undefined;
	const erroRepos = catalogo && "erro" in catalogo ? catalogo.erro : "";
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
	const controles = (
		<ControlesDaLista key="controles" mobile={mobile}>
			<div className="controles">
				<div className="campo">
					<span className="rotulo-campo">Instalação</span>
					<Selecao
						rotulo="Instalação"
						valor={id}
						aoMudar={(v) => {
							filtro("installation_id", v);
						}}
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
			{autorizado && (
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
							onChange={(e) => {
								setBusca(e.target.value);
							}}
							erro={erroBusca}
						/>
						<Botao type="submit">Buscar</Botao>
					</form>
					<Filtro
						rotulo="Repositório"
						valor={q.get("repositorio") ?? "todos"}
						aoMudar={(v) => {
							filtro("repositorio", v === "todos" ? "" : v);
						}}
						opcoes={[{ valor: "todos", rotulo: "Todos" }, ...(repos ?? []).map((r) => ({ valor: r, rotulo: r }))]}
					/>
					<Filtro
						rotulo="Nível"
						valor={q.get("nivel") ?? "todos"}
						aoMudar={(v) => {
							filtro("nivel", v);
						}}
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
						aoMudar={(v) => {
							filtro("estado", v);
						}}
						opcoes={[{ valor: "todos", rotulo: "Todos" }, ...[...estados].map(([valor, rotulo]) => ({ valor, rotulo }))]}
					/>
					<Filtro
						rotulo="Período"
						valor={q.get("periodo") ?? "30"}
						aoMudar={(v) => {
							filtro("periodo", v);
						}}
						opcoes={[
							{ valor: "7", rotulo: "Últimos 7 dias" },
							{ valor: "30", rotulo: "Últimos 30 dias" },
							{ valor: "90", rotulo: "Últimos 90 dias" },
							{ valor: "todos", rotulo: "Todo o histórico" },
						]}
					/>
				</div>
			)}
		</ControlesDaLista>
	);
	const resultados = (
		<Fragment key="resultados">
			{erroRepos && <p role="alert">{erroRepos}</p>}
			{!autorizado ? (
				<PainelDeEstado titulo="Instalação não autorizada" erro>
					<p>Selecione uma instalação disponível para sua conta.</p>
				</PainelDeEstado>
			) : (
				<Resultados
					key={`${q.toString()}:${atualizacao}`}
					id={id}
					q={q}
					aoPaginar={(pagina, snapshot) => {
						setQ(() => {
							const nova = new URLSearchParams(window.location.search);
							nova.set("installation_id", id);
							nova.set("pagina", String(pagina));
							nova.set("snapshot", snapshot);
							return nova;
						});
					}}
					aoOrdenar={() => {
						filtro("ordem", q.get("ordem") === "antigas" ? "recentes" : "antigas");
					}}
				/>
			)}
		</Fragment>
	);
	return <div className="lista-analises">{mobile ? [resultados, controles] : [controles, resultados]}</div>;
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
	const consulta = consultaDaLista(q, id);
	const { estado, repetir } = useLeitura(consultarLista, id, consulta?.toString() ?? "", { reutilizar: true, habilitada: !!consulta });
	const dados = estado && "dados" in estado ? estado.dados : undefined;
	const erro = consulta ? (estado && "erro" in estado ? estado.erro : "") : "Período inválido.";
	if (erro)
		return (
			<PainelDeEstado titulo="Não foi possível consultar análises" erro>
				<p>{erro}</p>
				<Botao onClick={repetir}>Tentar novamente</Botao>
			</PainelDeEstado>
		);
	if (!dados) return <p role="status">Carregando análises…</p>;
	const inicio = dados.itens.length ? (dados.pagina - 1) * dados.limite + 1 : 0;
	const voltar = new URLSearchParams(q);
	voltar.set("installation_id", id);
	return (
		<section className="resultados-analises">
			<h2 className="titulo-resultados-desktop">Resultados</h2>
			{dados.itens.length === 0 ? (
				<PainelDeEstado titulo="Nenhuma análise encontrada">
					<p>Não há resultados nesta página para os filtros e o escopo selecionados.</p>
				</PainelDeEstado>
			) : (
				<>
					<ul className="cards-analises" aria-label="Análises autorizadas">
						{dados.itens.map((a) => (
							<li key={a.id} className="card-analise">
								<h2>
									{a.repositorio} #{a.numero_pr}
								</h2>
								<div>
									<p className="apoio">{estados.get(a.estado) ?? a.estado}</p>
									<p className="sr-only">
										<Risco nivel={a.nivel} /> · {dataUTC(a.recebido_em)} · commit {a.sha_head}
									</p>
									<Link
										className="botao botao-secundario"
										aria-label={`Abrir análise ${a.repositorio} #${a.numero_pr}`}
										to={destinoAnalise(a.id, "api", id, voltar)}
									>
										Abrir análise
									</Link>
								</div>
							</li>
						))}
					</ul>
					{/* biome-ignore lint/a11y/noNoninteractiveTabindex: a região rolável precisa receber foco para rolagem por teclado. */}
					<section className="tabela-rolavel tabela-analises-desktop" aria-label="Resultados de análises, tabela rolável" tabIndex={0}>
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
												to={destinoAnalise(a.id, "api", id, voltar)}
												onClick={(e) => {
													if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
													e.preventDefault();
													const retorno = new URLSearchParams(window.location.search);
													retorno.set("installation_id", id);
													navegar(destinoAnalise(a.id, "api", id, retorno));
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
				</>
			)}
			<div className="paginacao-analises">
				<p className="apoio">
					{inicio} a {inicio ? inicio + dados.itens.length - 1 : 0} de {dados.total} análises · página {dados.pagina}
				</p>
				<Botao
					disabled={dados.pagina <= 1}
					onClick={() => {
						aoPaginar(dados.pagina - 1, dados.snapshot);
					}}
				>
					Anterior
				</Botao>
				<Botao
					disabled={!dados.tem_mais || dados.pagina >= 201}
					onClick={() => {
						aoPaginar(dados.pagina + 1, dados.snapshot);
					}}
				>
					Próxima
				</Botao>
			</div>
			<p className="apoio">
				Recorte de criação: {dataUTC(dados.snapshot)}. Alterações de estado podem mudar os resultados; atualize para iniciar nova consulta.
			</p>
		</section>
	);
}
