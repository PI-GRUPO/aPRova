import { type CSSProperties, type KeyboardEvent, useId, useState } from "react";
import { cn } from "@/lib/classes";
import { diaEMes, nomeDoNivel, plural } from "@/lib/formatos";
import type { Nivel } from "@/lib/tipos";
import type { DiaDoPeriodo } from "@/lib/visao-geral";

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
	linha: number;
	coluna: number;
	colunas: number;
}

function Dica({ id, texto, linha, coluna, colunas }: PropsDaDica) {
	return (
		<span
			id={id}
			role="tooltip"
			className={cn(
				"pointer-events-none absolute z-10 rounded-selo border border-borda bg-elevado px-2 py-1 text-12 font-normal whitespace-nowrap text-titulo",
				linha === 0 ? "top-full mt-1.5" : "bottom-full mb-1.5",
				ancoraDaDica(coluna, colunas),
			)}
		>
			{texto}
		</span>
	);
}

interface PropsDaLinha {
	nivel: Nivel;
	linha: number;
	dias: DiaDoPeriodo[];
	prefixo: string;
	ativa: Celula | null;
	destacada: Celula | null;
	idDaDica: string;
	aoApontar: (celula: Celula) => void;
}

function LinhaDoNivel({ nivel, linha, dias, prefixo, ativa, destacada, idDaDica, aoApontar }: PropsDaLinha) {
	const total = dias.reduce((soma, dia) => soma + dia[nivel], 0);

	return (
		<tr data-testid="linha-do-mapa">
			<th scope="row" className="pr-2 text-left text-12 font-normal text-secundario">
				{nomeDoNivel[nivel]}
				<span className="sr-only">, {plural(total, "análise", "análises")} no período</span>
			</th>
			{dias.map((dia, coluna) => {
				const quantas = dia[nivel];
				const passo = passoDaContagem(quantas);
				const comDica = destacada?.linha === linha && destacada.coluna === coluna;
				const comFoco = ativa?.linha === linha && ativa.coluna === coluna;
				return (
					<td
						key={dia.dia.toISOString()}
						id={idDaCelula(prefixo, { linha, coluna })}
						data-testid="celula-do-mapa"
						data-quantas={quantas}
						data-passo={passo}
						data-forca={forcaDoPasso[passo]}
						aria-label={descricaoDaCelula(dia, nivel)}
						aria-describedby={comDica ? idDaDica : undefined}
						className={cn(
							"relative h-5 rounded-[2px] p-0 text-center text-[11px] leading-none font-semibold text-superficie",
							comFoco && "outline-2 outline-offset-1 outline-acento",
						)}
						style={tintaDoPasso(passo, `var(--risco-${nivel})`)}
						onMouseEnter={() => aoApontar({ linha, coluna })}
					>
						<span aria-hidden="true" className="max-md:hidden">
							{quantas >= 2 ? quantas : ""}
						</span>
						{comDica ? (
							<Dica id={idDaDica} texto={descricaoDaCelula(dia, nivel)} linha={linha} coluna={coluna} colunas={dias.length} />
						) : null}
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
	const [foco, setFoco] = useState<Celula>({ linha: 0, coluna: dias.length - 1 });
	const [focada, setFocada] = useState(false);
	const [apontada, setApontada] = useState<Celula | null>(null);
	const [dicaFechada, setDicaFechada] = useState(false);
	const destacada = apontada ?? (focada && !dicaFechada ? foco : null);

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

	function aoTeclar(evento: KeyboardEvent<HTMLTableElement>) {
		if (evento.key === "Escape") {
			setDicaFechada(true);
			return;
		}
		const proxima = destino(evento.key);
		if (!proxima) return;
		evento.preventDefault();
		setFoco(proxima);
		setApontada(null);
		setDicaFechada(false);
	}

	return (
		<figure className={cn("m-0 flex flex-col justify-center", className)}>
			<table
				role="grid"
				tabIndex={0}
				aria-label="Análises por dia e nível de risco. Use as setas para percorrer os dias e os níveis."
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
							idDaDica={idDaDica}
							aoApontar={setApontada}
						/>
					))}
				</tbody>
			</table>
			<EixoDosDias dias={dias} />
		</figure>
	);
}
