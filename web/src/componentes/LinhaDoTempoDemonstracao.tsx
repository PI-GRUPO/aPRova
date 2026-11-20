import type { EtapaDemonstracao } from "@/servicos/analises-demonstracao";
import { Ativo } from "./Ativo";
import { Botao, Modal } from "./Primitivos";
export function LinhaDoTempoDemonstracao({ etapas }: { etapas: EtapaDemonstracao[] }) {
	return (
		<ol className="timeline-analise timeline-demonstracao">
			{etapas.map((e) => (
				<li key={e.id}>
					<span className={`marcador-etapa etapa-demo-${e.estado}`}>
						<Ativo nome={e.estado === "concluida" ? "concluido" : e.estado === "falhou" ? "falhou" : "clock"} />
						{e.estado === "concluida" ? "Concluída" : e.estado === "falhou" ? "Falhou" : "Aguardando"}
					</span>
					<Modal titulo={e.titulo} descricao="Detalhes da etapa" gatilho={<Botao variante="discreto">{e.titulo}</Botao>}>
						<p>{e.detalhe}</p>
					</Modal>
					<time className="apoio mono" dateTime={e.momento}>
						{new Date(e.momento).toLocaleTimeString("pt-BR", { timeZone: "UTC" })}
					</time>
				</li>
			))}
		</ol>
	);
}
