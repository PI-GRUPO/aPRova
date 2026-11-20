import { type ReactNode, useEffect, useState } from "react";
import { useSearchParams } from "react-router";
import { useSessao } from "@/app/sessao";
import { Protegida } from "@/paginas/Acesso";
import { erroDeAnalise } from "@/servicos/analises";
import { type Consulta, RecursoIndisponivel } from "@/servicos/fonte";
import { Botao, PainelDeEstado } from "./Primitivos";

type Props<T> = {
	titulo: string;
	consultar: Consulta<T>;
	children: (dados: T) => ReactNode;
	secao?: boolean;
	filtrosLocais?: boolean;
	ajuda?: ReactNode;
};
export function ConsultaPrivada<T>(props: Props<T>) {
	const { sessao, erro } = useSessao();
	const [q] = useSearchParams();
	const instalacao = q.get("installation_id") ?? (sessao?.autenticado ? String(sessao.instalacoes[0]?.id ?? "") : "");
	const filtros = props.filtrosLocais ? "" : q.toString();
	if (!erro && sessao?.autenticado && sessao.instalacoes.some((i) => String(i.id) === instalacao))
		return (
			<Resultado key={JSON.stringify([sessao.usuario.id, instalacao, filtros])} {...props} instalacao={instalacao} filtros={filtros} />
		);
	const Titulo = props.secao ? "h2" : "h1";
	return (
		<EspacoDaConsulta secao={props.secao}>
			<Titulo>{props.titulo}</Titulo>
			<Protegida>
				<PainelDeEstado titulo={instalacao ? "Instalação não autorizada" : "Nenhuma instalação autorizada"}>
					<p>Confira as instalações disponíveis na sua conta.</p>
				</PainelDeEstado>
			</Protegida>
		</EspacoDaConsulta>
	);
}
function EspacoDaConsulta({ secao, children }: { secao?: boolean; children: ReactNode }) {
	return secao ? children : <div className="pagina">{children}</div>;
}
function Resultado<T>({
	secao,
	titulo,
	consultar,
	children,
	ajuda,
	instalacao,
	filtros,
}: Props<T> & { instalacao: string; filtros: string }) {
	const [estado, definir] = useState<{ dados: T } | { erro: string } | { indisponivel: true }>();
	const [tentativa, repetir] = useState(0);
	// biome-ignore lint/correctness/useExhaustiveDependencies: tentativa repete a consulta após erro.
	useEffect(() => {
		const c = new AbortController();
		const q = new URLSearchParams(filtros);
		definir(undefined);
		consultar({ instalacao, repositorio: q.get("repositorio") ?? "" }, q, c.signal)
			.then((dados) => {
				if (!c.signal.aborted) definir({ dados });
			})
			.catch((e) => {
				if (!c.signal.aborted) definir(e instanceof RecursoIndisponivel ? { indisponivel: true } : { erro: erroDeAnalise(e) });
			});
		return () => c.abort();
	}, [consultar, instalacao, filtros, tentativa]);
	if (estado && "dados" in estado) return children(estado.dados);
	const Titulo = secao ? "h2" : "h1";
	return (
		<EspacoDaConsulta secao={secao}>
			<Titulo>{titulo}</Titulo>
			{!estado ? (
				<p role="status">Carregando…</p>
			) : "erro" in estado ? (
				<PainelDeEstado titulo="Falha na consulta" erro>
					<p>{estado.erro}</p>
					<Botao onClick={() => repetir((n) => n + 1)}>Tentar novamente</Botao>
				</PainelDeEstado>
			) : (
				<PainelDeEstado titulo="Recurso indisponível">{ajuda ?? <p>Esta consulta ainda não está disponível.</p>}</PainelDeEstado>
			)}
		</EspacoDaConsulta>
	);
}
export function useEscopo(repositorio: string) {
	const { sessao } = useSessao();
	const [q] = useSearchParams();
	return { instalacao: q.get("installation_id") ?? (sessao?.autenticado ? String(sessao.instalacoes[0]?.id ?? "") : ""), repositorio };
}
