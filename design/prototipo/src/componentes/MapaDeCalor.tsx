import { type CSSProperties, type KeyboardEvent, useId, useLayoutEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { cn } from "@/lib/classes";
import { diaEMes, nomeDoNivel, plural } from "@/lib/formatos";
import type { Nivel } from "@/lib/tipos";
import type { DiaDoPeriodo } from "@/lib/visao-geral";
import { ListaDaCelula } from "./ListaDaCelula";

const niveisDoMaiorParaOMenor: Nivel[] = ["alto", "medio", "baixo"];
const forcaDoPasso = [0, 76, 88, 100];
const rotuloDoPasso = ["0", "1", "2", "3+"];
const ultimoPasso = forcaDoPasso.length - 1;

interface Celula {
	linha: number;
	coluna: number;
}

const deslocamentos: Record<string, Celula> = {
	ArrowRight: { linha: 0, coluna: 1 },
	ArrowLeft: { linha: 0, coluna: -1 },
	ArrowUp: { linha: -1, coluna: 0 },
	ArrowDown: { linha: 1, coluna: 0 },
};

function passoDaContagem(quantas: number) {
	return Math.min(quantas, ultimoPasso);
}

function tintaDoPasso(passo: number, cor: string): CSSProperties {
	if (passo === 0) return { background: "var(--fundo-elevado)" };
	return { background: `color-mix(in srgb, ${cor} ${forcaDoPasso[passo]}%, var(--fundo-superficie))` };
}

function descricaoDaCelula(dia: DiaDoPeriodo, nivel: Nivel) {
	return `${diaEMes(dia.dia)}, ${nomeDoNivel[nivel]}: ${plural(dia[nivel], "análise", "análises")}`;
}

function ancoraDaDica(coluna: number, colunas: number) {
	if (coluna < 4) return "left-0";
	if (coluna > colunas - 5) return "right-0";
	return "left-1/2 -translate-x-1/2";
}

function idDaCelula(prefixo: string, celula: Celula) {
	return `${prefixo}-${celula.linha}-${celula.coluna}`;
}

function limitar(valor: number, maximo: number) {
	return Math.min(maximo, Math.max(0, valor));
}

function mesmaCelula(a: Celula | null, b: Celula) {
	return a?.linha === b.linha && a.coluna === b.coluna;
}

export function EscalaDoMapa() {
	return (
		<div data-testid="escala-do-mapa" className="flex items-center gap-2">
			{forcaDoPasso.map((_, passo) => (
				<span key={rotuloDoPasso[passo]} className="inline-flex items-center gap-1">
					<span
						aria-hidden="true"
						data-testid="passo-da-escala"
						className="size-2.5 rounded-[2px] border border-borda"
						style={tintaDoPasso(passo, "var(--texto-secundario)")}
					/>
					{rotuloDoPasso[passo]}
				</span>
			))}
			<span>por dia</span>
		</div>
	);
}

interface PropsDaDica {
	id: string;
	texto: string;
	celula: Celula;
	colunas: number;
}

function Dica({ id, texto, celula, colunas }: PropsDaDica) {
	return (
		<span
			id={id}
			role="tooltip"
			className={cn(
				"pointer-events-none absolute z-10 rounded-selo border border-borda bg-elevado px-2 py-1 text-12 font-normal whitespace-nowrap text-titulo",
				celula.linha === 0 ? "top-full mt-1.5" : "bottom-full mb-1.5",
				ancoraDaDica(celula.coluna, colunas),
			)}
		>
			{texto}
		</span>
	);
}

const coberturaDaCelula = "absolute inset-0 cursor-pointer rounded-[2px]";

interface PropsDoDestino {
	dia: DiaDoPeriodo;
	nivel: Nivel;
	aberta: boolean;
	aoMudar: (aberta: boolean) => void;
	aoFechar: () => void;
}

function DestinoDaCelula({ dia, nivel, aberta, aoMudar, aoFechar }: PropsDoDestino) {
	const analises = dia.analises[nivel];
	const rotulo = descricaoDaCelula(dia, nivel);
	const [unica] = analises;
	if (!unica) return null;

	if (analises.length === 1) {
		return <Link to={`/analises/${unica.id}`} tabIndex={-1} aria-label={`Abrir a análise de ${rotulo}`} className={coberturaDaCelula} />;
	}

	return (
		<ListaDaCelula rotulo={rotulo} analises={analises} aberta={aberta} aoMudar={aoMudar} aoFechar={aoFechar}>
			<button type="button" tabIndex={-1} aria-label={`Ver as ${analises.length} análises de ${rotulo}`} className={coberturaDaCelula} />
		</ListaDaCelula>
	);
}

interface PropsDaLinha {
	nivel: Nivel;
	linha: number;
	dias: DiaDoPeriodo[];
	prefixo: string;
	ativa: Celula | null;
	destacada: Celula | null;
	aberta: Celula | null;
	idDaDica: string;
	aoApontar: (celula: Celula) => void;
	aoAbrir: (celula: Celula | null) => void;
	aoFecharLista: () => void;
}

function LinhaDoNivel({
	nivel,
	linha,
	dias,
	prefixo,
	ativa,
	destacada,
	aberta,
	idDaDica,
	aoApontar,
	aoAbrir,
	aoFecharLista,
}: PropsDaLinha) {
	const total = dias.reduce((soma, dia) => soma + dia[nivel], 0);

	return (
		<tr data-testid="linha-do-mapa">
			<th scope="row" className="sticky left-0 z-[1] bg-superficie pr-2 text-left text-12 font-normal text-secundario">
				{nomeDoNivel[nivel]}
				<span className="sr-only">, {plural(total, "análise", "análises")} no período</span>
			</th>
			{dias.map((dia, coluna) => {
				const quantas = dia[nivel];
				const passo = passoDaContagem(quantas);
				const celula = { linha, coluna };
				const comDica = mesmaCelula(destacada, celula) && !aberta;
				return (
					<td
						key={dia.dia.toISOString()}
						id={idDaCelula(prefixo, celula)}
						data-testid="celula-do-mapa"
						data-quantas={quantas}
						data-passo={passo}
						data-forca={forcaDoPasso[passo]}
						aria-label={descricaoDaCelula(dia, nivel)}
						aria-describedby={comDica ? idDaDica : undefined}
						className={cn(
							"transicao-de-estado relative h-6 min-w-6 rounded-[2px] p-0 text-center text-[11px] leading-none font-semibold text-superficie",
							quantas > 0 && "hover:outline-1 hover:outline-offset-1 hover:outline-[var(--texto-titulo)]",
							mesmaCelula(ativa, celula) && "outline-2 outline-offset-1 outline-acento",
						)}
						style={tintaDoPasso(passo, `var(--risco-${nivel})`)}
						onMouseEnter={() => aoApontar(celula)}
					>
						<span aria-hidden="true">{quantas >= 2 ? quantas : ""}</span>
						<DestinoDaCelula
							dia={dia}
							nivel={nivel}
							aberta={mesmaCelula(aberta, celula)}
							aoMudar={(abrir) => aoAbrir(abrir ? celula : null)}
							aoFechar={aoFecharLista}
						/>
						{comDica ? <Dica id={idDaDica} texto={descricaoDaCelula(dia, nivel)} celula={celula} colunas={dias.length} /> : null}
					</td>
				);
			})}
			<td aria-hidden="true" className="pl-2 text-right text-12">
				{total}
			</td>
		</tr>
	);
}

function EixoDosDias({ dias }: { dias: DiaDoPeriodo[] }) {
	const primeiro = dias[0];
	const doMeio = dias[Math.floor(dias.length / 2)];

	return (
		<div aria-hidden="true" className="mt-1 mr-[34px] ml-[50px] flex justify-between text-12 text-secundario">
			<span>{primeiro ? diaEMes(primeiro.dia) : null}</span>
			<span>{doMeio ? diaEMes(doMeio.dia) : null}</span>
			<span>hoje</span>
		</div>
	);
}

interface Props {
	dias: DiaDoPeriodo[];
	legenda: string;
	className?: string;
}

export function MapaDeCalor({ dias, legenda, className }: Props) {
	const prefixo = useId();
	const idDaDica = useId();
	const navegar = useNavigate();
	const rolagem = useRef<HTMLDivElement>(null);
	const grade = useRef<HTMLTableElement>(null);
	const [foco, setFoco] = useState<Celula>({ linha: 0, coluna: dias.length - 1 });
	const [focada, setFocada] = useState(false);
	const [apontada, setApontada] = useState<Celula | null>(null);
	const [aberta, setAberta] = useState<Celula | null>(null);
	const [dicaFechada, setDicaFechada] = useState(false);
	const destacada = apontada ?? (focada && !dicaFechada ? foco : null);

	useLayoutEffect(() => {
		const caixa = rolagem.current;
		if (caixa) caixa.scrollLeft = caixa.scrollWidth;
	}, []);

	function destino(tecla: string): Celula | null {
		if (tecla === "Home") return { linha: foco.linha, coluna: 0 };
		if (tecla === "End") return { linha: foco.linha, coluna: dias.length - 1 };
		const passo = deslocamentos[tecla];
		if (!passo) return null;
		return {
			linha: limitar(foco.linha + passo.linha, niveisDoMaiorParaOMenor.length - 1),
			coluna: limitar(foco.coluna + passo.coluna, dias.length - 1),
		};
	}

	function abrirCelula(celula: Celula) {
		const dia = dias[celula.coluna];
		const nivel = niveisDoMaiorParaOMenor[celula.linha];
		if (!dia || !nivel) return;
		const analises = dia.analises[nivel];
		const [unica] = analises;
		if (!unica) return;
		if (analises.length === 1) navegar(`/analises/${unica.id}`);
		else setAberta(celula);
	}

	function aoTeclar(evento: KeyboardEvent<HTMLTableElement>) {
		if (evento.key === "Escape") {
			setDicaFechada(true);
			return;
		}
		if (evento.key === "Enter" || evento.key === " ") {
			evento.preventDefault();
			abrirCelula(foco);
			return;
		}
		const proxima = destino(evento.key);
		if (!proxima) return;
		evento.preventDefault();
		setFoco(proxima);
		setApontada(null);
		setDicaFechada(false);
		document.getElementById(idDaCelula(prefixo, proxima))?.scrollIntoView({ block: "nearest", inline: "nearest" });
	}

	return (
		<figure className={cn("m-0 flex min-w-0 flex-col justify-center", className)}>
			<div ref={rolagem} className="overflow-x-auto pt-2 pb-1">
				<div className="min-w-[900px]">
					<table
						ref={grade}
						role="grid"
						tabIndex={0}
						aria-label="Análises por dia e nível de risco. Use as setas para percorrer os dias e os níveis e Enter para abrir."
						aria-activedescendant={focada ? idDaCelula(prefixo, foco) : undefined}
						className="w-full table-fixed border-separate border-spacing-0.5 rounded-selo"
						onKeyDown={aoTeclar}
						onFocus={() => {
							setFocada(true);
							setDicaFechada(false);
						}}
						onBlur={() => setFocada(false)}
						onMouseLeave={() => setApontada(null)}
					>
						<caption className="sr-only">{legenda}</caption>
						<colgroup>
							<col className="w-12" />
							{dias.map((dia) => (
								<col key={dia.dia.toISOString()} />
							))}
							<col className="w-8" />
						</colgroup>
						<tbody>
							{niveisDoMaiorParaOMenor.map((nivel, linha) => (
								<LinhaDoNivel
									key={nivel}
									nivel={nivel}
									linha={linha}
									dias={dias}
									prefixo={prefixo}
									ativa={focada ? foco : null}
									destacada={destacada}
									aberta={aberta}
									idDaDica={idDaDica}
									aoApontar={setApontada}
									aoAbrir={setAberta}
									aoFecharLista={() => grade.current?.focus()}
								/>
							))}
						</tbody>
					</table>
					<EixoDosDias dias={dias} />
				</div>
			</div>
		</figure>
	);
}
