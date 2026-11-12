import * as Primitiva from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/classes";

export interface OpcaoDaSelecao {
	valor: string;
	rotulo: string;
}

interface Props {
	rotulo: string;
	valor: string;
	aoMudar: (valor: string) => void;
	opcoes: OpcaoDaSelecao[];
	className?: string;
}

export function Selecao({ rotulo, valor, aoMudar, opcoes, className }: Props) {
	return (
		<Primitiva.Root value={valor} onValueChange={aoMudar}>
			<Primitiva.Trigger
				aria-label={rotulo}
				className={cn(
					"transicao-de-estado inline-flex h-8 cursor-pointer items-center justify-between gap-2 rounded-controle border border-contorno bg-superficie pr-2 pl-3 text-14 text-primario hover:border-primario data-[state=open]:bg-elevado",
					className,
				)}
			>
				<span className="truncate">
					<Primitiva.Value />
				</span>
				<Primitiva.Icon className="text-secundario">
					<ChevronDown aria-hidden="true" size={16} strokeWidth={1.75} />
				</Primitiva.Icon>
			</Primitiva.Trigger>
			<Primitiva.Portal>
				<Primitiva.Content
					position="popper"
					sideOffset={4}
					collisionPadding={12}
					className="z-20 max-h-(--radix-select-content-available-height) min-w-(--radix-select-trigger-width) overflow-hidden rounded-controle border border-borda bg-elevado p-1 text-14 text-primario"
				>
					<Primitiva.Viewport>
						{opcoes.map((opcao) => (
							<Primitiva.Item
								key={opcao.valor}
								value={opcao.valor}
								className="relative flex h-8 cursor-default items-center rounded-selo pr-8 pl-2 outline-none select-none data-highlighted:bg-superficie"
							>
								<Primitiva.ItemText>{opcao.rotulo}</Primitiva.ItemText>
								<Primitiva.ItemIndicator className="absolute right-2 inline-flex">
									<Check aria-hidden="true" size={16} strokeWidth={1.75} />
								</Primitiva.ItemIndicator>
							</Primitiva.Item>
						))}
					</Primitiva.Viewport>
				</Primitiva.Content>
			</Primitiva.Portal>
		</Primitiva.Root>
	);
}
