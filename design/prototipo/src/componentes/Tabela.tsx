import { ArrowUp } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/classes";

export type Sentido = "crescente" | "decrescente";

const sentidoParaLeitores: Record<Sentido, "ascending" | "descending"> = {
	crescente: "ascending",
	decrescente: "descending",
};

interface PropsComFilhos {
	children: ReactNode;
	className?: string;
}

export function Tabela({ legenda, children, className }: PropsComFilhos & { legenda: string }) {
	return (
		<table className={cn("w-full border-collapse text-left", className)}>
			<caption className="sr-only">{legenda}</caption>
			{children}
		</table>
	);
}

export function CabecalhoDaTabela({ children }: { children: ReactNode }) {
	return (
		<thead>
			<tr className="border-b border-borda bg-elevado">{children}</tr>
		</thead>
	);
}

export function CorpoDaTabela({ children }: { children: ReactNode }) {
	return <tbody>{children}</tbody>;
}

export function LinhaDaTabela({ children, className }: PropsComFilhos) {
	return (
		<tr className={cn("transicao-de-estado relative border-b border-borda align-baseline last:border-b-0 hover:bg-elevado", className)}>
			{children}
		</tr>
	);
}

export function Coluna({ children, className }: PropsComFilhos) {
	return (
		<th scope="col" className={cn("px-3 py-1.5 md:max-lg:px-2 text-12 font-medium whitespace-nowrap text-secundario", className)}>
			{children}
		</th>
	);
}

interface PropsDaColunaOrdenavel extends PropsComFilhos {
	sentido: Sentido | null;
	aoOrdenar: () => void;
	aDireita?: boolean;
}

export function ColunaOrdenavel({ children, sentido, aoOrdenar, aDireita = false, className }: PropsDaColunaOrdenavel) {
	return (
		<th scope="col" aria-sort={sentido ? sentidoParaLeitores[sentido] : "none"} className={cn("p-0 text-12 font-medium", className)}>
			<button
				type="button"
				onClick={aoOrdenar}
				className={cn(
					"transicao-de-estado flex w-full cursor-pointer items-center gap-1 px-3 py-1.5 md:max-lg:px-2 whitespace-nowrap -outline-offset-2 hover:text-primario",
					aDireita && "flex-row-reverse",
					sentido ? "text-primario" : "text-secundario",
				)}
			>
				{children}
				<ArrowUp
					aria-hidden="true"
					size={14}
					strokeWidth={1.75}
					className={cn("transicao-de-estado", sentido === "decrescente" && "rotate-180", !sentido && "opacity-0")}
				/>
			</button>
		</th>
	);
}

export function Celula({ children, className }: PropsComFilhos) {
	return <td className={cn("px-3 py-2 whitespace-nowrap md:max-lg:px-2", className)}>{children}</td>;
}
