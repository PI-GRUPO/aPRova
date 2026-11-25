import * as Dialog from "@radix-ui/react-dialog";
import * as Tooltip from "@radix-ui/react-tooltip";
import { type ButtonHTMLAttributes, type ComponentProps, type InputHTMLAttributes, type ReactNode, useId, useState } from "react";
import { Ativo } from "./Ativo";
import { cn } from "./classes";

export function Botao({
	variante = "secundario",
	className,
	...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variante?: "primario" | "secundario" | "discreto" }) {
	return <button type="button" className={cn("botao", `botao-${variante}`, className)} {...props} />;
}
export function Ajuda({
	titulo,
	descricao,
	complemento,
	className,
}: {
	titulo: string;
	descricao: string;
	complemento?: string;
	className: string;
}) {
	const [aberta, definirAberta] = useState(false);
	return (
		<Tooltip.Root open={aberta} onOpenChange={definirAberta}>
			<Tooltip.Trigger
				asChild
				onFocus={(e) => {
					const gatilho = e.currentTarget;
					if (!gatilho.matches(":focus-visible")) return;
					// Aguarde a rolagem que o navegador faz ao mover o foco pelo teclado.
					e.preventDefault();
					requestAnimationFrame(() => {
						if (gatilho === document.activeElement) definirAberta(true);
					});
				}}
				onClick={(e) => {
					// O toque também mostra a explicação em telas sem mouse.
					e.preventDefault();
					definirAberta(true);
				}}
			>
				<Botao variante="discreto" className={className} aria-label={`Ajuda: ${titulo}`}>
					<Ativo nome="ajuda" />
				</Botao>
			</Tooltip.Trigger>
			<Tooltip.Portal>
				<Tooltip.Content className="balao-ajuda" sideOffset={8} collisionPadding={12}>
					<p>{descricao}</p>
					{complemento && <p>{complemento}</p>}
					<Tooltip.Arrow className="balao-ajuda-seta" />
				</Tooltip.Content>
			</Tooltip.Portal>
		</Tooltip.Root>
	);
}
export function Campo({ rotulo, erro, id, ...props }: InputHTMLAttributes<HTMLInputElement> & { rotulo: string; erro?: string }) {
	const gerado = useId();
	const campoId = id ?? gerado;
	return (
		<div className="campo">
			<label htmlFor={campoId}>{rotulo}</label>
			<input id={campoId} aria-invalid={!!erro} aria-describedby={erro ? `${campoId}-erro` : undefined} {...props} />
			{erro && (
				<p role="alert" id={`${campoId}-erro`}>
					{erro}
				</p>
			)}
		</div>
	);
}
export function Modal({
	gatilho,
	titulo,
	descricao,
	children,
	aberto,
	aoMudar,
	className = "",
}: {
	gatilho: ReactNode;
	titulo: string;
	descricao: string;
	children?: ReactNode;
	aberto?: boolean;
	aoMudar?: ComponentProps<typeof Dialog.Root>["onOpenChange"];
	className?: string;
}) {
	return (
		<Dialog.Root open={aberto} onOpenChange={aoMudar}>
			<Dialog.Trigger asChild>{gatilho}</Dialog.Trigger>
			<Dialog.Portal>
				<Dialog.Overlay className="modal-fundo" />
				<Dialog.Content className={cn("modal", className)}>
					<Dialog.Title>{titulo}</Dialog.Title>
					<Dialog.Description>{descricao}</Dialog.Description>
					{children}
					<Dialog.Close asChild>
						<Botao className="modal-fechar" aria-label="Fechar">
							<Ativo nome="fechar" />
						</Botao>
					</Dialog.Close>
				</Dialog.Content>
			</Dialog.Portal>
		</Dialog.Root>
	);
}
export function PainelDeEstado({ titulo, children, erro = false }: { titulo: string; children: ReactNode; erro?: boolean }) {
	return (
		<section className="estado" aria-live="polite" role={erro ? "alert" : undefined}>
			<h2>{titulo}</h2>
			<div>{children}</div>
		</section>
	);
}
