import { type Dispatch, useId, useRef, useState } from "react";
import { Link } from "react-router";
import { type ContextoAcao, motivoIndisponivel, type ResultadoAcao, registrarAcao, type TipoAcao } from "@/servicos/acoes-analise";
import { niveis } from "@/servicos/analises";
import { Risco } from "./Analise";
import { Botao, Modal } from "./Primitivos";
import "@/estilos/acoes.css";

type AcaoNoPainel = { tipo: TipoAcao; botao: string; titulo: string; confirmar: string; resultado: string };

const acoes: readonly AcaoNoPainel[] = [
	{ tipo: "aprovar", botao: "Aprovar", titulo: "Aprovar PR", confirmar: "Confirmar aprovação", resultado: "Solicitação recebida" },
	{
		tipo: "solicitar_alteracao",
		botao: "Pedir alterações",
		titulo: "Pedir alteração",
		confirmar: "Enviar solicitação",
		resultado: "Alteração solicitada",
	},
	{
		tipo: "reexecutar_analise",
		botao: "Reexecutar análise",
		titulo: "Reexecutar análise",
		confirmar: "Reexecutar análise",
		resultado: "Reexecução solicitada",
	},
];
function FormularioAcao({
	contexto,
	acao,
	resultado,
	aoReceber,
	aberto,
	aoMudar,
	csrf,
}: {
	contexto: ContextoAcao;
	acao: AcaoNoPainel;
	resultado?: ResultadoAcao;
	aoReceber: Dispatch<ResultadoAcao>;
	aberto: boolean;
	aoMudar: (v: boolean) => void;
	csrf: string;
}) {
	const { tipo } = acao;
	const [justificativa, setJustificativa] = useState("");
	const [erro, setErro] = useState("");
	const [enviando, setEnviando] = useState(false);
	const trava = useRef(false);
	const idAcao = useRef<string | null>(null);
	const campoId = useId();
	const motivo = motivoIndisponivel(contexto, tipo) || (!csrf ? "Ação indisponível no painel." : "");
	const respostaDaAcao = resultado?.tipo === tipo ? resultado : undefined;
	async function enviar() {
		if (motivo || trava.current || resultado) return;
		if (tipo !== "reexecutar_analise" && !justificativa.trim()) {
			setErro("Informe uma justificativa com até 4000 caracteres.");
			return;
		}
		trava.current = true;
		setEnviando(true);
		setErro("");
		try {
			if (idAcao.current === null) idAcao.current = crypto.randomUUID();
			aoReceber(await registrarAcao.executar(contexto, tipo, justificativa, idAcao.current, csrf));
		} catch (e) {
			setErro(e instanceof Error ? e.message : "Não foi possível enviar a solicitação. Tente novamente.");
		} finally {
			trava.current = false;
			setEnviando(false);
		}
	}
	return (
		<Modal
			aberto={aberto}
			aoMudar={(v) => {
				if (!trava.current) {
					aoMudar(v);
					setErro("");
				}
			}}
			className={`modal-acao modal-acao-${tipo}`}
			titulo={respostaDaAcao ? acao.resultado : tipo === "aprovar" ? `${acao.titulo} #${contexto.pr}` : acao.titulo}
			descricao="Confira a análise e o commit antes de confirmar."
			gatilho={
				<Botao disabled={!!motivo || !!resultado} title={motivo || undefined} variante={tipo === "aprovar" ? "primario" : "secundario"}>
					{acao.botao}
				</Botao>
			}
		>
			{respostaDaAcao ? (
				<>
					<p role="status">{respostaDaAcao.mensagem}</p>
					<p className="apoio caminho">
						Análise {respostaDaAcao.contexto.id} · commit {respostaDaAcao.contexto.sha}.
					</p>
					<div className="rodape-acao">
						{tipo === "reexecutar_analise" && (
							<Link
								className="botao botao-secundario"
								to={`/analises?${new URLSearchParams({
									installation_id: contexto.instalacao,
									repositorio: contexto.repo,
									numero_pr: String(contexto.pr),
								}).toString()}`}
								onClick={() => {
									aoMudar(false);
								}}
							>
								Ver análises deste PR
							</Link>
						)}
						<Botao
							variante="primario"
							onClick={() => {
								aoMudar(false);
							}}
						>
							Voltar à análise
						</Botao>
					</div>
				</>
			) : (
				<>
					{tipo !== "reexecutar_analise" && <Risco nivel={contexto.nivel} />}
					<p className="apoio mono caminho contexto-acao">
						{contexto.repo} #{contexto.pr} ·{" "}
						{tipo === "solicitar_alteracao" ? niveis.get(contexto.nivel ?? "nao_avaliado") : `commit ${contexto.sha.slice(0, 7)}`}
					</p>
					<p className="explicacao-acao">
						{tipo === "aprovar"
							? "Revise as evidências. Aprovar não executa o merge."
							: tipo === "reexecutar_analise"
								? "Solicitar uma nova tentativa para este commit."
								: "Informe quais alterações são necessárias."}
					</p>
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void enviar();
						}}
						noValidate
					>
						{tipo !== "reexecutar_analise" && (
							<div className="campo campo-justificativa">
								<label htmlFor={campoId}>Justificativa obrigatória</label>
								<textarea
									id={campoId}
									value={justificativa}
									onChange={(e) => {
										setJustificativa(e.target.value);
									}}
									maxLength={4000}
									rows={4}
									aria-invalid={!!erro}
									aria-describedby={erro ? `${campoId}-erro` : `${campoId}-ajuda`}
								/>
							</div>
						)}
						{erro && (
							<div>
								<p role="alert" id={`${campoId}-erro`}>
									{erro}
								</p>
							</div>
						)}
						<p className="apoio ajuda-acao" id={`${campoId}-ajuda`}>
							Até 4000 caracteres.
						</p>
						<div className="rodape-acao">
							<Botao
								disabled={enviando}
								onClick={() => {
									aoMudar(false);
								}}
							>
								Cancelar
							</Botao>
							<Botao variante="primario" type="submit" disabled={enviando}>
								{enviando ? "Registrando…" : acao.confirmar}
							</Botao>
						</div>
					</form>
				</>
			)}
		</Modal>
	);
}
export function AcoesDaAnalise({
	contexto,
	csrf,
	resultado,
	aoReceber,
	aoAtualizar,
}: {
	contexto: ContextoAcao;
	csrf: string;
	resultado?: ResultadoAcao;
	aoReceber?: Dispatch<ResultadoAcao>;
	aoAtualizar?: () => void;
}) {
	const [aberta, setAberta] = useState<TipoAcao>();
	const aguardando =
		resultado &&
		contexto.atual &&
		(resultado.tipo === "reexecutar_analise"
			? contexto.estado === "falhou"
			: contexto.estado === "aguardando_aprovacao" || contexto.estado === "aplicando");
	return (
		<div className="acoes-da-analise">
			<div className="controles">
				{acoes.map((acao) => (
					<FormularioAcao
						key={acao.tipo}
						contexto={contexto}
						acao={acao}
						csrf={csrf}
						resultado={resultado}
						aoReceber={(v) => aoReceber?.(v)}
						aberto={aberta === acao.tipo}
						aoMudar={(v) => {
							setAberta(v ? acao.tipo : undefined);
							if (!v && resultado?.tipo === acao.tipo) aoAtualizar?.();
						}}
					/>
				))}
			</div>
			{contexto.proprioPR && <p className="apoio">O autor não pode aprovar o próprio PR.</p>}
			{!contexto.atual && <p className="apoio">Esta análise foi substituída. Consulte a análise atual.</p>}
			{aguardando && !aberta && <p role="status">Aguardando atualização da análise.</p>}
		</div>
	);
}
