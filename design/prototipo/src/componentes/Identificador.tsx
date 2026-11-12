import type { ReactNode } from "react";
import { cn } from "@/lib/classes";

export function Identificador({ children, className }: { children: ReactNode; className?: string }) {
	return <span className={cn("font-mono", className)}>{children}</span>;
}
