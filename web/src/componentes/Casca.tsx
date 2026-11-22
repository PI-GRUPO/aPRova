import { useEffect, useRef, useState } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router";
import { comInstalacao, secoes } from "@/app/rotas";
import { useSessao } from "@/app/sessao";
import { useTema } from "@/app/tema";
import { Sair } from "@/paginas/Acesso";
import { Ativo } from "./Ativo";
import { Botao, Modal } from "./Primitivos";

function Tema() {
	const { tema, alternar } = useTema();
	const proximo = tema === "escuro" ? "claro" : "escuro";
	return (
		<Botao
			className="botao-tema"
			variante={tema === "escuro" ? "primario" : "secundario"}
			onClick={alternar}
			aria-label={`Usar tema ${proximo}`}
		>
			<Ativo nome="tema" />
			Tema {proximo}
		</Botao>
	);
}
export function Casca() {
	const { sessao } = useSessao();
	const { pathname, search } = useLocation();
	const instalacao = new URLSearchParams(search).get("installation_id");
	function destino(caminho: string) {
		return comInstalacao(caminho, instalacao);
	}
	const [menu, definirMenu] = useState(false);
	const anterior = useRef(pathname);
	useEffect(() => {
		if (anterior.current !== pathname) document.getElementById("conteudo")?.focus({ preventScroll: true });
		anterior.current = pathname;
	}, [pathname]);
	function fecharMenu() {
		definirMenu(false);
	}
	const links = (de = 0, ate: number = secoes.length) =>
		secoes.slice(de, ate).map((s) => (
			<NavLink key={s.caminho} to={destino(s.caminho)} end={s.caminho === "/"} className="item-nav" onClick={fecharMenu}>
				<Ativo nome={s.icone} ativo={pathname === s.caminho || (s.caminho !== "/" && pathname.startsWith(`${s.caminho}/`))} />
				<span>{s.titulo}</span>
			</NavLink>
		));
	return (
		<div className="aplicacao">
			<a className="pular" href="#conteudo">
				Pular para o conteúdo
			</a>
			<aside className="lateral">
				<Link to={destino("/")} aria-label="aPRova — Visão geral">
					<Ativo nome="marca" />
				</Link>
				<nav aria-label="Principal" className="menu-desktop">
					{links(0, 6)}
					<p className="grupo-nav">Gestão</p>
					{links(6)}
				</nav>
				<footer>
					<p className="contexto-conta caminho">{sessao?.autenticado ? `GitHub · @${sessao.usuario.login}` : "Nenhuma sessão ativa"}</p>
					<p className="apoio">
						{sessao?.autenticado ? `${sessao.instalacoes.length} instalações autorizadas` : "Entre para consultar seus repositórios."}
					</p>
					<Tema />
					{sessao?.autenticado ? (
						<Sair />
					) : (
						<Link className="botao botao-secundario" to="/entrar">
							Entrar com GitHub
						</Link>
					)}
				</footer>
			</aside>
			<header className="cabecalho-mobile">
				<div className="marca-mobile">
					<Modal
						aberto={menu}
						aoMudar={definirMenu}
						gatilho={
							<button type="button" className="marca-menu" aria-label="Menu completo">
								<Ativo nome="marca" />
							</button>
						}
						titulo="Navegação"
						descricao="Acesse as telas do dashboard."
					>
						<nav className="menu-mobile" aria-label="Todas as telas">
							{links()}
						</nav>
						{sessao?.autenticado ? (
							<>
								<p className="caminho">@{sessao.usuario.login}</p>
								<Sair />
							</>
						) : (
							<Link to="/entrar" onClick={fecharMenu}>
								Entrar com GitHub
							</Link>
						)}
						<Botao onClick={fecharMenu}>Fechar</Botao>
					</Modal>
					<Tema />
				</div>
				<nav aria-label="Principal no celular">
					{links(0, 3)}
					<NavLink to={destino("/regras")} className="item-nav">
						Regras
					</NavLink>
				</nav>
			</header>
			<main id="conteudo" tabIndex={-1}>
				<Outlet />
			</main>
		</div>
	);
}
