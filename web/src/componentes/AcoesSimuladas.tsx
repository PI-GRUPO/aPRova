import { useId, useRef, useState } from "react";
import { type ContextoAcao, motivoIndisponivel, type ResultadoAcao, registrarAcao, type TipoAcao } from "@/servicos/acoes-simuladas";
import { niveis } from "@/servicos/analises";
import { Risco } from "./Analise";
import { Botao, Modal } from "./Primitivos";
import "@/estilos/acoes.css";

const rotulos = {
	aprovar: { botao: "Aprovar", titulo: "Aprovar PR", confirmar: "Confirmar aprovação", resultado: "Solicitação recebida" },
	solicitar_alteracao: {
		botao: "Pedir alterações",
		titulo: "Pedir alteração",
		confirmar: "Enviar solicitação",
		resultado: "Alteração solicitada",
	},
	reexecutar_analise: {
		botao: "Reexecutar análise",
		titulo: "Reexecutar análise",
		confirmar: "Reexecutar análise",
		resultado: "Reexecução solicitada",
	},
};
function FormularioAcao({
	contexto,
	tipo,
	resultado,
	aoReceber,
	aberto,
	aoMudar,
}: {
	contexto: ContextoAcao;
	tipo: TipoAcao;
	resultado?: ResultadoAcao;
	aoReceber: (r: ResultadoAcao) => void;
	aberto: boolean;
	aoMudar: (v: boolean) => void;
}) {
	const [justificativa, setJustificativa] = useState("");
	const [erro, setErro] = useState("");
	const [enviando, setEnviando] = useState(false);
	const trava = useRef(false);
	const idAcao = useRef<string>(undefined);
	const campoId = useId();
	const texto = rotulos[tipo];
	const motivo = motivoIndisponivel(contexto, tipo);
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
			idAcao.current ??= crypto.randomUUID();
			aoReceber(await registrarAcao.executar(contexto, tipo, justificativa, idAcao.current));
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
			titulo={respostaDaAcao ? texto.resultado : tipo === "aprovar" ? `${texto.titulo} #${contexto.pr}` : texto.titulo}
			descricao="Confira a análise e o commit antes de confirmar."
			gatilho={
				<Botao
					disabled={!!motivo || (!!resultado && !respostaDaAcao)}
					title={motivo || undefined}
					variante={tipo === "aprovar" ? "primario" : "secundario"}
				>
					{texto.botao}
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
						<Botao variante="primario" onClick={() => aoMudar(false)}>
							Voltar à análise
						</Botao>
					</div>
				</>
			) : (
				<>
					{tipo !== "reexecutar_analise" && <Risco nivel={contexto.nivel} />}
					<p className="apoio mono caminho contexto-acao">
						{contexto.repo} #{contexto.pr} ·{" "}
						{tipo === "solicitar_alteracao" ? niveis[contexto.nivel ?? "nao_avaliado"] : `commit ${contexto.sha.slice(0, 7)}`}
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
									onChange={(e) => setJustificativa(e.target.value)}
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
							<Botao disabled={enviando} onClick={() => aoMudar(false)}>
								Cancelar
							</Botao>
							<Botao variante="primario" type="submit" disabled={enviando}>
								{enviando ? "Registrando…" : texto.confirmar}
							</Botao>
						</div>
					</form>
				</>
			)}
		</Modal>
	);
}
export function AcoesSimuladas({ contexto }: { contexto: ContextoAcao }) {
	const [resultado, setResultado] = useState<ResultadoAcao>();
	const [aberta, setAberta] = useState<TipoAcao>();
	return (
		<div className="acoes-simuladas">
			<div className="controles">
				{(Object.keys(rotulos) as TipoAcao[]).map((tipo) => (
					<FormularioAcao
						key={tipo}
						contexto={contexto}
						tipo={tipo}
						resultado={resultado}
						aoReceber={setResultado}
						aberto={aberta === tipo}
						aoMudar={(v) => setAberta(v ? tipo : undefined)}
					/>
				))}
			</div>
			{contexto.proprioPR && <p className="apoio">O autor não pode aprovar o próprio PR.</p>}
			{!contexto.atual && <p className="apoio">Esta análise foi substituída. Consulte a análise atual.</p>}
		</div>
	);
}
