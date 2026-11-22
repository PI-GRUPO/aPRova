import { Link, useSearchParams } from "react-router";
import { useMobile } from "@/app/responsivo";
import { type Analise, consultarFila, consultarRepos, type PaginaAnalises } from "@/servicos/analises";
import { dataUTC } from "./Analise";
import { ConsultaPrivada, useEscopo } from "./ConsultaPrivada";
import { ControlesDaLista } from "./ControlesDaLista";
import { Botao, Campo } from "./Primitivos";
import { Selecao } from "./Selecao";

const parametros = ["repositorio", "pagina", "snapshot"];
function LinkDaAnalise({ item }: { item: Analise }) {
	const [q] = useSearchParams();
	const { instalacao } = useEscopo("");
	const destino = new URLSearchParams({
		installation_id: instalacao,
		repositorio: item.repositorio,
		origem: "fila",
		retorno: q.toString(),
	});
	return (
		<Link className="link-fila" to={`/analises/${encodeURIComponent(item.id)}?${destino}`}>
			{item.repositorio} #{item.numero_pr}
		</Link>
	);
}

export function FilaDePendencias() {
	const mobile = useMobile();
	const [q, setQ] = useSearchParams();
	function filtrar(nome: string, valor: string) {
		setQ(
			() => {
				const nova = new URLSearchParams(window.location.search);
				if (valor) nova.set(nome, valor);
				else nova.delete(nome);
				if (nome === "repositorio") {
					nova.delete("mapa_pagina");
					nova.delete("mapa_snapshot");
					nova.delete("pagina");
					nova.delete("snapshot");
				}
				return nova;
			},
			{ replace: true },
		);
	}
	return (
		<section className="fila-overview" aria-label="Fila de pendências">
			<h2>Aguardando aprovação · risco alto</h2>
			<p className="apoio">Dados da API · a listagem não informa branch, motivos ou evidências.</p>
			<ControlesDaLista mobile={mobile} titulo="Filtros da fila">
				<div className="filtros-fila">
					<Campo
						type="search"
						rotulo="Buscar na fila"
						placeholder="PR, repositório ou commit nesta página"
						value={q.get("fila") ?? ""}
						onChange={(e) => filtrar("fila", e.target.value)}
					/>
					<ConsultaPrivada titulo="Repositórios" secao consultar={consultarRepos} filtrosDaConsulta={[]} reutilizar>
						{(repos) => (
							<div className="campo">
								<span className="rotulo-campo">Repositório</span>
								<Selecao
									rotulo="Repositório"
									valor={q.get("repositorio") || "todos"}
									aoMudar={(v) => filtrar("repositorio", v === "todos" ? "" : v)}
									opcoes={[{ valor: "todos", rotulo: "Todos os repositórios" }, ...repos.map((r) => ({ valor: r, rotulo: r }))]}
								/>
							</div>
						)}
					</ConsultaPrivada>
				</div>
			</ControlesDaLista>
			<ConsultaPrivada titulo="Pendências" secao consultar={consultarFila} filtrosDaConsulta={parametros} reutilizar>
				{(dados) => <Resultados dados={dados} />}
			</ConsultaPrivada>
		</section>
	);
}

function Resultados({ dados }: { dados: PaginaAnalises }) {
	const [q, setQ] = useSearchParams();
	const busca = (q.get("fila") ?? "").trim().toLocaleLowerCase("pt-BR");
	const itens = dados.itens.filter((i) => `${i.repositorio} #${i.numero_pr} ${i.sha_head}`.toLocaleLowerCase("pt-BR").includes(busca));
	function paginar(pagina: number) {
		setQ(() => {
			const nova = new URLSearchParams(window.location.search);
			nova.set("pagina", String(pagina));
			nova.set("snapshot", dados.snapshot);
			return nova;
		});
	}
	return (
		<>
			<p role="status" className="apoio">
				{itens.length} nesta página · {dados.total} no total · página {dados.pagina}
			</p>
			<div className="tabela-fila-viewport">
				<table className="tabela-fila">
					<thead>
						<tr>
							{["Espera", "Recebida em (UTC)", "Pull request", "Commit", "Autor"].map((t) => (
								<th scope="col" key={t}>
									{t}
								</th>
							))}
						</tr>
					</thead>
					<tbody>
						{itens.map((i) => (
							<tr key={i.id}>
								<td>{i.espera_segundos === undefined ? "Indisponível" : `${Math.floor(i.espera_segundos / 60)} min`}</td>
								<td>{dataUTC(i.recebido_em)}</td>
								<td>
									<LinkDaAnalise item={i} />
								</td>
								<td className="mono">{i.sha_head.slice(0, 7) || "Não informado"}</td>
								<td>{i.autor || "Não informado"}</td>
							</tr>
						))}
					</tbody>
				</table>
			</div>
			<div className="cards-fila">
				{itens.map((i) => (
					<article key={i.id}>
						<LinkDaAnalise item={i} />
						<p className="apoio">Espera {i.espera_segundos === undefined ? "indisponível" : `${Math.floor(i.espera_segundos / 60)} min`}</p>
						<p className="mono">Commit {i.sha_head.slice(0, 7) || "não informado"}</p>
						<p className="apoio">Recebida em {dataUTC(i.recebido_em)}</p>
					</article>
				))}
			</div>
			{!itens.length && <p className="apoio vazio-fila">Nenhuma análise encontrada nesta página com esses filtros.</p>}
			{(dados.pagina > 1 || dados.tem_mais) && (
				<div className="controles">
					<Botao disabled={dados.pagina <= 1} onClick={() => paginar(dados.pagina - 1)}>
						Anterior
					</Botao>
					<Botao disabled={!dados.tem_mais || dados.pagina >= 201} onClick={() => paginar(dados.pagina + 1)}>
						Próxima
					</Botao>
				</div>
			)}
		</>
	);
}
