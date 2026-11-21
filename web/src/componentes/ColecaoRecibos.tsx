import { type ReactNode, useEffect, useRef, useState } from "react";
import { useMobile } from "@/app/responsivo";
import { consultarColecao, exportarColecao, type RegistroRecibo, verificarCadeia } from "@/servicos/colecao-recibos";
import { baixar } from "@/servicos/recibos";
import { ConsultaPrivada, useEscopo } from "./ConsultaPrivada";
import { opcoesPeriodo, SecaoGestao, SeletorGestao, TabelaGestao, useFiltrosGestao } from "./Gestao";
import { Botao, Modal } from "./Primitivos";

export function ColecaoRecibos() {
	return (
		<ConsultaPrivada
			secao
			ajuda={<p>A coleção, a verificação da cadeia e a exportação por instalação ainda estão indisponíveis.</p>}
			titulo="Recibos por repositório"
			consultar={consultarColecao}
		>
			{(dados) => <Lista {...dados} />}
		</ConsultaPrivada>
	);
}
function Lista({ repositorios, itens }: { repositorios: string[]; itens: RegistroRecibo[] }) {
	const mobile = useMobile();
	const { q, atualizar } = useFiltrosGestao();
	const repo = q.get("repositorio") ?? "";
	const dias = ["7", "30", "90"].includes(q.get("periodo") ?? "") ? Number(q.get("periodo")) : 30;
	const opcoesRepos = repositorios.map((nome) => ({ valor: nome, rotulo: nome }));
	const escopo = useEscopo(repo);
	const [resultado, definir] = useState("");
	const [erro, setErro] = useState("");
	const [ocupado, setOcupado] = useState(false);
	const consulta = useRef<AbortController | null>(null);
	useEffect(() => () => consulta.current?.abort(), []);
	async function executar(acao: "cadeia" | "json" | "csv") {
		if (consulta.current !== null || !(acao === "cadeia" ? verificarCadeia : exportarColecao).disponivel) return;
		const c = new AbortController();
		consulta.current = c;
		setOcupado(true);
		definir("");
		setErro("");
		try {
			if (acao === "cadeia") {
				const v = await verificarCadeia.executar(escopo, c.signal);
				if (!c.signal.aborted)
					definir(v.valida ? "Cadeia verificada." : `Falha na verificação da cadeia. ${v.primeiroReciboInvalido ?? ""}`);
			} else {
				const arquivo = await exportarColecao.executar(escopo, dias, acao, c.signal);
				if (!c.signal.aborted) baixar(`recibos.${acao}`, arquivo);
			}
		} catch {
			if (!c.signal.aborted) setErro("Não foi possível concluir a operação.");
		} finally {
			consulta.current = null;
			if (!c.signal.aborted) setOcupado(false);
		}
	}
	const ferramentas = (
		<>
			<SeletorGestao rotulo="Período" valor={String(dias)} aoMudar={(v) => atualizar("periodo", v)} opcoes={opcoesPeriodo} />
			<div className="acoes-colecao">
				<Botao disabled={!verificarCadeia.disponivel || ocupado} onClick={() => void executar("cadeia")}>
					Verificar cadeia
				</Botao>
				{(["json", "csv"] as const).map((f) => (
					<Botao key={f} disabled={!exportarColecao.disponivel || ocupado} onClick={() => void executar(f)}>
						Exportar {f.toUpperCase()}
					</Botao>
				))}
			</div>
			{!verificarCadeia.disponivel && !exportarColecao.disponivel && (
				<p className="apoio">Verificação da cadeia e exportação da coleção indisponíveis.</p>
			)}
			{resultado && <p role="status">{resultado}</p>}
			{erro && <p role="alert">{erro}</p>}
		</>
	);
	return (
		<>
			<div className="filtros-gestao filtros-recibos">
				<SeletorGestao
					rotulo="Repositório"
					valor={repo === "" ? "todos" : repo}
					aoMudar={(v) => atualizar("repositorio", v === "todos" ? "" : v)}
					opcoes={[{ valor: "todos", rotulo: "Todos autorizados" }, ...opcoesRepos]}
				/>
				{!mobile && ferramentas}
			</div>
			{mobile ? (
				<ul className="cards-recibos" aria-label="Recibos">
					{itens.map((r) => (
						<li className="card-recibo" key={r.id}>
							<h2>{r.id.slice(0, 8)}…</h2>
							<div>
								<p>
									{r.repo} #{r.pr}
								</p>
								<p className="apoio">Assinatura não verificada</p>
								<PreviaDoRecibo recibo={r} gatilho={<Botao aria-label={`Ver prévia do recibo ${r.id}`}>Ver prévia do recibo</Botao>} />
							</div>
						</li>
					))}
					{!itens.length && <li className="apoio">Nenhum recibo para os filtros selecionados.</li>}
				</ul>
			) : (
				<SecaoGestao titulo="Recibos por repositório" ajuda="Consulte um recibo pelo UUID para verificar sua assinatura.">
					<TabelaGestao
						legenda="Recibos por repositório"
						colunas={["Recibo", "Análise", "Commit", "Emissão", "Verificação"]}
						larguras={["236px", "256px", "136px", "176px", "188px"]}
						linhas={itens.map((r) => ({
							id: r.id,
							celulas: [
								<PreviaDoRecibo
									key="recibo"
									recibo={r}
									gatilho={
										<button className="link-fila" type="button">
											{r.id.slice(0, 8)}…
										</button>
									}
								/>,
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
			)}
			{mobile && (
				<details className="ferramentas-recibos-mobile">
					<summary>Período, cadeia e exportação</summary>
					<div>{ferramentas}</div>
				</details>
			)}
		</>
	);
}

function PreviaDoRecibo({ recibo: r, gatilho }: { recibo: RegistroRecibo; gatilho: ReactNode }) {
	return (
		<Modal titulo="Prévia do recibo" descricao="Identificação do recibo" gatilho={gatilho}>
			<p>
				{r.repo} #{r.pr} · commit {r.sha}
			</p>
			<p>A verificação individual exige um recibo assinado.</p>
		</Modal>
	);
}
