import type { ReactNode } from "react";
import { cn } from "@/lib/classes";

export function Identificador({ children, className, title }: { children: ReactNode; className?: string; title?: string }) {
	return (
		<span className={cn("font-mono", className)} title={title}>
			{children}
		</span>
	);
}
