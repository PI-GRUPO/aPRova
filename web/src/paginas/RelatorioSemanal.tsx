import { useEffect, useState } from "react";
import { useSearchParams } from "react-router";
import { useSessao } from "@/app/sessao";
import { Botao, Campo, PainelDeEstado } from "@/componentes/Primitivos";
import { Selecao } from "@/componentes/Selecao";
import { ErroHTTP } from "@/servicos/http";
import { mensagemDeErro } from "@/servicos/recibos";
import { consultarRelatorio, type Relatorio } from "@/servicos/sessao";
import { Protegida } from "./Acesso";
export function RelatorioSemanalPagina() {
	return (
		<div className="pagina">
			<div className="titulo-contexto">
				<h1>Relatório semanal</h1>
				<p className="apoio">Relatório semanal · sete dias em UTC · somente repositórios autorizados</p>
			</div>
			<Protegida>
				<RelatorioSemanal />
			</Protegida>
		</div>
	);
}
function RelatorioSemanal() {
	const { sessao } = useSessao();
	const [q, setQ] = useSearchParams();
	if (!sessao?.autenticado) return null;
	if (!sessao.instalacoes.length)
		return (
			<PainelDeEstado titulo="Nenhuma instalação autorizada">
				<p>Sua conta precisa ter acesso a uma instalação do GitHub App e a seus repositórios.</p>
			</PainelDeEstado>
		);
	const id = q.get("installation_id") ?? String(sessao.instalacoes[0].id);
	const inicio = q.get("inicio") ?? "";
	function atualizar(nome: string, valor: string) {
		setQ((anterior) => {
			const nova = new URLSearchParams(anterior);
			if (valor) nova.set(nome, valor);
			else nova.delete(nome);
			return nova;
		});
	}
	return (
		<>
			<div className="controles">
				<Selecao
					rotulo="Instalação"
					valor={id}
					aoMudar={(v) => atualizar("installation_id", v)}
					opcoes={sessao.instalacoes.map((i) => ({ valor: String(i.id), rotulo: `${i.conta} · ${i.id}` }))}
				/>
				<Campo type="date" rotulo="Início da semana (UTC)" value={inicio} onChange={(e) => atualizar("inicio", e.target.value)} />
				<Botao onClick={() => atualizar("inicio", "")}>Semana anterior</Botao>
			</div>
			{sessao.instalacoes.some((i) => String(i.id) === id) ? (
				<DadosDoRelatorio key={`${sessao.usuario.id}:${id}:${inicio}`} id={id} inicio={inicio} />
			) : (
				<PainelDeEstado titulo="Instalação não autorizada" erro>
					<p>Selecione uma das instalações disponíveis para sua conta.</p>
				</PainelDeEstado>
			)}
		</>
	);
}
function DadosDoRelatorio({ id, inicio }: { id: string; inicio: string }) {
	const [dados, setDados] = useState<Relatorio>();
	const [erro, setErro] = useState("");
	const [tentativa, setTentativa] = useState(0);
	// biome-ignore lint/correctness/useExhaustiveDependencies: o contador repete uma consulta após erro, por ação explícita.
	useEffect(() => {
		const c = new AbortController();
		setDados(undefined);
		setErro("");
		consultarRelatorio(id, inicio, c.signal)
			.then((v) => {
				if (!c.signal.aborted) setDados(v);
			})
			.catch((e) => {
				if (!c.signal.aborted)
					setErro(
						e instanceof ErroHTTP && e.status === 403
							? "O GitHub não autorizou o acesso a este escopo. Confirme sua conta e as permissões da instalação."
							: mensagemDeErro(e),
					);
			});
		return () => c.abort();
	}, [id, inicio, tentativa]);
	if (erro)
		return (
			<PainelDeEstado titulo="Relatório indisponível" erro>
				<p>{erro}</p>
				<Botao onClick={() => setTentativa((v) => v + 1)}>Tentar novamente</Botao>
			</PainelDeEstado>
		);
	if (!dados) return <p role="status">Consultando relatório semanal…</p>;
	const finalizadas = dados.concluidas + dados.rejeitadas + dados.falhas;
	return (
		<>
			<p className="apoio">
				De {new Date(dados.inicio).toLocaleDateString("pt-BR", { timeZone: "UTC" })} (inclusive) a{" "}
				{new Date(dados.fim).toLocaleDateString("pt-BR", { timeZone: "UTC" })} (exclusive), UTC.
			</p>
			{dados.total_execucoes === 0 && (
				<PainelDeEstado titulo="Sem análises neste período">
					<p>Não há análises recebidas nos repositórios autorizados durante esta semana.</p>
				</PainelDeEstado>
			)}
			<dl className="metricas-semanais">
				{[
					["Análises recebidas", dados.total_execucoes, "na janela de sete dias"],
					["Concluídas", dados.concluidas, `de ${finalizadas} finalizadas`],
					[
						"Taxa de sucesso",
						dados.taxa_sucesso === null
							? "Sem dados"
							: `${(dados.taxa_sucesso * 100).toLocaleString("pt-BR", { maximumFractionDigits: 1 })}%`,
						dados.taxa_sucesso === null ? "nenhuma análise finalizada" : `${dados.concluidas} concluídas de ${finalizadas} finalizadas`,
					],
					["Em andamento", dados.em_andamento, "entre as recebidas no período"],
				].map(([rotulo, valor, detalhe]) => (
					<div key={rotulo}>
						<dt>{rotulo}</dt>
						<dd>{valor}</dd>
						<p>{detalhe}</p>
					</div>
				))}
			</dl>
			<section className="painel-sessao">
				<h2>Resultados da semana</h2>
				<dl className="resultados-semanais">
					{[
						["Rejeitadas", dados.rejeitadas],
						["Falhas", dados.falhas],
						["Exigiram aprovação humana", dados.exigiram_aprovacao_humana],
						["Risco médio", dados.risco_medio],
						["PRs revertidos no período", dados.revertidas],
					].map(([nome, n]) => (
						<div key={nome}>
							<dt>{nome}</dt>
							<dd>{n}</dd>
						</div>
					))}
				</dl>
			</section>
			<section className="painel-sessao">
				<h2>Escopo consultado</h2>
				<div>
					<p className="apoio">
						Instalação {dados.installation_id} · {dados.repositorios.length} repositórios autorizados na consulta
					</p>
					<ul>
						{dados.repositorios.map((r) => (
							<li className="caminho" key={r}>
								{r}
							</li>
						))}
					</ul>
				</div>
			</section>
			<p className="apoio">Fila atual, série diária, tempo médio e indicadores de 30 dias ainda não estão disponíveis.</p>
		</>
	);
}
