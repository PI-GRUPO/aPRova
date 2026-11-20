import { useState } from "react";
import { useNavigate } from "react-router";
import { useEscopo } from "@/componentes/ConsultaPrivada";
import { PaginaGestao, SecaoGestao, SeletorGestao, TabelaGestao, useFiltrosGestao, useGestao, useOpcoesRepos } from "@/componentes/Gestao";
import { Botao, Modal } from "@/componentes/Primitivos";
import { revisarSugestao, type Sugestao } from "@/servicos/gestao";

const rotulos = { pendente: "Pendente", em_revisao: "Em revisão", recusada: "Recusada", aplicada: "Aplicada" };
function RevisarSugestao({ sugestao }: { sugestao: Sugestao }) {
	const escopo = useEscopo(sugestao.repo);
	const navegar = useNavigate();
	const [aberto, setAberto] = useState(false);
	const [erro, setErro] = useState("");
	const [ocupado, setOcupado] = useState(false);
	async function agir(acao: "recusada" | "em_revisao") {
		if (ocupado || !revisarSugestao.disponivel) return;
		setOcupado(true);
		setErro("");
		try {
			await revisarSugestao.executar(escopo, sugestao.id, acao);
			setAberto(false);
			if (acao === "em_revisao") navegar(`/configuracao?${new URLSearchParams({ repositorio: sugestao.repo, proposta: sugestao.id })}`);
		} catch (e) {
			setErro(e instanceof Error ? e.message : "Não foi possível revisar.");
		} finally {
			setOcupado(false);
		}
	}
	return (
		<Modal
			aberto={aberto}
			aoMudar={(v) => {
				if (!ocupado) {
					setAberto(v);
					setErro("");
				}
			}}
			titulo="Revisar proposta de regra"
			descricao={sugestao.resumo}
			gatilho={
				<button type="button" className="link-fila">
					{sugestao.titulo}
				</button>
			}
		>
			<p className="secundario">{sugestao.evidencia}</p>
			<p>A proposta aprovada vira regra determinística somente após passar pela configuração versionada.</p>
			{!revisarSugestao.disponivel && <p className="apoio">A revisão de propostas está indisponível.</p>}
			{erro && <p role="alert">{erro}</p>}
			{
				<div className="acoes-modal">
					<div>
						{sugestao.estado === "pendente" && (
							<>
								<Botao disabled={ocupado || !revisarSugestao.disponivel} onClick={() => void agir("recusada")}>
									Recusar
								</Botao>
								<Botao variante="primario" disabled={ocupado || !revisarSugestao.disponivel} onClick={() => void agir("em_revisao")}>
									Revisar configuração
								</Botao>
							</>
						)}
						{sugestao.estado === "em_revisao" && (
							<Botao onClick={() => navegar(`/configuracao?${new URLSearchParams({ repositorio: sugestao.repo, proposta: sugestao.id })}`)}>
								Continuar revisão
							</Botao>
						)}
					</div>
					<Botao disabled={ocupado} onClick={() => setAberto(false)}>
						Cancelar
					</Botao>
				</div>
			}
		</Modal>
	);
}
export function Sugestoes() {
	const opcoesRepos = useOpcoesRepos();
	const { q, atualizar } = useFiltrosGestao();
	const dados = useGestao();
	const repo = q.get("repositorio") ?? dados.sugestoes[0]?.repo ?? opcoesRepos[0]?.valor ?? "";
	const estado = q.get("estado") ?? "pendente";
	const itens = dados.sugestoes.filter((s) => s.repo === repo && (estado === "todos" || s.estado === estado));
	return (
		<PaginaGestao titulo="Sugestões de regras pela IA" subtitulo="Propostas de regras para revisão humana">
			<div className="filtros-gestao filtros-sugestoes">
				<SeletorGestao
					rotulo="Estado"
					valor={estado}
					aoMudar={(v) => atualizar("estado", v)}
					opcoes={[
						{ valor: "pendente", rotulo: "Pendentes" },
						{ valor: "em_revisao", rotulo: "Em revisão" },
						{ valor: "recusada", rotulo: "Recusadas" },
						{ valor: "aplicada", rotulo: "Aplicadas" },
						{ valor: "todos", rotulo: "Todas" },
					]}
				/>
				<SeletorGestao rotulo="Repositório" valor={repo} aoMudar={(v) => atualizar("repositorio", v)} opcoes={opcoesRepos} />
			</div>
			<SecaoGestao
				titulo="Propostas aguardando revisão"
				ajuda="A IA propõe e a pessoa revisa. Uma proposta não altera o gate automaticamente."
			>
				<TabelaGestao
					legenda="Propostas de regras"
					colunas={["Proposta", "Evidência de origem", "Estado"]}
					larguras={["456px", "376px", "120px"]}
					linhas={itens.map((s) => ({ id: s.id, celulas: [<RevisarSugestao key="proposta" sugestao={s} />, s.origem, rotulos[s.estado]] }))}
				/>
			</SecaoGestao>
		</PaginaGestao>
	);
}
