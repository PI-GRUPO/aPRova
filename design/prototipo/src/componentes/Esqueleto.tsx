import { cn } from "@/lib/classes";

export function Esqueleto({ className }: { className?: string }) {
	return <div className={cn("rounded-selo bg-elevado", className)} />;
}

export function LinhasDeEsqueleto({ quantas, className }: { quantas: number; className?: string }) {
	const linhas = Array.from({ length: quantas }, (_, indice) => `linha-${indice}`);

	return (
		<div className={cn("divide-y divide-borda rounded-controle border border-borda bg-superficie", className)}>
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
