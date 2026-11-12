import { cn } from "@/lib/classes";
import { diaEMes, plural } from "@/lib/formatos";
import type { Nivel } from "@/lib/tipos";
import type { DiaDoPeriodo } from "@/lib/visao-geral";

const niveisDeBaixoParaCima: Nivel[] = ["baixo", "medio", "alto"];

const fundoDoNivel: Record<Nivel, string> = {
	baixo: "bg-risco-baixo",
	medio: "bg-risco-medio",
	alto: "bg-risco-alto",
};

function totalDoDia(dia: DiaDoPeriodo) {
	return dia.alto + dia.medio + dia.baixo;
}

function TabelaDosDias({ dias, legenda }: { dias: DiaDoPeriodo[]; legenda: string }) {
	return (
		<div className="sr-only">
			<table>
				<caption>{legenda}</caption>
				<thead>
					<tr>
						<th scope="col">Dia</th>
						<th scope="col">Alto</th>
						<th scope="col">Médio</th>
						<th scope="col">Baixo</th>
					</tr>
				</thead>
				<tbody>
					{dias
						.filter((dia) => totalDoDia(dia) > 0)
						.map((dia) => (
							<tr key={dia.dia.toISOString()}>
								<th scope="row">{diaEMes(dia.dia)}</th>
								<td>{dia.alto}</td>
								<td>{dia.medio}</td>
								<td>{dia.baixo}</td>
							</tr>
						))}
				</tbody>
			</table>
		</div>
	);
}

function EixoDosDias({ dias }: { dias: DiaDoPeriodo[] }) {
	const primeiro = dias[0];
	const doMeio = dias[Math.floor(dias.length / 2)];

	return (
		<div className="mt-1.5 flex justify-between text-12 text-secundario">
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

export function BarrasPorDia({ dias, legenda, className }: Props) {
	const pico = Math.max(1, ...dias.map(totalDoDia));

	return (
		<figure className={cn("m-0 flex min-h-0 flex-col", className)}>
			<div aria-hidden="true" className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_auto] grid-rows-[minmax(0,1fr)_auto] gap-x-2">
				<div className="flex min-h-12 items-end gap-[3px] border-t border-b border-borda">
					{dias.map((dia) => (
						<div
							key={dia.dia.toISOString()}
							data-testid="dia-do-grafico"
							title={`${diaEMes(dia.dia)}: ${dia.alto} alto, ${dia.medio} médio, ${dia.baixo} baixo`}
							className="flex h-full min-w-0 flex-1 flex-col-reverse gap-px"
						>
							{niveisDeBaixoParaCima.map((nivel) =>
								dia[nivel] > 0 ? (
									<span key={nivel} className={fundoDoNivel[nivel]} style={{ height: `${(dia[nivel] / pico) * 100}%` }} />
								) : null,
							)}
						</div>
					))}
				</div>
				<div className="flex flex-col justify-between text-12 leading-none text-secundario">
					<span className="-translate-y-1/2">{pico}</span>
					<span className="translate-y-1/2">0</span>
				</div>
				<EixoDosDias dias={dias} />
			</div>
			<figcaption className="sr-only">
				Análises por dia de chegada do PR, com pico de {plural(pico, "análise", "análises")} num dia.
			</figcaption>
			<TabelaDosDias dias={dias} legenda={legenda} />
		</figure>
	);
}
