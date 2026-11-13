import * as Primitiva from "@radix-ui/react-popover";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { nomeCurtoDoRepositorio } from "@/lib/dados";
import { horaDoDia } from "@/lib/formatos";
import type { Analise } from "@/lib/tipos";

interface Props {
	rotulo: string;
	analises: Analise[];
	aberta: boolean;
	aoMudar: (aberta: boolean) => void;
	aoFechar: () => void;
	children: ReactNode;
}

export function ListaDaCelula({ rotulo, analises, aberta, aoMudar, aoFechar, children }: Props) {
	return (
		<Primitiva.Root open={aberta} onOpenChange={aoMudar}>
			<Primitiva.Trigger asChild>{children}</Primitiva.Trigger>
			<Primitiva.Portal>
				<Primitiva.Content
					aria-label={rotulo}
					side="top"
					sideOffset={6}
					collisionPadding={12}
					onCloseAutoFocus={(evento) => {
						evento.preventDefault();
						aoFechar();
					}}
					className="z-30 w-64 rounded-controle border border-borda bg-elevado p-1 text-14 text-primario"
				>
					<p className="px-2 pt-1 pb-1.5 text-12 text-secundario">{rotulo}</p>
					<ul>
						{analises.map((analise) => (
							<li key={analise.id}>
								<Link
									to={`/analises/${analise.id}`}
									className="flex h-8 items-center justify-between gap-3 rounded-selo px-2 no-underline hover:bg-superficie"
								>
									<span className="truncate">
										{nomeCurtoDoRepositorio(analise.repository)} <span className="font-mono">#{analise.pr_number}</span>
									</span>
									<span className="shrink-0 text-12 text-secundario">{horaDoDia(new Date(analise.received_at))}</span>
								</Link>
							</li>
						))}
					</ul>
				</Primitiva.Content>
			</Primitiva.Portal>
		</Primitiva.Root>
	);
}
