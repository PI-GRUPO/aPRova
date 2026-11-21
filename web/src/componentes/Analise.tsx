import type { ReactNode } from "react";
import { estados, niveis } from "@/servicos/analises";
import { Ativo } from "./Ativo";
import { Botao, Modal } from "./Primitivos";
export function Risco({ nivel }: { nivel: string | null }) {
	const n = nivel === "alto" || nivel === "medio" || nivel === "baixo" ? nivel : "nao_avaliado";
	return (
		<span className={`risco-analise risco-${n}`}>
			<Ativo nome={`risco-${n}`} />
			{niveis.get(n)}
		</span>
	);
}
export function Estado({ estado }: { estado: string }) {
	return <span className="estado-analise">{estados.get(estado) ?? estado}</span>;
}
export function PainelAnalise({ titulo, ajuda, children }: { titulo: string; ajuda?: string; children: ReactNode }) {
	return (
		<section className="painel-analise">
			<header>
				<h2>{titulo}</h2>
				{ajuda && (
					<Modal
						titulo={titulo}
						descricao={ajuda}
						gatilho={
							<Botao variante="discreto" className="ajuda-analise" aria-label={`Ajuda: ${titulo}`}>
								<Ativo nome="ajuda" />
							</Botao>
						}
					>
						<p>Os dados apresentados pertencem a esta análise e a este commit.</p>
					</Modal>
				)}
			</header>
			<div className="conteudo-analise">{children}</div>
		</section>
	);
}
export const dataUTC = (v: string) => `${new Date(v).toLocaleString("pt-BR", { timeZone: "UTC" })} UTC`;
