import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { consultarRecorteMapa, type NivelMapa, nivelMapaValido, type RecorteMapa, rotuloDiaMapa } from "@/servicos/mapa";
import { destinoAnalise, type VisaoGeralDados } from "@/servicos/visao-geral";
import { useEscopo } from "./ConsultaPrivada";
import { Botao, Modal } from "./Primitivos";
import "@/estilos/mapa.css";

function ListaDoRecorte({ data, nivel }: { data: string; nivel: NivelMapa }) {
	const { instalacao } = useEscopo("");
	const [recorte, setRecorte] = useState<RecorteMapa>();
	const [erro, setErro] = useState(false);
	const [q, setQ] = useSearchParams();
	const repo = q.get("repositorio") ?? "";
	const pagina = __MOCKS__ ? 1 : Number(q.get("mapa_pagina") ?? 1);
	const snapshot = __MOCKS__ ? "" : (q.get("mapa_snapshot") ?? "");
	function paginar(numero: number, corte: string) {
		setQ((anterior) => {
			const nova = new URLSearchParams(anterior);
			nova.set("mapa_pagina", String(numero));
			nova.set("mapa_snapshot", corte);
			return nova;
		});
	}
	useEffect(() => {
		const c = new AbortController();
		setRecorte(undefined);
		setErro(false);
		void consultarRecorteMapa(instalacao, data, nivel, c.signal, repo, pagina, snapshot)
			.then((v) => {
				if (!c.signal.aborted) setRecorte(v);
			})
			.catch(() => {
				if (!c.signal.aborted) setErro(true);
			});
		return () => {
			c.abort();
		};
	}, [instalacao, data, nivel, repo, pagina, snapshot]);
	const retorno = new URLSearchParams();
	for (const nome of [
		"installation_id",
		"fila",
		"repositorio",
		"mapa_dia",
		"mapa_nivel",
		"mapa_pagina",
		"mapa_snapshot",
		"pagina",
		"snapshot",
	]) {
		const v = q.get(nome);
		if (v) retorno.set(nome, v);
	}
	if (erro)
		return (
			<div role="alert">
				<p>Não foi possível carregar as referências deste recorte.</p>
				<Botao
					onClick={() => {
						window.location.reload();
					}}
				>
					Tentar novamente
				</Botao>
			</div>
		);
	if (!recorte) return <p role="status">Carregando análises…</p>;
	if (!recorte.itens.length && pagina === 1) return <p role="status">Nenhuma análise neste recorte.</p>;
	return (
		<nav className="lista-recorte-mapa" aria-label="Análises deste dia e nível">
			{recorte.itens.map((item) => (
				<Link
					className="botao"
					key={item.id}
					to={
						recorte.origem === "mock"
							? destinoAnalise(item.id, recorte.origem, instalacao, retorno)
							: `/analises/${encodeURIComponent(item.id)}?${new URLSearchParams({ installation_id: instalacao, repositorio: item.repositorio, origem: "fila", retorno: retorno.toString() })}`
					}
				>
					<span>
						{item.repositorio} #{item.pr}
						<span className="horario-recorte">
							{" "}
							· {rotuloDiaMapa(data)} ·{" "}
							{new Date(item.avaliadoEm).toLocaleTimeString("pt-BR", { timeZone: "UTC", hour: "2-digit", minute: "2-digit" })}
						</span>
					</span>
				</Link>
			))}
			{(pagina > 1 || recorte.temMais) && (
				<div className="controles">
					<Botao disabled={pagina === 1} onClick={() => paginar(pagina - 1, recorte.snapshot ?? "")}>
						Anterior
					</Botao>
					<Botao disabled={!recorte.temMais || pagina >= 201} onClick={() => paginar(pagina + 1, recorte.snapshot ?? "")}>
						Próxima
					</Botao>
				</div>
			)}
		</nav>
	);
}
function CelulaMapa({
	data,
	nivel,
	quantidade,
	rotulo,
	aberto,
	aoMudar,
}: {
	data: string;
	nivel: NivelMapa;
	quantidade: number;
	rotulo: string;
	aberto: boolean;
	aoMudar: (aberto: boolean) => void;
}) {
	return (
		<Modal
			className="modal-mapa"
			aberto={aberto}
			aoMudar={aoMudar}
			titulo={`${rotuloDiaMapa(data)} · risco ${nivel === "medio" ? "médio" : nivel}`}
			descricao={`${__MOCKS__ ? "Dados simulados" : "Dados da API"} · ${quantidade} ${quantidade === 1 ? "análise" : "análises"} neste dia e nível`}
			gatilho={
				<button
					type="button"
					className={`celula-mapa intensidade-${Math.min(quantidade, 3)} nivel-${nivel}`}
					data-dia={data}
					data-nivel={nivel}
					aria-label={rotulo}
				>
					{quantidade >= 2 ? quantidade : ""}
				</button>
			}
		>
			{aberto && <ListaDoRecorte key={`${data}:${nivel}`} data={data} nivel={nivel} />}
			<div className="rodape-recorte">
				<Botao
					variante="primario"
					aria-label="Fechar recorte"
					onClick={() => {
						aoMudar(false);
					}}
				>
					Fechar
				</Botao>
			</div>
		</Modal>
	);
}

export function MapaDeAnalises({ dias }: { dias: VisaoGeralDados["mapa"] }) {
	const [q, setQ] = useSearchParams();
	const dataSelecionada = q.get("mapa_dia");
	const nivelSelecionado = q.get("mapa_nivel");
	const invalido =
		(dataSelecionada !== null || nivelSelecionado !== null) &&
		(!dias.some((d) => d.data === dataSelecionada) || !nivelMapaValido(nivelSelecionado));
	function selecionar(recorte?: Pick<RecorteMapa, "data" | "nivel">) {
		setQ(
			(anterior) => {
				const novo = new URLSearchParams(anterior);
				novo.delete("mapa_pagina");
				novo.delete("mapa_snapshot");
				if (recorte) {
					novo.set("mapa_dia", recorte.data);
					novo.set("mapa_nivel", recorte.nivel);
				} else {
					novo.delete("mapa_dia");
					novo.delete("mapa_nivel");
				}
				return novo;
			},
			{ replace: true },
		);
	}
	return (
		<section className="painel-overview">
			<h2>Análises por dia e nível</h2>
			<div className="conteudo-overview">
				{invalido && (
					<div role="status" className="recorte-invalido">
						<p>Recorte do mapa inválido. Escolha um dia e nível da janela apresentada.</p>
						<Botao
							onClick={() => {
								selecionar();
							}}
						>
							Limpar recorte
						</Botao>
					</div>
				)}
				<div className="legenda-mapa apoio">
					{__MOCKS__ ? "Dados simulados" : "Recebidas em UTC"} · por dia{" "}
					{["0", "1", "2", "3+"].map((v, i) => (
						<span key={v}>
							<i style={{ opacity: i ? 0.5 + i / 6 : 0.12 }} />
							{v}
						</span>
					))}
				</div>
				<div className="mapa-com-rotulos">
					<div className="rotulos-mapa">
						{["Alto", "Médio", "Baixo"].map((n) => (
							<span key={n}>{n}</span>
						))}
					</div>
					<section className="viewport-mapa" aria-label="Mapa de análises de 30 dias, rolagem horizontal">
						{(["alto", "medio", "baixo"] as const).map((nivel) => (
							<div className="linha-mapa" key={nivel}>
								{dias.map((dia) => {
									const n = nivel === "alto" ? dia.alto : nivel === "medio" ? dia.medio : dia.baixo;
									const data = new Date(`${dia.data}T12:00:00Z`).toLocaleDateString("pt-BR", {
										day: "numeric",
										month: "short",
										timeZone: "UTC",
									});
									const rotulo = `${data} · risco ${nivel === "medio" ? "médio" : nivel}: ${n} análise(s)`;
									return (
										<CelulaMapa
											key={dia.data}
											data={dia.data}
											nivel={nivel}
											quantidade={n}
											rotulo={rotulo}
											aberto={dataSelecionada === dia.data && nivelSelecionado === nivel}
											aoMudar={(aberto) => {
												selecionar(aberto ? { data: dia.data, nivel } : undefined);
											}}
										/>
									);
								})}
							</div>
						))}
						<div className="datas-mapa apoio">
							{dias
								.filter((_, i) => i === dias.length - 1 || (i % 7 === 0 && i + 7 < dias.length))
								.map((dia) => (
									<span key={dia.data}>{rotuloDiaMapa(dia.data)}</span>
								))}
						</div>
					</section>
				</div>
				<p className="apoio nota-mapa">
					Cada célula é um dia. Clique para ver as análises. Zero é neutro; a intensidade indica a quantidade.
				</p>
			</div>
		</section>
	);
}
