import { useState } from "react";
import { Link } from "react-router";
import { useSessao } from "@/app/sessao";
import { Risco } from "@/componentes/Analise";
import { Ativo } from "@/componentes/Ativo";
import { useLeitura } from "@/componentes/ConsultaPrivada";
import {
	opcoesDoPeriodo,
	PaginacaoPainel,
	PaginaGestao,
	SecaoGestao,
	SeletorGestao,
	TabelaGestao,
	useFiltrosGestao,
	useGestao,
	useOpcoesRepos,
} from "@/componentes/Gestao";
import { Botao, Modal, PainelDeEstado } from "@/componentes/Primitivos";
import { SaudeDosRepositorios } from "@/componentes/SaudeRepositorios";
import { saudeRepositorios } from "@/servicos/analises";
import { Protegida } from "./Acesso";

import "@/estilos/analises.css";

function resumoAprovacoes(r: ReturnType<typeof useGestao>["regras"][number]) {
	if (r.avaliadasAltas === undefined && r.nivel === "medio") return "Não se aplica";
	const total = r.avaliadasAltas ?? r.acionamentos;
	return total > 0 ? `${r.aprovadas} de ${total} · ${Math.round((r.aprovadas * 100) / total)}%` : "Sem avaliações de risco alto";
}

export function Regras() {
	const todosRepos = [{ valor: "todos", rotulo: "Todos autorizados" }, ...useOpcoesRepos()];
	const dados = useGestao();
	const [detalhes, setDetalhes] = useState(false);
	const { q, atualizar } = useFiltrosGestao();
	const repo = q.get("repositorio") ?? "";
	const periodo = q.get("periodo") ?? "30";
	const regras = dados.regras;
	return (
		<PaginaGestao
			titulo="Regras do gate"
			subtitulo={__MOCKS__ ? "Regras determinísticas em vigor · consulta somente leitura" : "Acionamentos registrados pelo gate no período"}
			voltar
		>
			<div className="regras-mobile">
				{(__MOCKS__
					? [
							["falha_sandbox", "Testes ou compilação falharam"],
							["dependencia_nova", "Dependência nova identificada"],
							["area_sensivel", "Alteração em área sensível"],
							["muitos_arquivos", "Limite de arquivos excedido"],
						]
					: regras.map((r) => [r.id, r.descricao, r.nivel])
				).map(([id, descricao, nivel]) => {
					const r = regras.find((r) => r.id === id);
					return (
						r && (
							<SecaoGestao key={id + (nivel ?? "")} titulo={id} caixa>
								<p>{descricao}</p>
							</SecaoGestao>
						)
					);
				})}
			</div>
			<Botao
				className="alternar-regras"
				aria-expanded={detalhes}
				onClick={() => {
					setDetalhes((v) => !v);
				}}
			>
				Estatísticas e configuração
			</Botao>
			<div className={`regras-detalhadas${detalhes ? " aberto" : ""}`}>
				<div className="filtros-gestao filtros-regras">
					<SeletorGestao
						rotulo="Período"
						valor={periodo}
						aoMudar={(v) => {
							atualizar("periodo", v);
						}}
						opcoes={opcoesDoPeriodo(periodo)}
					/>
					<SeletorGestao
						rotulo="Repositório"
						valor={repo || "todos"}
						aoMudar={(v) => {
							atualizar("repositorio", v === "todos" ? "" : v);
						}}
						opcoes={todosRepos}
					/>
					<Link
						className="botao botao-primario"
						to={`/configuracao?${new URLSearchParams([...q].filter(([k]) => ["installation_id", "repositorio"].includes(k)))}`}
					>
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
							id: r.id + r.nivel,
							celulas: [
								<span className="mono" key="regra">
									{r.id} · {r.descricao}
								</span>,
								<Risco key="nivel" nivel={r.nivel} />,
								`${r.acionamentos} no período`,
								resumoAprovacoes(r),
							],
						}))}
					/>
				</SecaoGestao>
				<SecaoGestao
					titulo="Calibração por histórico"
					ajuda="A calibração só suaviza risco com evidência suficiente e nunca altera uma análise já registrada."
				>
					<p className="contexto-tabela">
						{__MOCKS__ ? "Como o histórico influenciou as análises" : "Até 50 avaliações mais recentes no período"}
					</p>
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
	const periodo = q.get("periodo") ?? "30";
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
					aoMudar={(v) => {
						atualizar("resultado", v === "todos" ? "" : v);
					}}
					opcoes={[
						{ valor: "todos", rotulo: "Todas as tentativas" },
						...["Aceita", "Recusada", ...(__MOCKS__ ? ["Falhou"] : ["Falha", "Repetida"])].map((v) => ({ valor: v, rotulo: v })),
					]}
				/>
				<SeletorGestao
					rotulo="Origem"
					valor={origem || "todos"}
					aoMudar={(v) => {
						atualizar("origem", v === "todos" ? "" : v);
					}}
					opcoes={[
						{ valor: "todos", rotulo: "Slack e painel" },
						{ valor: "Slack", rotulo: "Slack" },
						{ valor: "Painel", rotulo: "Painel" },
					]}
				/>
				<SeletorGestao
					rotulo="Repositório"
					valor={repo || "todos"}
					aoMudar={(v) => {
						atualizar("repositorio", v === "todos" ? "" : v);
					}}
					opcoes={todosRepos}
				/>
				<SeletorGestao
					rotulo="Período"
					valor={periodo}
					aoMudar={(v) => {
						atualizar("periodo", v);
					}}
					opcoes={opcoesDoPeriodo(periodo)}
				/>
			</div>
			<SecaoGestao
				titulo="Histórico de ações"
				ajuda="O histórico distingue tentativa aceita, recusa e falha. Aceitar um pedido não comprova merge."
				extra={
					<label className="checkbox-gestao">
						<input
							type="checkbox"
							checked={recusadas}
							onChange={(e) => {
								atualizar("recusadas", e.target.checked ? "1" : "");
							}}
						/>
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
			{dados.paginacao && <PaginacaoPainel {...dados.paginacao} />}
		</PaginaGestao>
	);
}
export function Repositorios() {
	const { sessao, erro } = useSessao();
	const { q } = useFiltrosGestao();
	const instalacao = q.get("installation_id") ?? (sessao?.autenticado ? String(sessao.instalacoes[0]?.id ?? "") : "");
	if (!erro && sessao?.autenticado && sessao.instalacoes.length)
		return <ConteudoRepositorios key={JSON.stringify([sessao.usuario.id, instalacao, q.get("pagina")])} instalacao={instalacao} />;
	return (
		<div className="pagina">
			<h1>Repositórios e instalação</h1>
			<Protegida>
				<PainelDeEstado titulo="Nenhuma instalação autorizada">
					<p>Confira suas instalações no GitHub.</p>
				</PainelDeEstado>
			</Protegida>
		</div>
	);
}
function ConteudoRepositorios({ instalacao }: { instalacao: string }) {
	const { sessao } = useSessao();
	const { q, atualizar } = useFiltrosGestao();
	const pagina = Number(q.get("pagina") ?? 1);
	const autorizada = sessao?.autenticado === true && sessao.instalacoes.some((i) => String(i.id) === instalacao);
	const filtros = new URLSearchParams({ pagina: String(pagina) });
	const { estado, repetir } = useLeitura(saudeRepositorios, instalacao, filtros.toString(), { reutilizar: true, habilitada: autorizada });
	const dados = estado && "dados" in estado ? estado.dados : undefined;
	const erro = autorizada ? (estado && "erro" in estado ? estado.erro : "") : "Instalação não autorizada.";

	return (
		<PaginaGestao titulo="Repositórios e instalação" subtitulo={`${dados?.total ?? "—"} repositórios ativos`}>
			{sessao?.autenticado && (
				<SeletorGestao
					rotulo="Instalação"
					valor={instalacao ?? ""}
					aoMudar={(v) => {
						atualizar("installation_id", v);
					}}
					opcoes={sessao.instalacoes.map((i) => ({ valor: String(i.id), rotulo: i.conta }))}
				/>
			)}
			{erro ? (
				<PainelDeEstado titulo="Consulta de repositórios indisponível" erro>
					<p>{erro}</p>
					<Botao onClick={repetir}>Tentar novamente</Botao>
				</PainelDeEstado>
			) : !dados ? (
				<p role="status">Consultando repositórios…</p>
			) : (
				<>
					<SaudeDosRepositorios instalacao={instalacao} dados={dados} atualizar={repetir} />
					{dados.total > 10 && (
						<div className="controles">
							<Botao disabled={pagina <= 1} onClick={() => atualizar("pagina", String(pagina - 1))}>
								Anterior
							</Botao>
							<span>Página {pagina}</span>
							<Botao disabled={pagina * 10 >= dados.total} onClick={() => atualizar("pagina", String(pagina + 1))}>
								Próxima
							</Botao>
						</div>
					)}
				</>
			)}
		</PaginaGestao>
	);
}
