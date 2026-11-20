import { useRef, useState } from "react";
import { Ativo } from "@/componentes/Ativo";
import { useEscopo } from "@/componentes/ConsultaPrivada";
import { PaginaGestao, SecaoGestao, SeletorGestao, TabelaGestao, useFiltrosGestao, useGestao, useOpcoesRepos } from "@/componentes/Gestao";
import { Botao, Campo, Modal } from "@/componentes/Primitivos";
import { type Configuracao as Modelo, salvarConfiguracao } from "@/servicos/gestao";

export function Configuracao() {
	const opcoesRepos = useOpcoesRepos();
	const { q, atualizar } = useFiltrosGestao();
	const repo = q.get("repositorio") ?? opcoesRepos[0]?.valor ?? "";
	const dados = useGestao();
	const atual = opcoesRepos.some((r) => r.valor === repo) ? dados.configuracoes[repo] : undefined;
	if (!atual) return <p>Configuração indisponível para este repositório.</p>;
	return (
		<PaginaGestao titulo="Configuração do gate" subtitulo="Limites e áreas sensíveis por repositório">
			<div className="filtros-gestao escopo-config">
				<SeletorGestao rotulo="Repositório" valor={repo} aoMudar={(v) => atualizar("repositorio", v)} opcoes={opcoesRepos} />
				<Campo
					rotulo="Versão em vigor"
					readOnly
					value={`v${atual.versao} · aplicada em ${atual.historico[0]?.data.split(" · ")[0] ?? "data indisponível"}`}
				/>
			</div>
			<FormularioConfiguracao
				key={`${repo}:${q.get("proposta") ?? ""}`}
				repo={repo}
				atual={atual}
				proposta={q.get("proposta") ?? undefined}
			/>
		</PaginaGestao>
	);
}
function FormularioConfiguracao({ repo, atual, proposta }: { repo: string; atual: Modelo; proposta?: string }) {
	const dados = useGestao();
	const escopo = useEscopo(repo);
	const sugestao = dados.sugestoes.find((s) => s.id === proposta && s.repo === repo && ["pendente", "em_revisao"].includes(s.estado));
	const [limite, setLimite] = useState(String(sugestao?.limiteProposto ?? atual.limite));
	const [caminhos, setCaminhos] = useState([...atual.caminhos]);
	const [adicionar, setAdicionar] = useState(false);
	const [novo, setNovo] = useState("");
	const [erroCaminho, setErroCaminho] = useState("");
	const [erro, setErro] = useState("");
	const [revisar, setRevisar] = useState(false);
	const [ciencia, setCiencia] = useState(false);
	const [enviando, setEnviando] = useState(false);
	const [sucesso, setSucesso] = useState("");
	const trava = useRef(false);
	const numero = Number(limite);
	const invalido = !Number.isSafeInteger(numero) || numero < 1;
	const mudou = numero !== atual.limite || JSON.stringify(caminhos) !== JSON.stringify(atual.caminhos);
	const removidos = atual.caminhos.filter((c) => !caminhos.includes(c));
	const reduziu = numero > atual.limite || removidos.length > 0;
	function adicionarCaminho() {
		const valor = novo.trim();
		if (
			!valor ||
			valor.length > 200 ||
			caminhos.includes(valor) ||
			valor.startsWith("/") ||
			valor.includes("\\") ||
			valor.split("/").includes("..")
		) {
			setErroCaminho("Informe um caminho relativo, único e com até 200 caracteres.");
			return;
		}
		setCaminhos((v) => [...v, valor]);
		setNovo("");
		setErroCaminho("");
		setAdicionar(false);
		setSucesso("");
	}
	async function salvar() {
		if (!salvarConfiguracao.disponivel || trava.current || invalido || !mudou || (reduziu && !ciencia)) return;
		trava.current = true;
		setEnviando(true);
		setErro("");
		try {
			const v = await salvarConfiguracao.executar(escopo, numero, caminhos, atual.versao, sugestao?.id);
			setRevisar(false);
			setSucesso(`Versão v${v.versao} registrada.`);
		} catch (e) {
			setErro(e instanceof Error ? e.message : "Não foi possível salvar.");
		} finally {
			setEnviando(false);
			trava.current = false;
		}
	}
	return (
		<>
			<div className="colunas-config">
				<SecaoGestao
					titulo="Limites e áreas sensíveis"
					ajuda="O limite de arquivos e os caminhos sensíveis definem proteções por repositório. Mudanças devem ser revisadas e versionadas."
					caixa
				>
					<Campo
						rotulo="Limite de arquivos"
						type="number"
						min={1}
						step={1}
						value={limite}
						onChange={(e) => {
							setLimite(e.target.value);
							setSucesso("");
						}}
						erro={invalido ? "Informe um inteiro positivo." : undefined}
					/>
					<p className="apoio">Limite usado pela regra para identificar alterações com muitos arquivos.</p>
					<div className="lista-caminhos">
						<p className="apoio">Áreas sensíveis</p>
						{caminhos.map((c) => (
							<div className="caminho-sensivel" key={c}>
								<span>{c}</span>
								<Botao
									variante="discreto"
									aria-label={`Remover ${c}`}
									onClick={() => {
										setCaminhos((v) => v.filter((x) => x !== c));
										setSucesso("");
									}}
								>
									<Ativo nome="fechar" />
								</Botao>
							</div>
						))}
						{!caminhos.length && <p className="apoio">Nenhum caminho selecionado.</p>}
					</div>
					<p className="apoio">Adicione pastas ou arquivos. Use o × para remover um caminho da lista.</p>
					<Modal
						className="modal-pequeno"
						aberto={adicionar}
						aoMudar={(v) => {
							setAdicionar(v);
							if (!v) {
								setNovo("");
								setErroCaminho("");
							}
						}}
						titulo="Adicionar área sensível"
						descricao="Caminho da pasta ou arquivo"
						gatilho={<Botao variante="primario">Adicionar caminho</Botao>}
					>
						<form
							onSubmit={(e) => {
								e.preventDefault();
								adicionarCaminho();
							}}
						>
							<Campo
								rotulo="Caminho da pasta ou arquivo"
								placeholder="billing/"
								value={novo}
								onChange={(e) => setNovo(e.target.value)}
								erro={erroCaminho}
							/>
							<div className="acoes-modal">
								<Botao onClick={() => setAdicionar(false)}>Fechar</Botao>
								<Botao type="submit" variante="primario">
									Adicionar
								</Botao>
							</div>
						</form>
					</Modal>
					{!salvarConfiguracao.disponivel && <p className="apoio">A publicação de novas versões está indisponível.</p>}
					<div className="acoes-gestao">
						<Botao
							disabled={!mudou}
							onClick={() => {
								setLimite(String(atual.limite));
								setCaminhos([...atual.caminhos]);
								setSucesso("");
								setErro("");
							}}
						>
							Descartar alterações
						</Botao>
						<Modal
							aberto={revisar}
							aoMudar={(v) => {
								if (!enviando) {
									setRevisar(v);
									setCiencia(false);
									setErro("");
								}
							}}
							titulo="Confirmar nova versão do gate"
							descricao={`${repo} · nova versão v${atual.versao + 1}`}
							gatilho={
								<Botao variante="primario" disabled={!mudou || invalido}>
									Revisar mudanças
								</Botao>
							}
						>
							<TabelaGestao
								compacta
								legenda="Revisão das mudanças"
								colunas={["Campo", "Antes", "Depois"]}
								linhas={[
									...(numero !== atual.limite
										? [{ id: "limite", celulas: ["Limite de arquivos", String(atual.limite), String(numero)] }]
										: []),
									...(JSON.stringify(caminhos) !== JSON.stringify(atual.caminhos)
										? [
												{
													id: "caminhos",
													celulas: [
														"Áreas sensíveis",
														<span className="texto-quebrado" key="antes">
															{atual.caminhos.join(", ") || "Nenhuma"}
														</span>,
														<span className="texto-quebrado" key="depois">
															{caminhos.join(", ") || "Nenhuma"}
														</span>,
													],
												},
											]
										: []),
								]}
							/>
							{reduziu && (
								<label className="checkbox-gestao aviso-protecao">
									<input type="checkbox" checked={ciencia} onChange={(e) => setCiencia(e.target.checked)} />
									Estou ciente de que esta alteração reduz a proteção
									{removidos.length ? ` de ${removidos.join(", ")}` : " pelo limite de arquivos"}.
								</label>
							)}
							{erro && <p role="alert">{erro}</p>}
							<div className="acoes-modal">
								<Botao disabled={enviando} onClick={() => setRevisar(false)}>
									Cancelar
								</Botao>
								<Botao
									variante="primario"
									disabled={!salvarConfiguracao.disponivel || enviando || (reduziu && !ciencia)}
									onClick={() => void salvar()}
								>
									{enviando ? "Registrando…" : "Confirmar e criar versão"}
								</Botao>
							</div>
						</Modal>
					</div>
				</SecaoGestao>
				<SecaoGestao titulo="Histórico de versões" ajuda="Versões anteriores da configuração do repositório.">
					<TabelaGestao
						compacta
						legenda="Histórico de versões"
						colunas={["Versão", "Pessoa", "Data"]}
						linhas={atual.historico.map((h) => ({ id: String(h.versao), celulas: [`v${h.versao}`, h.pessoa, h.data] }))}
					/>
				</SecaoGestao>
			</div>
			{sucesso && (
				<p role="status" className="apoio">
					{sucesso}
				</p>
			)}
		</>
	);
}
