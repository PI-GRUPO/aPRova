import { type ReactNode, useEffect } from "react";
import { Link, useLocation } from "react-router";
import type { EstadoDaTela } from "@/lib/estado-da-tela";

interface Vazio {
	titulo: string;
	texto: string;
	acao?: ReactNode;
}

interface Props {
	titulo: ReactNode;
	tituloDoDocumento: string;
	contexto?: ReactNode;
	estado: EstadoDaTela;
	oQueCarrega: string;
	vazio: Vazio;
	esqueleto: ReactNode;
	children: ReactNode;
}

export function Pagina({ titulo, tituloDoDocumento, contexto, estado, oQueCarrega, vazio, esqueleto, children }: Props) {
	const { pathname } = useLocation();

	useEffect(() => {
		document.title = `${tituloDoDocumento} · aPRova`;
	}, [tituloDoDocumento]);

	return (
		<main
			id="conteudo"
			tabIndex={-1}
			data-estado={estado}
			aria-busy={estado === "carregando"}
			className="min-w-0 flex-1 px-4 pt-6 pb-16 md:px-8 md:pt-8"
		>
			<header className="mb-8 flex flex-wrap items-baseline justify-between gap-x-6 gap-y-1">
				<h1 className="text-24 font-semibold">{titulo}</h1>
				{contexto ? <p className="text-secundario">{contexto}</p> : null}
			</header>

			{estado === "dados" ? children : null}

			{estado === "carregando" ? (
				<div>
					<p role="status" className="sr-only">
						Carregando {oQueCarrega}…
					</p>
					<div aria-hidden="true">{esqueleto}</div>
				</div>
			) : null}

			{estado === "erro" ? (
				<div role="alert" className="max-w-[65ch] rounded-painel border border-borda bg-superficie p-6">
					<h2 className="text-16 font-semibold">Não foi possível carregar {oQueCarrega}</h2>
					<p className="mt-2 text-secundario">
						O servidor do aPRova não respondeu a tempo. Nenhuma análise foi perdida: as decisões e os recibos continuam gravados, e o que
						estava bloqueado no GitHub continua bloqueado.
					</p>
					<p className="mt-4">
						<Link to={pathname}>Tentar de novo</Link>
					</p>
				</div>
			) : null}

			{estado === "vazio" ? (
				<div data-testid="estado-vazio" className="max-w-[65ch] rounded-painel border border-borda bg-superficie p-6">
					<h2 className="text-16 font-semibold">{vazio.titulo}</h2>
					<p className="mt-2 text-secundario">{vazio.texto}</p>
					{vazio.acao ? <p className="mt-4">{vazio.acao}</p> : null}
				</div>
			) : null}
		</main>
	);
}
