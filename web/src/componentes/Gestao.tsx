import type { ReactNode } from "react";
import { Link, useSearchParams } from "react-router";
import { repositoriosDemonstracao } from "@/servicos/gestao";
import { Ativo } from "./Ativo";
import { Botao, Modal } from "./Primitivos";
import { Selecao } from "./Selecao";
import { CabecalhoDaTabela, Celula, Coluna, CorpoDaTabela, LinhaDaTabela, Tabela } from "./Tabela";
import "@/estilos/gestao.css";

export function PaginaGestao({
	titulo,
	subtitulo,
	children,
	simulado = true,
	voltar = false,
}: {
	titulo: string;
	subtitulo: string;
	children: ReactNode;
	simulado?: boolean;
	voltar?: boolean;
}) {
	return (
		<div className={`pagina pagina-gestao${voltar ? " pagina-regras" : ""}`}>
			{voltar && (
				<Link className="voltar-regras" to="/">
					<Ativo nome="voltar" />
					Voltar à Visão geral
				</Link>
			)}
			<div className="titulo-contexto">
				<h1>{titulo}</h1>
				<p className="apoio">{subtitulo}</p>
			</div>
			{children}
			{simulado && <p className="apoio nota-simulacao">Demonstração · dados e alterações simulados nesta sessão da página.</p>}
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
				{ajuda && (
					<Modal
						titulo={titulo}
						descricao={ajuda}
						gatilho={
							<Botao variante="discreto" className="ajuda-gestao" aria-label={`Ajuda: ${titulo}`}>
								<Ativo nome="ajuda" />
							</Botao>
						}
					>
						<p>Esta prévia usa dados simulados. As operações não alteram a instalação real.</p>
					</Modal>
				)}
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
							<span style={{ display: "block", minWidth: larguras?.[i] }}>{c}</span>
						</Coluna>
					))}
				</CabecalhoDaTabela>
				<CorpoDaTabela>
					{linhas.map((l) => (
						<LinhaDaTabela key={l.id}>
							{l.celulas.map((c, i) => (
								<Celula key={colunas[i]}>{c}</Celula>
							))}
						</LinhaDaTabela>
					))}
				</CorpoDaTabela>
			</Tabela>
			{!linhas.length && <p className="vazio-gestao">Nenhum registro encontrado para os filtros selecionados.</p>}
		</div>
	);
}
export function SeletorGestao({
	rotulo,
	valor,
	aoMudar,
	opcoes,
	className = "",
}: {
	rotulo: string;
	valor: string;
	aoMudar: (valor: string) => void;
	opcoes: { valor: string; rotulo: string }[];
	className?: string;
}) {
	return (
		<div className={`campo seletor-gestao ${className}`}>
			<span className="rotulo-campo">{rotulo}</span>
			<Selecao rotulo={rotulo} valor={valor} aoMudar={aoMudar} opcoes={opcoes} />
		</div>
	);
}
export const opcoesRepos = repositoriosDemonstracao.map((r) => ({ valor: r.nome, rotulo: r.nome }));
export const opcoesPeriodo = [
	{ valor: "7", rotulo: "Últimos 7 dias" },
	{ valor: "30", rotulo: "Últimos 30 dias" },
	{ valor: "90", rotulo: "Últimos 90 dias" },
];
export function useFiltrosGestao() {
	const [q, setQ] = useSearchParams();
	const atualizar = (nome: string, valor: string) =>
		setQ(
			(anterior) => {
				const novo = new URLSearchParams(anterior);
				if (valor) novo.set(nome, valor);
				else novo.delete(nome);
				return novo;
			},
			{ replace: true },
		);
	return { q, atualizar };
}
