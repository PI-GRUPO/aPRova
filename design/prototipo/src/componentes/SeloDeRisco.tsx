import type { CSSProperties } from "react";
import { cn } from "@/lib/classes";
import { nomeDoNivel } from "@/lib/formatos";
import type { Nivel } from "@/lib/tipos";

type Variante = "ponto" | "tingido";

const textoDoNivel: Record<Nivel, string> = {
	baixo: "text-risco-baixo",
	medio: "text-risco-medio",
	alto: "text-risco-alto",
};

const fundoDoNivel: Record<Nivel, string> = {
	baixo: "bg-risco-baixo",
	medio: "bg-risco-medio",
	alto: "bg-risco-alto",
};

const formaDaVariante: Record<Variante, string> = {
	ponto: "gap-1.5",
	tingido: "selo-tingido h-5 gap-1.5 rounded-selo px-1.5 text-primario",
};

interface Props {
	nivel: Nivel;
	variante?: Variante;
	comPrefixo?: boolean;
	explicacao?: string;
	className?: string;
}

export function SeloDeRisco({ nivel, variante = "tingido", comPrefixo = false, explicacao, className }: Props) {
	return (
		<span
			data-testid={`selo-risco-${nivel}`}
			title={explicacao}
			className={cn(
				"inline-flex items-center text-12 font-medium whitespace-nowrap",
				formaDaVariante[variante],
				variante === "ponto" && textoDoNivel[nivel],
				className,
			)}
			style={{ "--cor-do-selo": `var(--risco-${nivel})` } as CSSProperties}
		>
			<span aria-hidden="true" className={cn("size-1.5 rounded-full", fundoDoNivel[nivel])} />
			<span>
				<span className={comPrefixo ? undefined : "sr-only"}>risco </span>
				{nomeDoNivel[nivel]}
			</span>
		</span>
	);
}
