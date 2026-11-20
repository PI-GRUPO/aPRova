import { Link, useLocation } from "react-router";
import { Ativo } from "@/componentes/Ativo";

export function NaoEncontrada() {
	const { pathname } = useLocation();
	const caminho = pathname.length > 28 ? `${pathname.slice(0, 27)}…` : pathname;
	return (
		<div className="pagina pagina-404">
			<Link className="voltar" to="/">
				<Ativo nome="voltar" />
				Voltar à Visão geral
			</Link>
			<div className="titulo-contexto">
				<h1>Página não encontrada</h1>
				<p className="apoio caminho">{pathname}</p>
			</div>
			<p className="secundario">Este endereço não foi encontrado. Confira o link ou volte à Visão geral.</p>
			<div className="arte-404" aria-hidden="true">
				<div className="arte-404-interna">
					<span className="numero-fundo">404</span>
					<span className="quatro-a">4</span>
					<span className="quatro-b">4</span>
					<span className="zero">0</span>
					<Ativo nome="fio" className="fio" />
					<Ativo nome="vertical" className="vertical" />
					<div className="etiqueta">
						<Ativo nome="etiqueta" />
						<span>{caminho}</span>
					</div>
					<span className="caminho-riscado">{caminho}</span>
					<span className="rotulo-vertical">NÃO ENCONTRADA</span>
					<span className="rotulo-pagina">PÁGINA</span>
					<span className="rotulo-aqui">NENHUMA TELA AQUI</span>
				</div>
			</div>
		</div>
	);
}
