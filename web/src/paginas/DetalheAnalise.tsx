import { useEffect, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { useSessao } from "@/app/sessao";
import { AcoesSimuladas } from "@/componentes/AcoesSimuladas";
import { dataUTC, Estado, PainelAnalise, Risco } from "@/componentes/Analise";
import { Ativo } from "@/componentes/Ativo";
import { Botao, Modal, PainelDeEstado } from "@/componentes/Primitivos";
import { CabecalhoDaTabela, Celula, Coluna, CorpoDaTabela, LinhaDaTabela, Tabela } from "@/componentes/Tabela";
import { type Detalhe, detalharAnalise, erroDeAnalise, estados, urlGitHub } from "@/servicos/analises";
import { buscarRecibo, type Verificacao, verificarRecibo } from "@/servicos/recibos";
import { Protegida } from "./Acesso";
import "@/estilos/analises.css";
export function DetalheAnalise() {
	const { id = "" } = useParams();
	const [q] = useSearchParams();
	const { sessao } = useSessao();
	const instalacao = q.get("installation_id") ?? (sessao?.autenticado ? String(sessao.instalacoes[0]?.id ?? "") : "");
	return (
		<div className="pagina">
			{!sessao?.autenticado && <h1>Detalhe da análise</h1>}
			<Protegida>
				<ConsultaAnalise key={`${sessao?.autenticado ? sessao.usuario.id : ""}:${instalacao}:${id}`} id={id} instalacao={instalacao} />
			</Protegida>
		</div>
	);
}
function ConsultaAnalise({ id, instalacao }: { id: string; instalacao: string }) {
	const [dados, setDados] = useState<Detalhe>();
	const [erro, setErro] = useState("");
	const [tentativa, setTentativa] = useState(0);
	const [q] = useSearchParams();
	// biome-ignore lint/correctness/useExhaustiveDependencies: tentativa permite repetir uma consulta por ação explícita.
	useEffect(() => {
		const c = new AbortController();
		setDados(undefined);
		setErro("");
		detalharAnalise(id, instalacao, c.signal)
			.then((v) => {
				if (!c.signal.aborted) setDados(v);
			})
			.catch((e) => {
				if (!c.signal.aborted) setErro(erroDeAnalise(e));
			});
		return () => c.abort();
	}, [id, instalacao, tentativa]);
	const retorno = new URLSearchParams(q.get("retorno") ?? "");
	retorno.set("installation_id", instalacao);
	if (!q.has("retorno") && dados) retorno.set("repositorio", dados.analise.repositorio);
	return (
		<>
			<Link className="voltar" to={`/analises?${retorno}`}>
				<Ativo nome="voltar" />
				Voltar às análises
			</Link>
			{erro ? (
				<>
					<h1>Detalhe da análise</h1>
					<PainelDeEstado titulo="Não foi possível abrir a análise" erro>
						<p>{erro}</p>
						<Botao onClick={() => setTentativa((v) => v + 1)}>Tentar novamente</Botao>
					</PainelDeEstado>
				</>
			) : !dados ? (
				<>
					<h1>Detalhe da análise</h1>
					<p role="status">Carregando análise…</p>
				</>
			) : (
				<Conteudo dados={dados} instalacao={instalacao} />
			)}
		</>
	);
}
function Conteudo({ dados: d, instalacao }: { dados: Detalhe; instalacao: string }) {
	const { sessao } = useSessao();
	const a = d.analise;
	const g = d.gate;
	return (
		<>
			<div className="titulo-contexto">
				<h1 className="caminho">
					{a.repositorio} / PR #{a.numero_pr}
				</h1>
				<p className="apoio caminho">
					Análise {a.id} · commit {a.sha_head.slice(0, 7)} · recebida em {dataUTC(a.recebido_em)}
				</p>
			</div>
			<div className="controles">
				<Risco nivel={a.nivel} />
				<Estado estado={a.estado} />
			</div>
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
						<a className="botao" href={urlGitHub(d.url_pr)} target="_blank" rel="noreferrer">
							Abrir PR
						</a>
						{d.url_check ? (
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
							<p>{g.motivo}</p>
							{g.motivos.map((m) => (
								<div className="motivo-analise" key={m.id}>
									<code>{m.id}</code>
									<p>{m.descricao}</p>
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
							{g.nivel_deterministico} → {g.nivel}
						</p>
						<p>{g.calibracao_razao}</p>
						<p className="apoio">Estado registrado: {g.calibracao_estado}</p>
						<details>
							<summary>Ver casos semelhantes</summary>
							{g.vizinhos.length ? (
								<ul>
									{g.vizinhos.map((v) => (
										<li key={v.task_run_id}>
											<Link to={`/analises/${v.task_run_id}?installation_id=${instalacao}`}>
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
				{d.etapas.length ? (
					<ol className="timeline-analise">
						{d.etapas.map((e) => (
							<li key={e.id}>
								<span className="marcador-etapa">
									<Ativo nome={e.para === "concluido" ? "concluido" : e.para === "falhou" ? "falhou" : "clock"} />
									{e.para === "concluido" ? "Concluída" : e.para === "falhou" ? "Falhou" : "Transição registrada"}
								</span>
								<Modal
									titulo={estados[e.para] ?? e.para}
									descricao={`De ${estados[e.de] ?? e.de} para ${estados[e.para] ?? e.para}, em ${dataUTC(e.criado_em)}.`}
									gatilho={<Botao variante="discreto">{estados[e.para] ?? e.para}</Botao>}
								>
									<p>{e.detalhe || "Nenhum detalhe adicional registrado."}</p>
								</Modal>
								<time className="apoio" dateTime={e.criado_em}>
									{dataUTC(e.criado_em)}
								</time>
							</li>
						))}
					</ol>
				) : (
					<p>Histórico de transições não disponível.</p>
				)}
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
				<AcoesSimuladas
					key={`${instalacao}:${a.id}:${a.sha_head}:${a.estado}:${d.atual}:${sessao?.autenticado ? sessao.usuario.id : ""}`}
					contexto={{
						instalacao,
						id: a.id,
						repo: a.repositorio,
						pr: a.numero_pr,
						sha: a.sha_head,
						usuario: sessao?.autenticado ? String(sessao.usuario.id) : "",
						nivel: a.nivel,
						estado: a.estado,
						atual: d.atual,
						proprioPR: !!sessao?.autenticado && sessao.usuario.login.toLowerCase() === a.autor.replace(/^@/, "").toLowerCase(),
						falhaInfra: d.tentativas.at(-1)?.tipo_falha === "infraestrutura",
					}}
				/>
				<p className="apoio">As ações reais pelo painel ainda não estão disponíveis. Você pode revisar pelo GitHub.</p>
			</PainelAnalise>
			<PainelAnalise
				titulo="Recibo assinado"
				ajuda="A verificação confere hash, chave, assinatura, conteúdo e elo anterior; não verifica toda a cadeia."
			>
				{d.recibo_id ? <ReciboDaAnalise id={d.recibo_id} analiseID={a.id} /> : <p>Recibo ainda não emitido para esta análise.</p>}
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
