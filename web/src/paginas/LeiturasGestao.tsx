import { useEffect, useState } from "react";
import { Link } from "react-router";
import { useSessao } from "@/app/sessao";
import { Risco } from "@/componentes/Analise";
import { Ativo } from "@/componentes/Ativo";
import {
	opcoesPeriodo,
	PaginaGestao,
	SecaoGestao,
	SeletorGestao,
	TabelaGestao,
	useFiltrosGestao,
	useGestao,
	useOpcoesRepos,
} from "@/componentes/Gestao";
import { Botao, Modal, PainelDeEstado } from "@/componentes/Primitivos";
import { erroDeAnalise, listarRepositorios } from "@/servicos/analises";
import { Protegida } from "./Acesso";

import "@/estilos/analises.css";

function periodoValido(v: string | null) {
	return opcoesPeriodo.some((p) => p.valor === v) ? (v as string) : "30";
}
export function Regras() {
	const todosRepos = [{ valor: "todos", rotulo: "Todos autorizados" }, ...useOpcoesRepos()];
	const dados = useGestao();
	const [detalhes, setDetalhes] = useState(false);
	const { q, atualizar } = useFiltrosGestao();
	const repo = q.get("repositorio") ?? "";
	const periodo = periodoValido(q.get("periodo"));
	const regras = dados.regras;
	return (
		<PaginaGestao titulo="Regras do gate" subtitulo="Regras determinísticas em vigor · consulta somente leitura" voltar>
			<div className="regras-mobile">
				{["falha_sandbox", "dependencia_nova", "area_sensivel", "muitos_arquivos"].map((id) => {
					const r = regras.find((r) => r.id === id);
					return (
						r && (
							<SecaoGestao key={id} titulo={id} caixa>
								<p>
									{
										{
											falha_sandbox: "Testes ou compilação falharam",
											dependencia_nova: "Dependência nova identificada",
											area_sensivel: "Alteração em área sensível",
											muitos_arquivos: "Limite de arquivos excedido",
										}[id]
									}
								</p>
							</SecaoGestao>
						)
					);
				})}
			</div>
			<Botao className="alternar-regras" aria-expanded={detalhes} onClick={() => setDetalhes((v) => !v)}>
				Estatísticas e configuração
			</Botao>
			<div className={`regras-detalhadas${detalhes ? " aberto" : ""}`}>
				<div className="filtros-gestao filtros-regras">
					<SeletorGestao rotulo="Período" valor={periodo} aoMudar={(v) => atualizar("periodo", v)} opcoes={opcoesPeriodo} />
					<SeletorGestao
						rotulo="Repositório"
						valor={repo || "todos"}
						aoMudar={(v) => atualizar("repositorio", v === "todos" ? "" : v)}
						opcoes={todosRepos}
					/>
					<Link className="botao botao-primario" to={`/configuracao${repo ? `?repositorio=${encodeURIComponent(repo)}` : ""}`}>
						<Ativo nome="settings-primario" />
						Configurar regras
					</Link>
				</div>
				<SecaoGestao
					titulo="Regras e estatísticas por regra"
					ajuda="As regras são determinísticas. A tabela mostra quantas vezes cada motivo apareceu e quantas revisões humanas aprovaram apesar do risco alto."
				>
					<TabelaGestao
						legenda="Regras e estatísticas por regra"
						colunas={["Identificador / descrição", "Risco", "Acionamentos", "Aprovação apesar do alto"]}
						larguras={["446px", "96px", "176px", "298px"]}
						linhas={regras.map((r) => ({
							id: r.id,
							celulas: [
								<span className="mono" key="regra">
									{r.id} · {r.descricao}
								</span>,
								<Risco key="nivel" nivel={r.nivel} />,
								`${r.acionamentos} no período`,
								r.nivel === "medio"
									? "Não se aplica"
									: `${r.aprovadas} de ${r.acionamentos} · ${r.acionamentos ? Math.round((r.aprovadas / r.acionamentos) * 100) : 0}%`,
							],
						}))}
					/>
				</SecaoGestao>
				<SecaoGestao
					titulo="Calibração por histórico"
					ajuda="A calibração só suaviza risco com evidência suficiente e nunca altera uma análise já registrada."
				>
					<p className="contexto-tabela">Como o histórico influenciou as análises</p>
					<TabelaGestao
						legenda="Calibração por histórico"
						colunas={["Análise", "Risco inicial → final", "Resultado"]}
						linhas={dados.calibracoes.map((c) => ({
							id: c.id,
							celulas: [
								<Modal
									key="analise"
									titulo={c.nome}
									descricao="Resultado da calibração"
									gatilho={
										<button type="button" className="link-fila">
											{c.nome}
										</button>
									}
								>
									<p>{c.resultado}</p>
								</Modal>,
								c.risco,
								<span className="apoio" key="resultado">
									{c.resultado}
								</span>,
							],
						}))}
					/>
				</SecaoGestao>
			</div>
		</PaginaGestao>
	);
}
export function Auditoria() {
	const todosRepos = [{ valor: "todos", rotulo: "Todos autorizados" }, ...useOpcoesRepos()];
	const dados = useGestao();
	const { q, atualizar } = useFiltrosGestao();
	const repo = q.get("repositorio") ?? "";
	const periodo = periodoValido(q.get("periodo"));
	const decisao = q.get("resultado") ?? "";
	const origem = q.get("origem") ?? "";
	const recusadas = q.get("recusadas") === "1";
	const registros = dados.auditoria;
	return (
		<PaginaGestao titulo="Auditoria de ações" subtitulo="Tentativas aceitas, recusadas e falhas · somente leitura">
			<div className="filtros-gestao filtros-auditoria">
				<SeletorGestao
					rotulo="Resultado"
					valor={decisao || "todos"}
					aoMudar={(v) => atualizar("resultado", v === "todos" ? "" : v)}
					opcoes={[
						{ valor: "todos", rotulo: "Todas as tentativas" },
						...["Aceita", "Recusada", "Falhou"].map((v) => ({ valor: v, rotulo: v })),
					]}
				/>
				<SeletorGestao
					rotulo="Origem"
					valor={origem || "todos"}
					aoMudar={(v) => atualizar("origem", v === "todos" ? "" : v)}
					opcoes={[
						{ valor: "todos", rotulo: "Slack e painel" },
						{ valor: "Slack", rotulo: "Slack" },
						{ valor: "Painel", rotulo: "Painel" },
					]}
				/>
				<SeletorGestao
					rotulo="Repositório"
					valor={repo || "todos"}
					aoMudar={(v) => atualizar("repositorio", v === "todos" ? "" : v)}
					opcoes={todosRepos}
				/>
				<SeletorGestao rotulo="Período" valor={periodo} aoMudar={(v) => atualizar("periodo", v)} opcoes={opcoesPeriodo} />
			</div>
			<SecaoGestao
				titulo="Histórico de ações"
				ajuda="O histórico distingue tentativa aceita, recusa e falha. Aceitar um pedido não comprova merge."
				extra={
					<label className="checkbox-gestao">
						<input type="checkbox" checked={recusadas} onChange={(e) => atualizar("recusadas", e.target.checked ? "1" : "")} />
						Somente recusadas
					</label>
				}
			>
				<TabelaGestao
					legenda="Histórico de ações"
					colunas={["Quando / pessoa", "Ação / PR", "Origem", "Decisão", "Motivo"]}
					larguras={["226px", "206px", "106px", "126px", "328px"]}
					linhas={registros.map((a) => ({
						id: a.id,
						celulas: [
							`${new Date(a.data).toLocaleString("pt-BR", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", timeZone: "UTC" })} · ${a.pessoa}`,
							`${a.acao} · #${a.pr}`,
							a.origem,
							a.decisao,
							a.motivo,
						],
					}))}
				/>
			</SecaoGestao>
		</PaginaGestao>
	);
}
export function Repositorios() {
	const [consulta, setConsulta] = useState(0);
	const { sessao } = useSessao();
	const { q } = useFiltrosGestao();
	const instalacao = q.get("installation_id") ?? (sessao?.autenticado ? String(sessao.instalacoes[0]?.id ?? "") : "");
	return (
		<>
			{!sessao?.autenticado && <h1>Repositórios e instalação</h1>}
			<Protegida>
				{sessao?.autenticado &&
					(sessao.instalacoes.length ? (
						<ConteudoRepositorios
							key={JSON.stringify([sessao.usuario.id, instalacao, consulta])}
							instalacao={instalacao}
							aoAtualizar={() => setConsulta((n) => n + 1)}
						/>
					) : (
						<PainelDeEstado titulo="Nenhuma instalação autorizada">
							<p>Confira suas instalações no GitHub.</p>
						</PainelDeEstado>
					))}
			</Protegida>
		</>
	);
}
function ConteudoRepositorios({ instalacao, aoAtualizar }: { instalacao: string; aoAtualizar: () => void }) {
	const { sessao } = useSessao();
	const { atualizar } = useFiltrosGestao();
	const [dados, setDados] = useState<string[]>();
	const [erro, setErro] = useState("");
	const autorizada = sessao?.autenticado && sessao.instalacoes.some((i) => String(i.id) === instalacao);
	useEffect(() => {
		setDados(undefined);
		setErro("");
		if (!autorizada) {
			setErro("Instalação não autorizada.");
			return;
		}
		const c = new AbortController();
		void listarRepositorios(instalacao, c.signal)
			.then((v) => {
				if (!c.signal.aborted) setDados(v);
			})
			.catch((e) => {
				if (!c.signal.aborted) setErro(erroDeAnalise(e));
			});
		return () => c.abort();
	}, [instalacao, autorizada]);

	return (
		<PaginaGestao titulo="Repositórios e instalação" subtitulo={`${dados?.length ?? "—"} repositórios conectados`}>
			{sessao?.autenticado && (
				<SeletorGestao
					rotulo="Instalação"
					valor={instalacao ?? ""}
					aoMudar={(v) => atualizar("installation_id", v)}
					opcoes={sessao.instalacoes.map((i) => ({ valor: String(i.id), rotulo: i.conta }))}
				/>
			)}
			{erro ? (
				<PainelDeEstado titulo="Consulta de repositórios indisponível" erro>
					<p>{erro}</p>
					<Botao onClick={aoAtualizar}>Tentar novamente</Botao>
				</PainelDeEstado>
			) : !dados ? (
				<p role="status">Consultando repositórios…</p>
			) : (
				<>
					<SecaoGestao
						titulo="Saúde por repositório"
						ajuda="Sem Check obrigatório na proteção de branch, a classificação de risco não impede o merge. Falta de permissão significa estado não confirmado."
					>
						<TabelaGestao
							legenda="Saúde por repositório"
							colunas={["Repositório", "Check obrigatório", "Última análise", "Permissões"]}
							larguras={["276px", "226px", "196px", "318px"]}
							linhas={dados.map((nome) => ({
								id: nome,
								celulas: [
									<Link key="repo" to={`/analises?${new URLSearchParams({ installation_id: instalacao ?? "", repositorio: nome })}`}>
										{nome}
									</Link>,
									"Não confirmado",
									"Não disponível",
									"Não confirmadas",
								],
							}))}
						/>
						<div className="acoes-gestao">
							<Botao variante="primario" onClick={aoAtualizar}>
								Atualizar repositórios
							</Botao>
						</div>
					</SecaoGestao>
					<p className="apoio">Proteção de branch, última análise e permissões detalhadas indisponíveis.</p>
				</>
			)}
		</PaginaGestao>
	);
}
