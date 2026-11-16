import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { Ativo } from "@/componentes/Ativo";
import { LinhaDoTempoDemonstracao } from "@/componentes/LinhaDoTempoDemonstracao";
import { Botao, PainelDeEstado } from "@/componentes/Primitivos";
import { consultarAnaliseDemonstracao, type AnaliseDemonstracao as Modelo, retornoDaFila } from "@/servicos/analises-demonstracao";
import { ConteudoAnalise } from "./DetalheAnalise";
import "@/estilos/demonstracao.css";

export function AnaliseDemonstracao() {
	const { id = "" } = useParams();
	return <ConsultaDemonstracao key={id} id={id} />;
}
function ConsultaDemonstracao({ id }: { id: string }) {
	const [dados, setDados] = useState<Modelo | null>();
	const [erro, setErro] = useState(false);
	const [q] = useSearchParams();
	useEffect(() => {
		const c = new AbortController();
		void consultarAnaliseDemonstracao(id, c.signal)
			.then((v) => {
				if (!c.signal.aborted) setDados(v);
			})
			.catch(() => {
				if (!c.signal.aborted) setErro(true);
			});
		return () => c.abort();
	}, [id]);
	return (
		<div className="pagina pagina-demonstracao">
			<Link className="voltar" to={retornoDaFila(q.get("retorno"))}>
				<Ativo nome="voltar" />
				Voltar à visão geral
			</Link>
			{erro ? (
				<>
					<h1>Não foi possível carregar a demonstração</h1>
					<PainelDeEstado titulo="Demonstração indisponível" erro>
						<p>Tente carregar novamente os dados locais de referência.</p>
						<Botao onClick={() => window.location.reload()}>Tentar novamente</Botao>
					</PainelDeEstado>
				</>
			) : dados === null ? (
				<>
					<h1>Análise de demonstração não encontrada</h1>
					<PainelDeEstado titulo="Referência indisponível">
						<p>Escolha uma análise da fila de demonstração. Nenhuma consulta foi enviada à API de análises.</p>
					</PainelDeEstado>
				</>
			) : !dados ? (
				<>
					<h1>Detalhe da análise</h1>
					<p role="status">Carregando demonstração…</p>
				</>
			) : (
				<ConteudoAnalise
					dados={dados.dados}
					instalacao={dados.identidade.instalacao}
					usuarioSimulado={dados.identidade.usuario}
					complementoSimulado={dados.complemento}
					linhaDoTempoSimulada={dados.complemento && <LinhaDoTempoDemonstracao etapas={dados.complemento.etapas} />}
				/>
			)}
		</div>
	);
}
