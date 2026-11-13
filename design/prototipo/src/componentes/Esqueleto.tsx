import { cn } from "@/lib/classes";

export function Esqueleto({ className }: { className?: string }) {
	return <div className={cn("rounded-selo bg-elevado", className)} />;
}

export function LinhasDeEsqueleto({ quantas, className }: { quantas: number; className?: string }) {
	const linhas = Array.from({ length: quantas }, (_, indice) => `linha-${indice}`);

	return (
		<div className={cn("divide-y divide-borda rounded-painel border border-borda bg-superficie", className)}>
			{linhas.map((linha) => (
				<div key={linha} className="flex items-center gap-4 px-4 py-3">
					<Esqueleto className="h-4 w-16" />
					<Esqueleto className="h-4 w-48" />
					<Esqueleto className="ml-auto h-4 w-24" />
				</div>
			))}
		</div>
	);
}

export function EsqueletoDeTabela({ linhas, className }: { linhas: number; className?: string }) {
	const chaves = Array.from({ length: linhas }, (_, indice) => `linha-${indice}`);

	return (
		<div data-testid="esqueleto-da-fila" className={cn("overflow-hidden rounded-painel border border-borda bg-superficie", className)}>
			<div className="flex items-center gap-3 border-b border-borda px-3 py-2">
				<Esqueleto className="h-4 w-40" />
				<Esqueleto className="ml-auto h-8 w-56 max-md:hidden" />
				<Esqueleto className="h-8 w-52 max-md:hidden" />
			</div>
			<div className="flex gap-6 border-b border-borda bg-elevado px-3 py-1.5">
				<Esqueleto className="h-4 w-14 bg-superficie" />
				<Esqueleto className="h-4 w-24 bg-superficie" />
				<Esqueleto className="h-4 w-20 bg-superficie" />
			</div>
			{chaves.map((chave) => (
				<div key={chave} className="flex items-center gap-6 border-b border-borda px-3 py-2 last:border-b-0">
					<Esqueleto className="h-5 w-14" />
					<Esqueleto className="h-5 w-40" />
					<Esqueleto className="h-5 w-32 max-md:hidden" />
					<Esqueleto className="ml-auto h-5 w-48 max-md:hidden" />
				</div>
			))}
		</div>
	);
}
