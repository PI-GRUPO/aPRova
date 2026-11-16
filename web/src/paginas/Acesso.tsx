import { type ReactNode, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { useSessao } from "@/app/sessao";
import { useTema } from "@/app/tema";
import { Ativo } from "@/componentes/Ativo";
import { Botao, PainelDeEstado } from "@/componentes/Primitivos";
import { mensagemDeErro } from "@/servicos/recibos";
import "@/estilos/sessao.css";
export function Protegida({ children }: { children: ReactNode }) {
	const { sessao, erro, carregar } = useSessao();
	if (erro)
		return (
			<PainelDeEstado titulo="Não foi possível confirmar sua sessão" erro>
				<p>{erro}</p>
				<Botao onClick={carregar}>Tentar novamente</Botao>
			</PainelDeEstado>
		);
	if (!sessao) return <p role="status">Confirmando sessão…</p>;
	if (!sessao.autenticado)
		return (
			<PainelDeEstado titulo="Entre para continuar">
				<p>Use sua conta GitHub para consultar os repositórios autorizados.</p>
				<Link to="/entrar">Entrar com GitHub</Link>
			</PainelDeEstado>
		);
	return children;
}
export function Entrar() {
	const { sessao, erro, carregar } = useSessao();
	const [q] = useSearchParams();
	const { tema, alternar } = useTema();
	return (
		<main className="pagina-entrar">
			<section className="cartao-entrar">
				<Ativo nome="marca" />
				<h1>
					Acompanhe decisões.
					<br />
					Confira as evidências.
				</h1>
				<p className="secundario">Entre com sua conta GitHub para acessar os repositórios autorizados.</p>
				{q.has("erro") && <p role="alert">Não foi possível concluir a autenticação. Inicie uma nova tentativa.</p>}
				{erro ? (
					<>
						<p role="alert">{erro}</p>
						<Botao onClick={carregar}>Tentar novamente</Botao>
					</>
				) : !sessao ? (
					<p role="status">Confirmando disponibilidade…</p>
				) : sessao.autenticado ? (
					<Link className="botao botao-primario" to="/">
						Continuar para o dashboard
					</Link>
				) : sessao.login_disponivel ? (
					<a className="botao botao-primario" href="/auth/github">
						<Ativo nome="github" />
						Continuar com GitHub
					</a>
				) : (
					<PainelDeEstado titulo="Login indisponível">
						<p>O acesso com GitHub ainda não está configurado neste ambiente.</p>
					</PainelDeEstado>
				)}
				<p className="apoio">Você autoriza o acesso no GitHub. O aPRova não pede sua senha.</p>
			</section>
			<Botao onClick={alternar}>Usar tema {tema === "escuro" ? "claro" : "escuro"}</Botao>
			<Link to="/recibos">Consultar recibo público</Link>
		</main>
	);
}
export function Sair() {
	const { sair } = useSessao();
	const [erro, setErro] = useState("");
	const [enviando, setEnviando] = useState(false);
	return (
		<>
			<Botao
				disabled={enviando}
				onClick={async () => {
					setEnviando(true);
					setErro("");
					try {
						await sair();
					} catch (e) {
						setErro(mensagemDeErro(e));
					} finally {
						setEnviando(false);
					}
				}}
			>
				Sair
			</Botao>
			{erro && (
				<p role="alert" className="apoio">
					{erro}
				</p>
			)}
		</>
	);
}
export function Conta() {
	const { sessao } = useSessao();
	return (
		<div className="pagina">
			<div className="titulo-contexto">
				<h1>Conta e integrações</h1>
				<p className="apoio">Sua conta e instalações autorizadas.</p>
			</div>
			<Protegida>
				{sessao?.autenticado && (
					<>
						<section className="painel-sessao">
							<h2>Sua identidade GitHub</h2>
							<div>
								<dl className="dados-identidade">
									<div>
										<dt>Conta autenticada</dt>
										<dd>@{sessao.usuario.login}</dd>
									</div>
								</dl>
								<p className="apoio">Sessão até {new Date(sessao.expira_em).toLocaleString("pt-BR", { timeZone: "UTC" })} UTC</p>
								{sessao.instalacoes.length ? (
									sessao.instalacoes.map((i) => (
										<p className="caminho" key={i.id}>
											{i.conta} · Instalação {i.id}
										</p>
									))
								) : (
									<p>Nenhuma instalação autorizada para esta conta.</p>
								)}
							</div>
						</section>
						<section className="painel-sessao">
							<h2>Conectar Slack</h2>
							<div>
								<p>A conexão pessoal com Slack ainda não está disponível.</p>
								<Botao disabled>Conectar minha conta Slack</Botao>
							</div>
						</section>
					</>
				)}
			</Protegida>
		</div>
	);
}
