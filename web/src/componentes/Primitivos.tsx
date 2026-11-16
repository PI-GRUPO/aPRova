import * as Dialog from "@radix-ui/react-dialog";
import { type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode, useId } from "react";
import { Ativo } from "./Ativo";
import { cn } from "./classes";

export function Botao({
	variante = "secundario",
	className,
	...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variante?: "primario" | "secundario" | "discreto" }) {
	return <button type="button" className={cn("botao", `botao-${variante}`, className)} {...props} />;
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
}: {
	gatilho: ReactNode;
	titulo: string;
	descricao: string;
	children: ReactNode;
	aberto?: boolean;
	aoMudar?: (aberto: boolean) => void;
}) {
	return (
		<Dialog.Root open={aberto} onOpenChange={aoMudar}>
			<Dialog.Trigger asChild>{gatilho}</Dialog.Trigger>
			<Dialog.Portal>
				<Dialog.Overlay className="modal-fundo" />
				<Dialog.Content className="modal">
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
