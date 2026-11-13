import { type ReactNode, useId } from "react";
import { cn } from "@/lib/classes";
import { Proporcao } from "./Proporcao";

interface Props {
	titulo: string;
	resumo: ReactNode;
	children: ReactNode;
	className?: string;
}

export function Painel({ titulo, resumo, children, className }: Props) {
	const idDoTitulo = useId();

	return (
		<section
			aria-labelledby={idDoTitulo}
			data-testid="painel-do-periodo"
			className={cn("flex flex-col overflow-hidden rounded-controle border border-borda bg-superficie", className)}
		>
			<header className="flex flex-wrap items-baseline justify-between gap-x-4 border-b border-borda bg-elevado px-3 py-1.5 whitespace-nowrap">
				<h2 id={idDoTitulo} className="text-14 font-semibold">
					{titulo}
				</h2>
				<div className="text-12 text-secundario">{resumo}</div>
			</header>
			{children}
		</section>
	);
}

export function TabelaDeContagem({ legenda, children }: { legenda: string; children: ReactNode }) {
	return (
		<table className="h-full w-full flex-1 border-collapse">
			<caption className="sr-only">{legenda}</caption>
			<tbody>{children}</tbody>
		</table>
	);
}

export function LinhaDeContagem({ rotulo, parte, todo }: { rotulo: string; parte: number; todo: number }) {
	return (
		<tr className="border-b border-borda last:border-b-0">
			<th scope="row" className="px-3 py-1.5 text-left font-normal whitespace-nowrap">
				{rotulo}
			</th>
			<td className="w-full px-3">
				<Proporcao parte={parte} todo={todo} />
			</td>
			<td className="px-3 py-1.5 text-right whitespace-nowrap">{parte}</td>
		</tr>
	);
}
