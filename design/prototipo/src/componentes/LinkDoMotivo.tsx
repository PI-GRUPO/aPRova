import { Link } from "react-router";
import { cn } from "@/lib/classes";
import type { Analise } from "@/lib/tipos";

export function LinkDoMotivo({ id }: { id: string }) {
	return (
		<Link to={`/regras#${id}`} className="relative z-10 font-mono">
			{id}
		</Link>
	);
}

export function MotivosDaAnalise({ analise, className }: { analise: Analise; className?: string }) {
	return (
		<span className={cn("flex gap-x-3", className)}>
			{analise.risk_gate?.rules_hit.map((motivo) => (
				<LinkDoMotivo key={motivo.id} id={motivo.id} />
			))}
		</span>
	);
}
