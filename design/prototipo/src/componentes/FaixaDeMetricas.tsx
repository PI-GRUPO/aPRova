import type { ReactNode } from "react";
import { Esqueleto } from "./Esqueleto";
import { Proporcao } from "./Proporcao";

interface Fracao {
	parte: number;
	todo: number;
}

interface PropsDaMetrica {
	rotulo: string;
	valor: string;
	unidade?: string;
	fracao?: Fracao;
	apoio: ReactNode;
}

export function Metrica({ rotulo, valor, unidade, fracao, apoio }: PropsDaMetrica) {
	return (
		<div data-testid="metrica" className="flex min-w-0 flex-col px-4 py-3 xl:px-5 xl:py-5">
			<dt className="text-12 font-medium text-primario">{rotulo}</dt>
			<dd className="mt-1 flex flex-wrap items-baseline gap-x-2">
				<span data-testid="metrica-valor" className="text-32 font-semibold text-titulo tabular-nums">
					{valor}
				</span>
				{unidade ? <span className="whitespace-nowrap text-secundario">{unidade}</span> : null}
			</dd>
			{fracao ? (
				<dd className="mt-2">
					<Proporcao parte={fracao.parte} todo={fracao.todo} />
				</dd>
			) : null}
			<dd className="mt-2 text-12 text-secundario">{apoio}</dd>
		</div>
	);
}

export function FaixaDeMetricas({ rotulo, children }: { rotulo: string; children: ReactNode }) {
	return (
		<section aria-label={rotulo}>
			<dl
				data-testid="faixa-de-metricas"
				className="emenda grid grid-cols-2 overflow-hidden rounded-painel border border-borda xl:grid-cols-4"
			>
				{children}
			</dl>
		</section>
	);
}

const celulasDoEsqueleto = ["primeira", "segunda", "terceira", "quarta"];

export function EsqueletoDaFaixa() {
	return (
		<div
			data-testid="esqueleto-da-faixa"
			className="emenda grid grid-cols-2 overflow-hidden rounded-painel border border-borda xl:grid-cols-4"
		>
			{celulasDoEsqueleto.map((celula) => (
				<div key={celula} className="flex flex-col px-4 py-3 xl:px-5 xl:py-5">
					<Esqueleto className="my-0.5 h-3 w-28" />
					<Esqueleto className="mt-1.5 h-9 w-20" />
					<Esqueleto className="mt-2 h-1.5 w-full" />
					<Esqueleto className="mt-2.5 h-3 w-40 max-w-full" />
					<Esqueleto className="mt-1.5 h-3 w-32 max-w-full" />
				</div>
			))}
		</div>
	);
}
