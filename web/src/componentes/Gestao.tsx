import type { ComponentProps, ReactNode } from "react";
import { createContext, useContext } from "react";
import { Link, useSearchParams } from "react-router";
import { comInstalacao } from "@/app/rotas";
import type { Consulta } from "@/servicos/fonte";
import type { Gestao } from "@/servicos/gestao";
import { Ativo } from "./Ativo";
import { ConsultaPrivada } from "./ConsultaPrivada";
import { Ajuda, Botao } from "./Primitivos";
import { Selecao } from "./Selecao";
import { CabecalhoDaTabela, Celula, Coluna, CorpoDaTabela, LinhaDaTabela, Tabela } from "./Tabela";
import "@/estilos/gestao.css";

export function PaginaGestao({
	titulo,
	subtitulo,
	children,
	voltar = false,
}: {
	titulo: string;
	subtitulo: string;
	children: ReactNode;
	voltar?: boolean;
}) {
	const simulada = !!useContext(Contexto) && __MOCKS__;
	const [q] = useSearchParams();
	const instalacao = q.get("installation_id");
	return (
		<div className={`pagina pagina-gestao${voltar ? " pagina-regras" : ""}`}>
			{voltar && (
				<Link className="voltar-regras" to={comInstalacao("/", instalacao)}>
					<Ativo nome="voltar" />
					Voltar à Visão geral
				</Link>
			)}
			<div className="titulo-contexto">
				<h1>{titulo}</h1>
				<p className="apoio">
					{simulada && "Dados simulados · "}
					{subtitulo}
				</p>
			</div>
			{children}
		</div>
	);
}
export function SecaoGestao({
	titulo,
	ajuda,
	children,
	caixa = false,
	extra,
}: {
	titulo: string;
	ajuda?: string;
	children: ReactNode;
	caixa?: boolean;
	extra?: ReactNode;
}) {
	return (
		<section className={`secao-gestao${caixa ? " secao-gestao-caixa" : ""}`}>
			<header>
				<h2>{titulo}</h2>
				{ajuda && <Ajuda titulo={titulo} descricao={ajuda} className="ajuda-gestao" />}
				{extra && <div className="extra-secao">{extra}</div>}
			</header>
			<div className="corpo-gestao">{children}</div>
		</section>
	);
}
export function TabelaGestao({
	colunas,
	linhas,
	legenda,
	compacta = false,
	larguras,
}: {
	colunas: string[];
	linhas: { id: string; celulas: ReactNode[] }[];
	legenda: string;
	compacta?: boolean;
	larguras?: string[];
}) {
	return (
		<div className={`viewport-gestao${compacta ? " tabela-gestao-compacta" : ""}`}>
			<Tabela legenda={legenda}>
				<CabecalhoDaTabela>
					{colunas.map((c, i) => (
						<Coluna key={c}>
							<span style={{ display: "block", minWidth: larguras?.at(i) }}>{c}</span>
						</Coluna>
					))}
				</CabecalhoDaTabela>
				<CorpoDaTabela>
					{linhas.map((l) => (
						<LinhaDaTabela key={l.id}>
							{l.celulas.map((c, i) => (
								<Celula key={colunas.at(i)}>{c}</Celula>
							))}
						</LinhaDaTabela>
					))}
				</CorpoDaTabela>
			</Tabela>
			{!linhas.length && <p className="vazio-gestao">Nenhum registro encontrado para os filtros selecionados.</p>}
		</div>
	);
}
export function SeletorGestao({ rotulo, valor, aoMudar, opcoes, className = "" }: ComponentProps<typeof Selecao>) {
	return (
		<div className={`campo seletor-gestao ${className}`}>
			<span className="rotulo-campo">{rotulo}</span>
			<Selecao rotulo={rotulo} valor={valor} aoMudar={aoMudar} opcoes={opcoes} />
		</div>
	);
}
const Contexto = createContext<Gestao | null>(null);
export function ConsultaGestao({
	titulo,
	children,
	consultar,
	filtrosDaConsulta,
	reutilizar,
}: {
	titulo: string;
	children: ReactNode;
	consultar: Consulta<Gestao>;
	filtrosDaConsulta?: string[];
	reutilizar?: boolean;
}) {
	return (
		<ConsultaPrivada titulo={titulo} consultar={consultar} filtrosDaConsulta={filtrosDaConsulta} reutilizar={reutilizar}>
			{(dados) => <Contexto value={dados}>{children}</Contexto>}
		</ConsultaPrivada>
	);
}
export function useGestao() {
	const dados = useContext(Contexto);
	if (!dados) throw new Error("Consulta de gestão ausente.");
	return dados;
}
export function useOpcoesRepos() {
	return useGestao().repositorios.map((nome) => ({ valor: nome, rotulo: nome }));
}
export const opcoesPeriodo = [
	{ valor: "7", rotulo: "Últimos 7 dias" },
	{ valor: "30", rotulo: "Últimos 30 dias" },
	{ valor: "90", rotulo: "Últimos 90 dias" },
];
export function opcoesDoPeriodo(valor: string) {
	return opcoesPeriodo.some((p) => p.valor === valor) ? opcoesPeriodo : [...opcoesPeriodo, { valor, rotulo: `Últimos ${valor} dias` }];
}
export function useFiltrosGestao() {
	const [q, setQ] = useSearchParams();
	const atualizar = (nome: string, valor: string) =>
		setQ(
			(anterior) => {
				const novo = new URLSearchParams(anterior);
				novo.delete("pagina");
				novo.delete("snapshot");
				if (valor) novo.set(nome, valor);
				else novo.delete(nome);
				return novo;
			},
			{ replace: true },
		);
	return { q, atualizar };
}

export function PaginacaoPainel({ pagina, limite, total, snapshot }: { pagina: number; limite: number; total: number; snapshot: string }) {
	const [, setQ] = useSearchParams();
	function paginar(p: number) {
		setQ((q) => {
			const v = new URLSearchParams(q);
			v.set("pagina", String(p));
			v.set("snapshot", snapshot);
			return v;
		});
	}
	return (
		<div className="controles">
			<span className="apoio" role="status">
				{total} registros · página {pagina}
			</span>
			<Botao disabled={pagina <= 1} onClick={() => paginar(pagina - 1)}>
				Anterior
			</Botao>
			<Botao disabled={pagina >= 201 || pagina * limite >= total} onClick={() => paginar(pagina + 1)}>
				Próxima
			</Botao>
		</div>
	);
}
