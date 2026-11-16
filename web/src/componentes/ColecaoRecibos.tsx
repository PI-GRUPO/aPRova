import { useState } from "react";
import { exportarColecaoSimulada, listarRecibosSimulados, verificarCadeiaSimulada } from "@/servicos/colecao-recibos";
import { baixar } from "@/servicos/recibos";
import { opcoesPeriodo, opcoesRepos, SecaoGestao, SeletorGestao, TabelaGestao, useFiltrosGestao } from "./Gestao";
import { Botao, Modal } from "./Primitivos";

export function ColecaoRecibos() {
	const { q, atualizar } = useFiltrosGestao();
	const repo = q.get("repositorio") ?? "";
	const dias = ["7", "30", "90"].includes(q.get("periodo") ?? "") ? Number(q.get("periodo")) : 30;
	const itens = listarRecibosSimulados(repo, dias);
	const [cadeia, setCadeia] = useState<Awaited<ReturnType<typeof verificarCadeiaSimulada>>>();
	const [modal, setModal] = useState(false);
	const [exportacao, setExportacao] = useState<"json" | "csv" | null>(null);
	return (
		<>
			<div className="filtros-gestao filtros-recibos">
				<SeletorGestao
					rotulo="Repositório"
					valor={repo || "todos"}
					aoMudar={(v) => atualizar("repositorio", v === "todos" ? "" : v)}
					opcoes={[{ valor: "todos", rotulo: "Todos autorizados" }, ...opcoesRepos]}
				/>
				<SeletorGestao rotulo="Período" valor={String(dias)} aoMudar={(v) => atualizar("periodo", v)} opcoes={opcoesPeriodo} />
				<div className="acoes-colecao">
					<Modal
						aberto={modal}
						aoMudar={setModal}
						titulo="Cadeia da instalação"
						descricao="Resultado simulado · não comprova a cadeia real da instalação"
						gatilho={
							<Botao
								variante="primario"
								onClick={() => {
									setCadeia(undefined);
									void verificarCadeiaSimulada().then(setCadeia);
								}}
							>
								Verificar cadeia
							</Botao>
						}
					>
						{cadeia ? (
							<div className="estado">
								<h2>{cadeia.estado}</h2>
								<p>
									Recibo {cadeia.recibo} · hash anterior divergente. Último elo íntegro: {cadeia.ultimoIntegro}.
								</p>
							</div>
						) : (
							<p role="status">Carregando cenário…</p>
						)}
						<div className="acoes-modal">
							<Botao onClick={() => setModal(false)}>Fechar resultado</Botao>
						</div>
					</Modal>
					{(["json", "csv"] as const).map((formato) => (
						<Modal
							key={formato}
							aberto={exportacao === formato}
							aoMudar={(v) => setExportacao(v ? formato : null)}
							titulo="Exportar trilha de auditoria"
							descricao="Exportação de demonstração · sem assinaturas ou valor de prova"
							gatilho={<Botao>Exportar {formato.toUpperCase()}</Botao>}
						>
							<p>
								O arquivo contém apenas os {itens.length} registros simulados dos filtros atuais. Para verificar um recibo real, consulte
								seu UUID abaixo.
							</p>
							<div className="acoes-modal">
								<Botao onClick={() => setExportacao(null)}>Cancelar</Botao>
								<Botao
									variante="primario"
									onClick={() => {
										baixar(
											`recibos-demonstracao.${formato}`,
											exportarColecaoSimulada(repo, dias, formato),
											formato === "json" ? "application/json" : "text/csv;charset=utf-8",
										);
										setExportacao(null);
									}}
								>
									Baixar demonstração
								</Botao>
							</div>
						</Modal>
					))}
				</div>
			</div>
			<SecaoGestao
				titulo="Recibos por repositório"
				ajuda="A coleção é simulada. A consulta por UUID e a verificação individual abaixo usam os serviços reais."
			>
				<TabelaGestao
					legenda="Recibos por repositório · demonstração"
					colunas={["Recibo", "Análise", "Commit", "Emissão", "Verificação"]}
					larguras={["236px", "256px", "136px", "176px", "188px"]}
					linhas={itens.map((r) => ({
						id: r.id,
						celulas: [
							<Modal
								key="recibo"
								titulo="Prévia do recibo"
								descricao="Metadados simulados · nenhum recibo assinado foi gerado"
								gatilho={
									<button className="link-fila" type="button">
										{r.id.slice(0, 8)}…
									</button>
								}
							>
								<p>
									{r.repo} #{r.pr} · commit {r.sha}
								</p>
								<p>Esta prévia não possui assinatura, hash ou validade criptográfica. Consulte um UUID real para conferir o recibo.</p>
							</Modal>,
							`${r.repo} #${r.pr}`,
							<code key="sha">{r.sha}</code>,
							new Date(r.emissao).toLocaleString("pt-BR", {
								timeZone: "UTC",
								day: "2-digit",
								month: "short",
								hour: "2-digit",
								minute: "2-digit",
							}),
							"Não executada",
						],
					}))}
				/>
			</SecaoGestao>
			<p className="apoio">Coleção e cadeia simuladas. Nenhum resultado acima comprova assinatura ou integridade real.</p>
		</>
	);
}
