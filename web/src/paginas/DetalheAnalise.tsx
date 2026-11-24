import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { useSessao } from "@/app/sessao";
import { AcoesDaAnalise } from "@/componentes/AcoesDaAnalise";
import { dataUTC, Estado, PainelAnalise, Risco } from "@/componentes/Analise";
import { Ativo, type NomeDoAtivo } from "@/componentes/Ativo";
import { useLeitura } from "@/componentes/ConsultaPrivada";
import { Botao, Modal, PainelDeEstado } from "@/componentes/Primitivos";
import { CabecalhoDaTabela, Celula, Coluna, CorpoDaTabela, LinhaDaTabela, Tabela } from "@/componentes/Tabela";
import type { ResultadoAcao } from "@/servicos/acoes-analise";
import { type Detalhe, detalharAnalise, type Etapa, estados, urlGitHub } from "@/servicos/analises";
import type { ComplementoDemonstracao } from "@/servicos/analises-demonstracao";
import type { Consulta } from "@/servicos/fonte";
import { buscarRecibo, type Verificacao, verificarRecibo } from "@/servicos/recibos";
import { Protegida } from "./Acesso";
import "@/estilos/analises.css";
export function DetalheAnalise() {
	const { id = "" } = useParams();
	const [q] = useSearchParams();
	const { sessao } = useSessao();
	const repositorio = q.get("repositorio") ?? "";
	const instalacao = q.get("installation_id") ?? (sessao?.autenticado ? String(sessao.instalacoes[0]?.id ?? "") : "");
	return (
		<div className="pagina">
			{!sessao?.autenticado && <h1>Detalhe da análise</h1>}
			<Protegida>
				{sessao?.autenticado && sessao.instalacoes.some((i) => String(i.id) === instalacao) ? (
					<ConsultaAnalise
						key={JSON.stringify([sessao.usuario.id, instalacao, repositorio, id])}
						id={id}
						instalacao={instalacao}
						repositorio={repositorio}
					/>
				) : (
					<PainelDeEstado titulo="Instalação não autorizada">
						<p>Selecione uma instalação disponível na lista de análises.</p>
					</PainelDeEstado>
				)}
			</Protegida>
		</div>
	);
}
const consultarDetalhe: Consulta<Detalhe> = ({ instalacao, repositorio }, filtros, signal) =>
	detalharAnalise(filtros.get("id") ?? "", instalacao, signal, repositorio);

function marcadorDaEtapa(etapa: Etapa, proxima: Etapa | undefined, estadoAtual: string): { icone: NomeDoAtivo; rotulo: string } {
	switch (etapa.para) {
		case "concluido":
			return { icone: "concluido", rotulo: "Concluída" };
		case "falhou":
			return { icone: "falhou", rotulo: "Falhou" };
		case "rejeitado":
			return { icone: "falhou", rotulo: "Rejeitada" };
	}
	// O card representa o estado alcançado. A próxima transição registra a saída dessa etapa.
	if (proxima?.de === etapa.para) {
		return proxima.para === "falhou" ? { icone: "falhou", rotulo: "Interrompida" } : { icone: "concluido", rotulo: "Concluída" };
	}
	if (etapa.para === estadoAtual) return { icone: "clock", rotulo: estadoAtual === "aguardando_aprovacao" ? "Aguardando" : "Em andamento" };
	return { icone: "clock", rotulo: "Transição registrada" };
}

function ConsultaAnalise({ id, instalacao, repositorio }: { id: string; instalacao: string; repositorio: string }) {
	const [q] = useSearchParams();
	const [resultadoAcao, setResultadoAcao] = useState<ResultadoAcao>();
	const filtros = new URLSearchParams({ id, repositorio });
	const { estado, repetir } = useLeitura(consultarDetalhe, instalacao, filtros.toString(), {
		reutilizar: true,
		manterDadosDuranteAtualizacao: true,
	});
	const dados = estado && "dados" in estado ? estado.dados : undefined;
	const erro = estado && "erro" in estado ? estado.erro : "";
	useEffect(() => {
		if (!dados?.atual) return;
		const estadoAtual = dados.analise.estado;
		if (
			estadoAtual !== "aguardando_aprovacao" &&
			estadoAtual !== "aplicando" &&
			!(estadoAtual === "falhou" && resultadoAcao?.tipo === "reexecutar_analise")
		)
			return;
		// Agenda a próxima leitura só depois que os dados anteriores chegarem.
		let solicitada = false;
		const atualizar = () => {
			if (solicitada || document.visibilityState !== "visible") return;
			solicitada = true;
			repetir();
		};
		const timer = window.setTimeout(atualizar, 5_000);
		document.addEventListener("visibilitychange", atualizar);
		window.addEventListener("focus", atualizar);
		return () => {
			window.clearTimeout(timer);
			document.removeEventListener("visibilitychange", atualizar);
			window.removeEventListener("focus", atualizar);
		};
	}, [dados, repetir, resultadoAcao?.tipo]);
	const retorno = new URLSearchParams(q.get("retorno") ?? "");
	retorno.set("installation_id", instalacao);
	if (!q.has("retorno") && dados) retorno.set("repositorio", dados.analise.repositorio);
	return (
		<>
			<Link className="voltar" to={`${q.get("origem") === "fila" ? "/" : "/analises"}?${retorno}`}>
				<Ativo nome="voltar" />
				{q.get("origem") === "fila" ? "Voltar à visão geral" : "Voltar às análises"}
			</Link>
			{erro ? (
				<>
					<h1>Detalhe da análise</h1>
					<PainelDeEstado titulo="Não foi possível abrir a análise" erro>
						<p>{erro}</p>
						<Botao onClick={repetir}>Tentar novamente</Botao>
					</PainelDeEstado>
				</>
			) : !dados ? (
				<>
					<h1>Detalhe da análise</h1>
					<p role="status">Carregando análise…</p>
				</>
			) : (
				<ConteudoAnalise
					dados={dados}
					instalacao={instalacao}
					aoAtualizar={repetir}
					resultadoAcao={resultadoAcao}
					aoReceberAcao={setResultadoAcao}
				/>
			)}
		</>
	);
}
export function ConteudoAnalise({
	dados: d,
	instalacao,
	demonstracao = false,
	complementoSimulado,
	linhaDoTempoSimulada,
	aoAtualizar,
	resultadoAcao,
	aoReceberAcao,
}: {
	dados: Detalhe;
	instalacao: string;
	demonstracao?: boolean;
	complementoSimulado?: ComplementoDemonstracao;
	linhaDoTempoSimulada?: ReactNode;
	aoAtualizar?: () => void;
	resultadoAcao?: ResultadoAcao;
	aoReceberAcao?: (resultado: ResultadoAcao) => void;
}) {
	const { sessao, cacheDeLeituras } = useSessao();
	const csrfDasAcoes = !demonstracao && !__MOCKS__ && sessao?.autenticado && sessao.acoes_disponiveis ? sessao.csrf_token : "";
	const a = d.analise;
	const g = d.gate;

	const usuario = sessao?.autenticado ? String(sessao.usuario.id) : "";
	return (
		<>
			<div className="titulo-contexto">
				<h1 className="caminho">
					{demonstracao ? (
						<>
							<span className="titulo-demonstracao-desktop">
								{a.repositorio} / PR #{a.numero_pr}
							</span>
							<span className="titulo-demonstracao-mobile">
								{a.repositorio} #{a.numero_pr}
							</span>
						</>
					) : (
						<>
							{a.repositorio} / PR #{a.numero_pr}
						</>
					)}
				</h1>
				<p className="apoio caminho metadados-analise">
					{demonstracao && "Dados simulados · "}Análise {a.id} · commit {a.sha_head.slice(0, 7)} · recebida em {dataUTC(a.recebido_em)}
				</p>
			</div>
			<div className="controles estado-do-detalhe">
				<Risco nivel={a.nivel} />
				<Estado estado={a.estado} />
			</div>
			{demonstracao && (
				<p className="apoio mono resumo-demonstracao-mobile">
					Commit {a.sha_head.slice(0, 7)} · autor {a.autor}
				</p>
			)}
			{!d.atual && (
				<PainelDeEstado titulo="Esta análise foi substituída">
					<p>Existe uma análise mais recente deste PR. A decisão e as evidências abaixo pertencem exclusivamente ao commit {a.sha_head}.</p>
					<Link
						to={`/analises?${new URLSearchParams({ installation_id: instalacao, repositorio: a.repositorio, numero_pr: String(a.numero_pr), periodo: "todos" })}`}
					>
						Consultar histórico deste PR
					</Link>
				</PainelDeEstado>
			)}
			<div className="grade-analise">
				<PainelAnalise titulo="Identificação">
					<dl className="identificacao-analise">
						{[
							["Autor", a.autor],
							["Branch", d.branch ?? "Não informada nesta análise"],
							["Commit", a.sha_head],
						].map(([nome, valor]) => (
							<div key={nome}>
								<dt>{nome}</dt>
								<dd>{valor}</dd>
							</div>
						))}
					</dl>
					<div className="controles">
						{demonstracao ? (
							<Botao disabled>Abrir PR</Botao>
						) : (
							<a className="botao" href={urlGitHub(d.url_pr)} target="_blank" rel="noreferrer">
								Abrir PR
							</a>
						)}
						{demonstracao ? (
							<Botao disabled>Abrir Check Run</Botao>
						) : d.url_check ? (
							<a className="botao" href={urlGitHub(d.url_check)} target="_blank" rel="noreferrer">
								Abrir Check Run
							</a>
						) : (
							<p className="apoio">Check Run não registrado.</p>
						)}
					</div>
				</PainelAnalise>
				<PainelAnalise
					titulo="Motivo e evidências"
					ajuda="O gate registra o nível decidido e as regras acionadas. Ausência de gate significa que o risco ainda não foi avaliado."
				>
					{g ? (
						<>
							{!demonstracao && <p>{g.motivo}</p>}
							{g.motivos.map((m) => (
								<div className="motivo-analise" key={m.id}>
									<code>{m.id}</code>
									<p className={demonstracao ? "descricao-regra-demonstracao" : undefined}>{m.descricao}</p>
									<ul>
										{m.evidencias.map((e) => (
											<li key={e}>{e}</li>
										))}
									</ul>
								</div>
							))}
							<Link className="botao" to="/regras">
								Ver regra e limites
							</Link>
						</>
					) : (
						<p>Risco não avaliado. Nenhum gate registrado para esta execução.</p>
					)}
				</PainelAnalise>
			</div>
			<PainelAnalise
				titulo="Calibração do risco"
				ajuda="A calibração e os casos semelhantes são o histórico registrado durante esta análise, limitado aos repositórios atualmente autorizados."
			>
				{g ? (
					<>
						<p>
							{demonstracao ? g.nivel_deterministico.replace(/^./, (l) => l.toUpperCase()) : g.nivel_deterministico} →{" "}
							{demonstracao ? g.nivel.replace(/^./, (l) => l.toUpperCase()) : g.nivel}
						</p>
						<p>{g.calibracao_razao}</p>
						{complementoSimulado ? (
							<>
								<p className="apoio">{complementoSimulado.contextoCalibracao}</p>
								<p className="apoio">{complementoSimulado.ajudaCalibracao}</p>
							</>
						) : (
							<p className="apoio">Estado registrado: {g.calibracao_estado}</p>
						)}
						<details>
							<summary>Ver casos semelhantes</summary>
							{g.vizinhos.length ? (
								<ul>
									{g.vizinhos.map((v) => (
										<li key={v.task_run_id}>
											<Link
												to={
													demonstracao
														? `/analises/demonstracao/${v.task_run_id}`
														: `/analises/${v.task_run_id}?installation_id=${instalacao}`
												}
											>
												{v.repositorio} #{v.numero_pr}
											</Link>{" "}
											· similaridade {(v.similaridade * 100).toLocaleString("pt-BR", { maximumFractionDigits: 1 })}% · {v.desfecho}
										</li>
									))}
								</ul>
							) : (
								<p className="apoio">Nenhum caso semelhante disponível neste escopo.</p>
							)}
						</details>
					</>
				) : (
					<p>Calibração indisponível sem decisão do gate.</p>
				)}
			</PainelAnalise>
			<PainelAnalise titulo="Linha do tempo da execução">
				{linhaDoTempoSimulada ??
					(d.etapas.length ? (
						<ol className="timeline-analise">
							{d.etapas.map((e, i) => {
								const { icone, rotulo } = marcadorDaEtapa(e, d.etapas[i + 1], a.estado);
								const titulo = estados.get(e.para) ?? e.para;
								return (
									<li key={e.id}>
										<span className="marcador-etapa">
											<Ativo nome={icone} />
											{rotulo}
										</span>
										<Modal
											titulo={titulo}
											descricao={`De ${estados.get(e.de) ?? e.de} para ${titulo}, em ${dataUTC(e.criado_em)}.`}
											gatilho={<Botao variante="discreto">{titulo}</Botao>}
										>
											<p>{e.detalhe || "Nenhum detalhe adicional registrado."}</p>
										</Modal>
										<time className="apoio" dateTime={e.criado_em}>
											{dataUTC(e.criado_em)}
										</time>
									</li>
								);
							})}
						</ol>
					) : (
						<p>Histórico de transições não disponível.</p>
					))}
			</PainelAnalise>
			<PainelAnalise
				titulo="Sandbox e saídas"
				ajuda="Os contadores e a saída pertencem a cada tentativa registrada. Sem resultados estruturados, nomes de testes não podem ser inferidos."
			>
				{d.tentativas.length ? (
					<>
						{/* biome-ignore lint/a11y/noNoninteractiveTabindex: a região rolável precisa receber foco para rolagem por teclado. */}
						<section className="tabela-rolavel" aria-label="Tentativas do sandbox, tabela rolável" tabIndex={0}>
							<Tabela legenda="Tentativas do sandbox">
								<CabecalhoDaTabela>
									{["Tentativa", "Compilou", "Passaram", "Falharam", "Duração", "Tipo de falha"].map((c) => (
										<Coluna key={c}>{c}</Coluna>
									))}
								</CabecalhoDaTabela>
								<CorpoDaTabela>
									{d.tentativas.map((t) => (
										<LinhaDaTabela key={t.tentativa}>
											<Celula>{t.tentativa}</Celula>
											<Celula>{t.compilou ? "Sim" : "Não"}</Celula>
											<Celula>{t.testes_passaram}</Celula>
											<Celula>{t.testes_falharam}</Celula>
											<Celula>{(t.duracao_ms / 1000).toLocaleString("pt-BR")} s</Celula>
											<Celula>{t.tipo_falha || "Nenhuma"}</Celula>
										</LinhaDaTabela>
									))}
								</CorpoDaTabela>
							</Tabela>
						</section>
						{d.tentativas.map((t) => (
							<details key={t.tentativa}>
								<summary>Saída da tentativa {t.tentativa}</summary>
								{t.saida_truncada && <p className="apoio">Saída truncada pela consulta em 65.536 caracteres.</p>}
								<pre className="saida-analise">{t.saida || "Saída não disponível."}</pre>
							</details>
						))}
						{!d.testes_estruturados_disponiveis && (
							<p className="apoio">Nomes e estados individuais de testes não estão disponíveis; os valores acima são contadores.</p>
						)}
						<p className="apoio">
							{d.rede_isolada === null
								? "Isolamento de rede não informado pelo executor."
								: d.rede_isolada
									? "Isolamento de rede confirmado pelo executor."
									: "Executor informou rede não isolada."}
						</p>
					</>
				) : (
					<p>Nenhuma tentativa de sandbox registrada.</p>
				)}
			</PainelAnalise>
			<PainelAnalise titulo="Decisão humana">
				{d.aprovacoes.length ? (
					<ul className="aprovacoes-analise">
						{d.aprovacoes.map((p) => (
							<li key={p.id}>
								<p>
									{p.decisao} · @{p.revisor} · origem {p.origem}
								</p>
								<p className="apoio">
									{dataUTC(p.decidido_em)} · commit {p.commit_sha ?? "não registrado"}
								</p>
								<p>{p.justificativa ?? "Justificativa não registrada."}</p>
								{!p.aplicavel_ao_commit && (
									<p className="apoio">Registro de outro commit ou sem SHA confirmado; não aplicado ao commit desta análise.</p>
								)}
							</li>
						))}
					</ul>
				) : (
					<p>Nenhuma decisão humana registrada para esta análise.</p>
				)}
				<AcoesDaAnalise
					key={`${instalacao}:${a.id}:${a.sha_head}:${usuario}`}
					csrf={csrfDasAcoes}
					resultado={resultadoAcao}
					aoReceber={aoReceberAcao}
					aoAtualizar={() => {
						cacheDeLeituras?.limpar();
						aoAtualizar?.();
					}}
					contexto={{
						instalacao,
						id: a.id,
						repo: a.repositorio,
						pr: a.numero_pr,
						sha: a.sha_head,
						usuario,
						nivel: a.nivel,
						estado: a.estado,
						atual: d.atual,
						proprioPR:
							!demonstracao && !!sessao?.autenticado && sessao.usuario.login.toLowerCase() === a.autor.replace(/^@/, "").toLowerCase(),
						falhaInfra: d.gate === null && d.tentativas.at(-1)?.tipo_falha === "infraestrutura",
					}}
				/>
				{!csrfDasAcoes && <p className="apoio">Ações indisponíveis nesta visualização.</p>}
			</PainelAnalise>
			<PainelAnalise
				titulo="Recibo assinado"
				ajuda="A verificação confere hash, chave, assinatura, conteúdo e elo anterior; não verifica toda a cadeia."
			>
				{demonstracao ? (
					<>
						<p>Verificação de assinatura indisponível.</p>
						{d.recibo_id ? <code className="caminho">{d.recibo_id}</code> : <p>Recibo indisponível.</p>}
						<Link to={`/recibos?${new URLSearchParams({ repositorio: a.repositorio })}`}>Consultar recibos</Link>
					</>
				) : d.recibo_id ? (
					<ReciboDaAnalise id={d.recibo_id} analiseID={a.id} />
				) : (
					<p>Recibo ainda não emitido para esta análise.</p>
				)}
			</PainelAnalise>
		</>
	);
}
function ReciboDaAnalise({ id, analiseID }: { id: string; analiseID: string }) {
	const [resultado, setResultado] = useState<{ resposta: Verificacao; data: string }>();
	const [erro, setErro] = useState("");
	const [verificando, setVerificando] = useState(false);
	const controlador = useRef<AbortController | null>(null);
	useEffect(() => () => controlador.current?.abort(), []);
	async function verificar() {
		if (verificando) return;
		const c = new AbortController();
		controlador.current = c;
		setVerificando(true);
		setResultado(undefined);
		setErro("");
		try {
			const recibo = await buscarRecibo(id, c.signal);
			if (recibo.task_run_id !== analiseID) throw new Error("O recibo não corresponde à análise.");
			const resposta = await verificarRecibo(recibo, c.signal);
			if (!c.signal.aborted) setResultado({ resposta, data: new Date().toISOString() });
		} catch (e) {
			if (!c.signal.aborted) setErro(e instanceof Error ? e.message : "Não foi possível verificar o recibo.");
		} finally {
			if (!c.signal.aborted) setVerificando(false);
		}
	}
	return (
		<>
			<p className="caminho">Recibo {id}</p>
			<div className="controles">
				<Botao onClick={verificar} disabled={verificando}>
					Verificar recibo
				</Botao>
				<Link className="botao" to={`/recibos/${id}`}>
					Abrir recibo e exportar
				</Link>
			</div>
			<div aria-live="polite">
				{erro ? (
					<p role="alert">{erro}</p>
				) : verificando ? (
					<p role="status">Verificando recibo…</p>
				) : resultado ? (
					<>
						<p>{resultado.resposta.valido ? "Recibo válido." : "Não foi possível validar o recibo."}</p>
						<p className="apoio">Consultado nesta sessão em {dataUTC(resultado.data)}</p>
						{resultado.resposta.detalhe && <p>{resultado.resposta.detalhe}</p>}
						<ul>
							{resultado.resposta.verificacoes.map((v) => (
								<li key={v.nome}>
									{v.nome}: {v.resultado === "ok" ? "Confere" : v.resultado === "falhou" ? "Falhou" : "Não verificada"}
								</li>
							))}
						</ul>
					</>
				) : (
					<p className="apoio">Ainda não executada nesta sessão.</p>
				)}
			</div>
		</>
	);
}
