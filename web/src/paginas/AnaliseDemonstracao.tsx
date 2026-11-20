import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { useSessao } from "@/app/sessao";
import { Ativo } from "@/componentes/Ativo";
import { useEscopo } from "@/componentes/ConsultaPrivada";
import { LinhaDoTempoDemonstracao } from "@/componentes/LinhaDoTempoDemonstracao";
import { Botao, PainelDeEstado } from "@/componentes/Primitivos";
import { consultarAnaliseDemonstracao, type AnaliseDemonstracao as Modelo, retornoDaFila } from "@/servicos/analises-demonstracao";
import { ConteudoAnalise } from "./DetalheAnalise";
import "@/estilos/demonstracao.css";

export function AnaliseDemonstracao() {
	const { id = "" } = useParams();
	const { sessao } = useSessao();
	const { instalacao } = useEscopo("");
	if (!sessao?.autenticado || !sessao.instalacoes.some((i) => String(i.id) === instalacao))
		return (
			<PainelDeEstado titulo="Instalação não autorizada">
				<p>Confira suas instalações na conta.</p>
			</PainelDeEstado>
		);
	return <ConsultaDemonstracao key={JSON.stringify([sessao.usuario.id, instalacao, id])} instalacao={instalacao} id={id} />;
}
function ConsultaDemonstracao({ id, instalacao }: { id: string; instalacao: string }) {
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
					<h1>Não foi possível carregar a análise</h1>
					<PainelDeEstado titulo="Análise indisponível" erro>
						<p>Tente carregar novamente a análise.</p>
						<Botao onClick={() => window.location.reload()}>Tentar novamente</Botao>
					</PainelDeEstado>
				</>
			) : dados === null ? (
				<>
					<h1>Análise não encontrada</h1>
					<PainelDeEstado titulo="Análise indisponível">
						<p>Escolha uma análise na fila.</p>
					</PainelDeEstado>
				</>
			) : !dados ? (
				<>
					<h1>Detalhe da análise</h1>
					<p role="status">Carregando análise…</p>
				</>
			) : (
				<ConteudoAnalise
					dados={dados.dados}
					instalacao={instalacao}
					demonstracao
					complementoSimulado={dados.complemento}
					linhaDoTempoSimulada={dados.complemento && <LinhaDoTempoDemonstracao etapas={dados.complemento.etapas} />}
				/>
			)}
		</div>
	);
}
