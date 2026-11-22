import { type ReactNode, useEffect, useState } from "react";
import { useSearchParams } from "react-router";
import type { CacheDeLeituras } from "@/app/cache-de-leituras";
import { useSessao } from "@/app/sessao";
import { Protegida } from "@/paginas/Acesso";
import { erroDeAnalise } from "@/servicos/analises";
import { type Consulta, RecursoIndisponivel } from "@/servicos/fonte";
import { ErroHTTP } from "@/servicos/http";
import { Botao, PainelDeEstado } from "./Primitivos";

type Props<T> = {
	titulo: string;
	consultar: Consulta<T>;
	children: (dados: T) => ReactNode;
	secao?: boolean;
	filtrosDaConsulta?: string[];
	ajuda?: ReactNode;
	reutilizar?: boolean;
};
export function ConsultaPrivada<T>(props: Props<T>) {
	const { sessao, erro } = useSessao();
	const [q] = useSearchParams();
	const instalacao = q.get("installation_id") ?? (sessao?.autenticado ? String(sessao.instalacoes[0]?.id ?? "") : "");
	const filtros = props.filtrosDaConsulta
		? new URLSearchParams([...q].filter(([nome]) => props.filtrosDaConsulta?.includes(nome))).toString()
		: q.toString();
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
type EstadoDaLeitura<T> = { dados: T } | { erro: string } | { indisponivel: true };
type ResultadoDaLeitura<T> = {
	consultar: Consulta<T>;
	chave: string;
	cache?: CacheDeLeituras;
	estado?: EstadoDaLeitura<T>;
};

export function useLeitura<T>(
	consultar: Consulta<T>,
	instalacao: string,
	filtros: string,
	{ reutilizar = false, habilitada = true }: { reutilizar?: boolean; habilitada?: boolean } = {},
) {
	const { cacheDeLeituras } = useSessao();
	const cache = reutilizar ? cacheDeLeituras : undefined;
	const chave = JSON.stringify([instalacao, filtros]);
	const [resultado, definir] = useState<ResultadoDaLeitura<T>>();
	const [tentativa, repetir] = useState(0);
	// biome-ignore lint/correctness/useExhaustiveDependencies: tentativa repete a consulta após erro.
	useEffect(() => {
		if (!habilitada) return;
		const c = new AbortController();
		const q = new URLSearchParams(filtros);
		const geracao = cache?.geracao;
		const registrar = (estado?: EstadoDaLeitura<T>) => definir({ consultar, chave, cache, estado });
		registrar(cache?.buscar(consultar, chave));
		consultar({ instalacao, repositorio: q.get("repositorio") ?? "" }, q, c.signal)
			.then((dados) => {
				if (c.signal.aborted) return;
				if (cache && cache.geracao === geracao) cache.guardar(consultar, chave, dados);
				registrar({ dados });
			})
			.catch((e) => {
				if (c.signal.aborted) return;
				if (e instanceof ErroHTTP && (e.status === 401 || e.status === 403)) cache?.limpar();
				else cache?.descartar(consultar, chave);
				registrar(e instanceof RecursoIndisponivel ? { indisponivel: true } : { erro: erroDeAnalise(e) });
			});
		return () => {
			c.abort();
		};
	}, [consultar, instalacao, filtros, tentativa, cache, chave, habilitada]);
	const mesmaConsulta = resultado?.consultar === consultar && resultado.chave === chave && resultado.cache === cache;
	const estado = habilitada ? (mesmaConsulta ? resultado.estado : cache?.buscar(consultar, chave)) : undefined;
	return { estado, repetir: () => repetir((n) => n + 1) };
}
function Resultado<T>({
	secao,
	titulo,
	consultar,
	children,
	ajuda,
	instalacao,
	filtros,
	reutilizar,
}: Props<T> & { instalacao: string; filtros: string }) {
	const { estado, repetir } = useLeitura(consultar, instalacao, filtros, { reutilizar });
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
					<Botao
						onClick={() => {
							repetir();
						}}
					>
						Tentar novamente
					</Botao>
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
