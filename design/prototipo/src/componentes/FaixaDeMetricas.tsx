import type { ReactNode } from "react";
import { Proporcao } from "./Proporcao";

interface PropsDaMetrica {
	rotulo: string;
	valor: string;
	unidade: string;
	parte: number;
	todo: number;
	apoio: ReactNode;
}

export function Metrica({ rotulo, valor, unidade, parte, todo, apoio }: PropsDaMetrica) {
	return (
		<div data-testid="metrica" className="flex min-w-0 flex-col px-4 py-3 xl:px-5 xl:py-4">
			<dt className="text-12 font-medium text-secundario">{rotulo}</dt>
			<dd className="mt-1 flex flex-wrap items-baseline gap-x-2">
				<span data-testid="metrica-valor" className="text-32 font-semibold tabular-nums">
					{valor}
				</span>
				<span className="whitespace-nowrap text-secundario">{unidade}</span>
			</dd>
			<dd className="mt-2">
				<Proporcao parte={parte} todo={todo} />
			</dd>
			<dd className="mt-2 text-12 text-secundario">{apoio}</dd>
		</div>
	);
}

export function FaixaDeMetricas({ rotulo, children }: { rotulo: string; children: ReactNode }) {
	return (
		<section aria-label={rotulo}>
			<dl
				data-testid="faixa-de-metricas"
				className="emenda grid grid-cols-2 overflow-hidden rounded-controle border border-borda xl:grid-cols-4"
			>
				{children}
			</dl>
		</section>
	);
}
