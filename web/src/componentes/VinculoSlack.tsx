import { useRef, useState } from "react";
import { simularVinculoSlack } from "@/servicos/vinculo-slack";
import { Botao, Modal } from "./Primitivos";
export function VinculoSlack({ login }: { login: string }) {
	const [conectado, setConectado] = useState(false);
	const [aberto, setAberto] = useState(false);
	const [enviando, setEnviando] = useState(false);
	const trava = useRef(false);
	async function confirmar() {
		if (trava.current) return;
		trava.current = true;
		setEnviando(true);
		try {
			const resposta = await simularVinculoSlack(!conectado);
			setConectado(resposta.conectado);
			setAberto(false);
		} finally {
			trava.current = false;
			setEnviando(false);
		}
	}
	return (
		<section className="painel-sessao">
			<h2>Conectar Slack</h2>
			<div>
				<dl className="dados-identidade">
					<div>
						<dt>Workspace</dt>
						<dd>aPRova · demonstração</dd>
					</div>
					<div>
						<dt>Estado do vínculo</dt>
						<dd>{conectado ? "Conectado na simulação" : "Ainda não conectado"}</dd>
					</div>
				</dl>
				<p className="apoio">Autorize a conta Slack que pertence a você.</p>
				<Modal
					aberto={aberto}
					aoMudar={(v) => {
						if (!enviando) setAberto(v);
					}}
					titulo={conectado ? "Desvincular Slack" : "Conectar sua conta Slack"}
					descricao="Demonstração · nenhuma identidade real será vinculada ou removida"
					gatilho={<Botao variante="primario">{conectado ? "Desvincular conta" : "Conectar minha conta Slack"}</Botao>}
				>
					{conectado ? (
						<>
							<p>Remover o vínculo simulado de @{login} no workspace aPRova?</p>
							<p>A demonstração não altera o acesso pelo GitHub nem o reconhecimento de ações no Slack.</p>
						</>
					) : (
						<dl className="dados-identidade">
							<div>
								<dt>GitHub da sessão</dt>
								<dd>@{login}</dd>
							</div>
							<div>
								<dt>Workspace</dt>
								<dd>aPRova · demonstração</dd>
							</div>
							<div>
								<dt>Conta Slack</dt>
								<dd>Pessoa de demonstração</dd>
							</div>
						</dl>
					)}
					<div className="acoes-modal">
						<Botao disabled={enviando} onClick={() => setAberto(false)}>
							Cancelar
						</Botao>
						<Botao variante="primario" disabled={enviando} onClick={() => void confirmar()}>
							{enviando ? "Atualizando…" : conectado ? "Desvincular conta" : "Autorizar vínculo"}
						</Botao>
					</div>
				</Modal>
				<p className="apoio" role="status">
					Vínculo simulado em memória. {conectado ? "Conectado somente nesta página." : "Nenhuma conta Slack conectada pela demonstração."}
				</p>
			</div>
		</section>
	);
}
