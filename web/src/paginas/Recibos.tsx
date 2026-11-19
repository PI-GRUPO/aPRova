import { type ReactNode, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { useMobile } from "@/app/responsivo";
import { Ativo } from "@/componentes/Ativo";
import { ColecaoRecibos } from "@/componentes/ColecaoRecibos";
import { Botao, Campo, Modal, PainelDeEstado } from "@/componentes/Primitivos";
import {
	baixar,
	buscarChaves,
	buscarRecibo,
	type Chave,
	mensagemDeErro,
	objeto,
	type Recibo,
	uuidValido,
	type Verificacao,
	verificarRecibo,
} from "@/servicos/recibos";
import "@/estilos/recibos.css";

function Painel({
	titulo,
	ajuda,
	children,
	assinado = false,
}: {
	titulo: string;
	ajuda?: string;
	children: ReactNode;
	assinado?: boolean;
}) {
	return (
		<section className={`painel-recibo ${assinado ? "recibo-assinado" : ""}`}>
			<header>
				<h2>{titulo}</h2>
				{ajuda && (
					<Modal
						gatilho={
							<Botao variante="discreto" className="ajuda-recibo" aria-label={`Ajuda: ${titulo}`}>
								<Ativo nome="ajuda" />
							</Botao>
						}
						titulo={titulo}
						descricao={ajuda}
					>
						<p>Consulte as conferências para conhecer o resultado da verificação.</p>
					</Modal>
				)}
			</header>
			<div className="conteudo-recibo">{children}</div>
			{assinado && (
				<div className="serrilhado-recibo">
					<Ativo nome="serrilhado" />
				</div>
			)}
		</section>
	);
}
function Localizar() {
	const [id, setId] = useState("");
	const [erro, setErro] = useState("");
	const navegar = useNavigate();
	return (
		<form
			className="controles localizar-recibo"
			onSubmit={(e) => {
				e.preventDefault();
				const valor = id.trim().toLowerCase();
				if (!uuidValido(valor)) {
					setErro("Informe um UUID válido.");
					return;
				}
				navegar(`/recibos/${valor}`);
			}}
		>
			<Campo
				rotulo="UUID do recibo"
				value={id}
				onChange={(e) => {
					setId(e.target.value);
					setErro("");
				}}
				erro={erro}
				autoComplete="off"
				required
			/>
			<Botao type="submit">Localizar recibo</Botao>
		</form>
	);
}
function ChavesPublicas({ chaves }: { chaves: Chave[] }) {
	return (
		<>
			{chaves.length === 0 ? (
				<p>Nenhuma chave pública disponível.</p>
			) : (
				chaves.map((c) => (
					<div className="chave-recibo" key={c.key_id}>
						<p className="caminho">
							{c.key_id} · {c.algoritmo}
						</p>
						<code className="caminho">{c.chave_publica_base64}</code>
						<Botao onClick={() => baixar("chave-publica.base64", c.chave_publica_base64, "text/plain")}>
							Exportar chave pública de {c.key_id}
						</Botao>
					</div>
				))
			)}
		</>
	);
}
export function Recibos() {
	const [chaves, setChaves] = useState<Chave[]>();
	const [erro, setErro] = useState("");
	const [tentativa, setTentativa] = useState(0);
	// biome-ignore lint/correctness/useExhaustiveDependencies: o contador permite repetir a consulta após uma falha.
	useEffect(() => {
		const c = new AbortController();
		setErro("");
		setChaves(undefined);
		buscarChaves(c.signal)
			.then((v) => {
				if (!c.signal.aborted) setChaves(v);
			})
			.catch((e) => {
				if (!c.signal.aborted) setErro(mensagemDeErro(e));
			});
		return () => c.abort();
	}, [tentativa]);
	return (
		<div className="pagina pagina-recibos">
			<Link className="voltar voltar-recibos-mobile" to="/">
				<Ativo nome="voltar" />
				Voltar à visão geral
			</Link>
			<div className="titulo-contexto">
				<h1>
					<span className="titulo-recibos-desktop">Recibos e auditoria</span>
					<span className="titulo-recibos-mobile">Recibos</span>
				</h1>
				<p className="apoio titulo-recibos-desktop">
					Trilha de decisões assinadas · coleção de demonstração e consulta pública real por UUID
				</p>
				<p className="apoio titulo-recibos-mobile">Demonstração · dados simulados</p>
			</div>
			<ColecaoRecibos />
			<h2>Consultar recibo real</h2>
			<Localizar />
			<Painel titulo="Chaves públicas">
				{erro ? (
					<div role="alert">
						<p>{erro}</p>
						<Botao onClick={() => setTentativa((v) => v + 1)}>Tentar novamente</Botao>
					</div>
				) : chaves ? (
					<ChavesPublicas chaves={chaves} />
				) : (
					<p role="status">Carregando chaves públicas…</p>
				)}
			</Painel>
		</div>
	);
}
const nomes: Record<string, string> = {
	hash: "Hash",
	chave: "Chave conhecida",
	assinatura: "Assinatura Ed25519",
	conteudo: "Conteúdo",
	elo: "Elo anterior",
};
const resultados = { ok: "Confere", falhou: "Falhou", nao_verificada: "Não verificada" };
function texto(v: unknown) {
	return typeof v === "string" || typeof v === "number" ? String(v) : "Não informado";
}
function data(v: unknown) {
	const d = typeof v === "string" ? new Date(v) : null;
	return d && !Number.isNaN(d.valueOf()) ? `${d.toLocaleString("pt-BR", { timeZone: "UTC" })} UTC` : "Não informada";
}
export function DetalheDoRecibo() {
	const { id = "" } = useParams();
	// A chave desmonta o estado anterior também em navegação entre dois UUIDs.
	return <ConsultaDoRecibo key={id} id={id.toLowerCase()} />;
}
type ConsultaVerificacao = { resposta: Verificacao; instante: string };

function ConsultaDoRecibo({ id }: { id: string }) {
	const mobile = useMobile();
	const [mostrarVerificacao, setMostrarVerificacao] = useState(false);
	const tituloVerificacao = useRef<HTMLHeadingElement>(null);
	const [recibo, setRecibo] = useState<Recibo>();
	const [erro, setErro] = useState("");
	const [tentativa, setTentativa] = useState(0);
	const [chaves, setChaves] = useState<Chave[]>();
	const [erroChaves, setErroChaves] = useState("");
	const [tentativaChaves, setTentativaChaves] = useState(0);
	const [verificacao, setVerificacao] = useState<ConsultaVerificacao>();
	const [verificando, setVerificando] = useState(false);
	const [erroVerificacao, setErroVerificacao] = useState("");
	const controlador = useRef<AbortController | null>(null);
	// biome-ignore lint/correctness/useExhaustiveDependencies: o contador permite repetir a consulta após uma falha.
	useEffect(() => {
		const c = new AbortController();
		setErro("");
		setRecibo(undefined);
		if (!uuidValido(id)) {
			setErro("Recibo não encontrado. Confira o UUID informado.");
			return;
		}
		buscarRecibo(id, c.signal)
			.then((v) => {
				if (!c.signal.aborted) setRecibo(v);
			})
			.catch((e) => {
				if (!c.signal.aborted) setErro(mensagemDeErro(e));
			});
		return () => c.abort();
	}, [id, tentativa]);
	// biome-ignore lint/correctness/useExhaustiveDependencies: o contador permite repetir a consulta após uma falha.
	useEffect(() => {
		const c = new AbortController();
		setErroChaves("");
		buscarChaves(c.signal)
			.then((v) => {
				if (!c.signal.aborted) setChaves(v);
			})
			.catch((e) => {
				if (!c.signal.aborted) setErroChaves(mensagemDeErro(e));
			});
		return () => c.abort();
	}, [tentativaChaves]);
	useEffect(() => () => controlador.current?.abort(), []);
	async function verificar() {
		if (!recibo || verificando) return;
		const c = new AbortController();
		controlador.current = c;
		setVerificando(true);
		setErroVerificacao("");
		setVerificacao(undefined);
		try {
			const resposta = await verificarRecibo(recibo, c.signal);
			if (!c.signal.aborted) {
				setVerificacao({ resposta, instante: new Date().toISOString() });
				setMostrarVerificacao(true);
			}
		} catch (e) {
			if (!c.signal.aborted) setErroVerificacao(mensagemDeErro(e));
		} finally {
			if (!c.signal.aborted) setVerificando(false);
		}
	}
	useEffect(() => {
		if (mobile && mostrarVerificacao && verificacao) {
			window.scrollTo(0, 0);
			tituloVerificacao.current?.focus({ preventScroll: true });
		}
	}, [mobile, mostrarVerificacao, verificacao]);
	const p = recibo?.payload;
	const chave = chaves?.find((c) => c.key_id === recibo?.key_id);
	const anterior = p && objeto(p.anterior) ? p.anterior : null;
	if (mobile && mostrarVerificacao && verificacao)
		return (
			<div className="pagina pagina-verificacao-mobile">
				<button
					type="button"
					className="voltar voltar-verificacao"
					onClick={() => {
						setMostrarVerificacao(false);
						requestAnimationFrame(() => document.getElementById("verificar-recibo")?.focus());
					}}
				>
					<Ativo nome="voltar" />
					Voltar ao recibo
				</button>
				<h1 ref={tituloVerificacao} tabIndex={-1}>
					Verificação do recibo
				</h1>
				<p className="apoio">Recibo {id.slice(0, 8)}… · conferência do núcleo</p>
				<div className="resultado-verificacao-mobile">
					<Painel
						titulo="Conferências"
						ajuda="Hash, chave, assinatura, conteúdo e elo anterior. Esta consulta não comprova toda a cadeia da instalação."
					>
						<ResultadoVerificacao consulta={verificacao} compacto />
					</Painel>
				</div>
			</div>
		);

	return (
		<div className="pagina">
			<Link className="voltar" to="/recibos">
				<Ativo nome="voltar" />
				Voltar para recibos
			</Link>
			<div className="titulo-contexto">
				<h1>Recibo / {id.slice(0, 8)}…</h1>
				{recibo && (
					<p className="apoio">
						{texto(p?.repositorio)} · PR {texto(p?.numero_pr)} · commit {texto(p?.head_sha)} · emissão {data(p?.emitido_em)}
					</p>
				)}
			</div>
			{erro ? (
				<PainelDeEstado titulo="Não foi possível abrir o recibo" erro>
					<p>{erro}</p>
					<Botao onClick={() => setTentativa((v) => v + 1)}>Tentar novamente</Botao>
					<Localizar />
				</PainelDeEstado>
			) : !recibo ? (
				<p role="status">Carregando recibo…</p>
			) : (
				<>
					<div className="grade-recibo">
						<Painel
							titulo="Conteúdo assinado"
							ajuda="O conteúdo registrado só tem sua integridade confirmada após a verificação. A exportação preserva os bytes assinados em base64."
							assinado
						>
							<dl className="dados-recibo">
								{[
									["Recibo", recibo.id],
									["Análise", recibo.task_run_id],
									["Repositório", texto(p?.repositorio)],
									["PR / commit", `#${texto(p?.numero_pr)} · ${texto(p?.head_sha)}`],
									["key_id", recibo.key_id],
									["Algoritmo", chave?.algoritmo ?? "Chave não disponível"],
								].map(([rotulo, valor]) => (
									<div key={rotulo}>
										<dt>{rotulo}</dt>
										<dd>{valor}</dd>
									</div>
								))}
							</dl>
							<p>Risco: {texto(p?.nivel)}</p>
							<p className="mono">
								Motivos:{" "}
								{Array.isArray(p?.motivos)
									? p.motivos
											.filter(objeto)
											.map((m) => texto(m.id))
											.join(", ") || "Nenhum registrado"
									: "Não informados"}
							</p>
							{!p && (
								<p role="status">Conteúdo legível ausente. Os bytes exportados continuam disponíveis para verificação independente.</p>
							)}
						</Painel>
						<Painel
							titulo="Vínculo com o recibo anterior"
							ajuda="A API verifica o elo anterior deste recibo. Isso não equivale a verificar toda a cadeia da instalação."
						>
							<p>Hash e assinatura preservam a integridade da decisão registrada.</p>
							<div className="mono apoio">
								<p>anterior.id: {recibo.recibo_anterior_id ?? "Nenhum recibo anterior"}</p>
								<p>anterior.hash: {texto(anterior?.hash)}</p>
							</div>
							{recibo.recibo_anterior_id && uuidValido(recibo.recibo_anterior_id) && (
								<Link to={`/recibos/${recibo.recibo_anterior_id}`}>Consultar recibo anterior</Link>
							)}
							<p className="apoio">A verificação deste recibo não valida toda a cadeia.</p>
							<Botao disabled>Verificar cadeia da instalação</Botao>
							<p className="apoio">Verificação da cadeia completa ainda indisponível.</p>
						</Painel>
					</div>
					<Painel titulo="Evidências da classificação">
						<p>
							Pelas regras: {texto(p?.nivel_deterministico)} · Risco final: {texto(p?.nivel)}
						</p>
						{p && objeto(p.sandbox) ? (
							<p className="apoio">
								Testes: {texto(p.sandbox.testes_passaram)} passaram · {texto(p.sandbox.testes_falharam)} falharam · tipo de falha:{" "}
								{texto(p.sandbox.tipo_falha)}
							</p>
						) : (
							<p className="apoio">Resultado do sandbox não informado.</p>
						)}
						{Array.isArray(p?.motivos) &&
							p.motivos.filter(objeto).map((m) => (
								<p className="apoio" key={JSON.stringify(m)}>
									{texto(m.id)}: {Array.isArray(m.evidencias) ? m.evidencias.map(texto).join(" · ") : "Evidências não informadas"}
								</p>
							))}
					</Painel>
					<Painel
						titulo="Verificação"
						ajuda="Hash, chave, assinatura, conteúdo e elo são conferidos pelo núcleo. Falha não significa que as etapas seguintes foram executadas."
					>
						<div aria-live="polite">
							{verificando ? (
								<p role="status">Verificando recibo…</p>
							) : erroVerificacao ? (
								<p role="alert">{erroVerificacao}</p>
							) : verificacao ? (
								<ResultadoVerificacao consulta={verificacao} />
							) : (
								<p>Ainda não executada nesta sessão.</p>
							)}
						</div>
					</Painel>
					<div className="acoes-recibo">
						<Botao id="verificar-recibo" variante="primario" onClick={verificar} disabled={verificando}>
							Verificar recibo
						</Botao>
						<Botao onClick={() => baixar(`recibo-${recibo.id}.json`, JSON.stringify(recibo, null, 2))}>Exportar recibo</Botao>
					</div>
					<Painel titulo="Chave pública e verificação independente">
						{erroChaves ? (
							<div role="alert">
								<p>{erroChaves}</p>
								<Botao onClick={() => setTentativaChaves((v) => v + 1)}>Consultar chaves novamente</Botao>
							</div>
						) : chaves ? (
							chave ? (
								<ChavesPublicas chaves={[chave]} />
							) : (
								<p>Chave desconhecida: {recibo.key_id}. Nenhuma chave correspondente disponível para exportação.</p>
							)
						) : (
							<p role="status">Carregando chave pública…</p>
						)}
						<p className="apoio">Baixe o recibo e a chave pública. Com o executável verificar-recibo do projeto, execute:</p>
						<pre className="comando-recibo">./verificar-recibo -chave 'CHAVE_PUBLICA_BASE64' recibo-{recibo.id}.json</pre>
						<p className="apoio">
							Use o valor base64 do arquivo de chave exportado. A conferência independente de assinatura e conteúdo não verifica o elo com o
							recibo anterior.
						</p>
					</Painel>
				</>
			)}
		</div>
	);
}

function ResultadoVerificacao({ consulta, compacto = false }: { consulta: ConsultaVerificacao; compacto?: boolean }) {
	const nomesCurtos: Record<string, string> = { hash: "Hash", chave: "Chave", assinatura: "Assinatura", conteudo: "Conteúdo", elo: "Elo" };
	return (
		<div className="resultado-verificacao">
			<p>{consulta.resposta.valido ? "Recibo válido." : "Não foi possível validar o recibo."}</p>
			<p className="apoio">Consultado nesta sessão em {data(consulta.instante)}</p>
			{consulta.resposta.detalhe && <p>{consulta.resposta.detalhe}</p>}
			<table className="conferencias-recibo">
				<caption className="apoio">Conferências do núcleo</caption>
				<thead>
					<tr>
						<th scope="col">Etapa</th>
						<th scope="col">Resultado</th>
					</tr>
				</thead>
				<tbody>
					{consulta.resposta.verificacoes.map((c) => (
						<tr key={c.nome}>
							<td>{compacto ? nomesCurtos[c.nome] : nomes[c.nome]}</td>
							<td>{resultados[c.resultado]}</td>
						</tr>
					))}
				</tbody>
			</table>
		</div>
	);
}
