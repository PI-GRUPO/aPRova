import { Search, X } from "lucide-react";
import { useRef } from "react";
import { cn } from "@/lib/classes";

interface Props {
	rotulo: string;
	valor: string;
	aoMudar: (valor: string) => void;
	exemplo?: string;
	className?: string;
}

export function CampoDeBusca({ rotulo, valor, aoMudar, exemplo, className }: Props) {
	const campo = useRef<HTMLInputElement>(null);

	function limpar() {
		aoMudar("");
		campo.current?.focus();
	}

	return (
		<div className={cn("relative", className)}>
			<Search
				aria-hidden="true"
				size={16}
				strokeWidth={1.75}
				className={cn("transicao-de-estado pointer-events-none absolute top-2 left-2", valor ? "text-primario" : "text-secundario")}
			/>
			<input
				ref={campo}
				type="search"
				aria-label={rotulo}
				value={valor}
				onChange={(evento) => aoMudar(evento.target.value)}
				placeholder={exemplo}
				className="transicao-de-estado h-8 w-full rounded-controle border border-contorno bg-superficie pr-8 pl-8 text-14 text-primario placeholder:text-secundario hover:border-primario"
			/>
			{valor ? (
				<button
					type="button"
					onClick={limpar}
					aria-label="Limpar busca"
					className="transicao-de-estado absolute top-1 right-1 grid size-6 cursor-pointer place-items-center rounded-selo text-secundario hover:bg-elevado hover:text-primario"
				>
					<X aria-hidden="true" size={14} strokeWidth={1.75} />
				</button>
			) : null}
		</div>
	);
}
