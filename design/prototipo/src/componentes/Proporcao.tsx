import { cn } from "@/lib/classes";

interface Props {
	parte: number;
	todo: number;
	className?: string;
}

export function Proporcao({ parte, todo, className }: Props) {
	const largura = todo === 0 ? 0 : Math.min(100, (parte / todo) * 100);

	return (
		<span aria-hidden="true" data-testid="proporcao" className={cn("trilha block h-1.5 w-full overflow-hidden rounded-[2px]", className)}>
			<span className="block h-full bg-secundario" style={{ width: `${largura}%` }} />
		</span>
	);
}
