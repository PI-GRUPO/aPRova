import { ListChecks, type LucideIcon, Moon, Receipt, Scale, SquareActivity, Sun } from "lucide-react";
import { NavLink, Outlet } from "react-router";
import { cn } from "@/lib/classes";
import { instalacao } from "@/lib/dados";
import { plural } from "@/lib/formatos";
import { useTema } from "@/lib/tema";
import logoParaFundoClaro from "../../../marca/svg/logo-horizontal-fundo-claro.svg";
import logoParaFundoEscuro from "../../../marca/svg/logo-horizontal-fundo-escuro.svg";

const secoes: {
	para: string;
	rotulo: string;
	complemento?: string;
	icone: LucideIcon;
	exata?: boolean;
}[] = [
	{ para: "/", rotulo: "Visão geral", icone: SquareActivity, exata: true },
	{ para: "/analises", rotulo: "Análises", icone: ListChecks },
	{ para: "/recibos", rotulo: "Recibos", icone: Receipt },
	{ para: "/regras", rotulo: "Regras", complemento: "do gate", icone: Scale },
];

function BotaoDeTema() {
	const { tema, alternar } = useTema();
	const proximo = tema === "escuro" ? "claro" : "escuro";
	const Icone = tema === "escuro" ? Sun : Moon;

	return (
		<button
			type="button"
			onClick={alternar}
			aria-label={`Usar tema ${proximo}`}
			className="transicao-de-estado inline-flex h-8 cursor-pointer items-center gap-2 rounded-controle border border-borda bg-superficie px-2 text-primario hover:bg-elevado"
		>
			<Icone aria-hidden="true" size={16} strokeWidth={1.75} />
			<span>Tema {proximo}</span>
		</button>
	);
}

export function Casca() {
	const { tema } = useTema();

	return (
		<div className="flex min-h-dvh flex-col md:flex-row">
			<a
				href="#conteudo"
				className="absolute top-2 left-2 z-10 -translate-y-16 rounded-controle border border-borda bg-superficie px-3 py-2 focus-visible:translate-y-0"
			>
				Pular para o conteúdo
			</a>

			<aside data-testid="barra-lateral" className="shrink-0 border-b border-borda bg-superficie md:w-60 md:border-r md:border-b-0">
				<div className="flex flex-col md:sticky md:top-0 md:h-dvh">
					<div className="flex items-center justify-between gap-4 px-4 pt-4 pb-3 md:px-5 md:pt-6 md:pb-6">
						<NavLink to="/" className="inline-flex rounded-selo">
							<img
								src={tema === "escuro" ? logoParaFundoEscuro : logoParaFundoClaro}
								alt="aPRova"
								width={116}
								height={28}
								className="h-7 w-auto"
							/>
						</NavLink>
						<div className="md:hidden">
							<BotaoDeTema />
						</div>
					</div>

					<nav aria-label="Principal" className="px-2 pb-2 md:min-h-0 md:flex-1 md:overflow-y-auto md:px-3">
						<ul className="grid grid-cols-4 gap-1 md:grid-cols-1 md:gap-1.5">
							{secoes.map(({ para, rotulo, complemento, icone: Icone, exata }) => (
								<li key={para} className="min-w-0">
									<NavLink
										to={para}
										end={exata}
										className={({ isActive }) =>
											cn(
												"transicao-de-estado relative flex h-8 items-center justify-center gap-2 rounded-controle px-1 whitespace-nowrap text-secundario no-underline -outline-offset-2 hover:bg-elevado hover:text-primario md:justify-start md:px-2.5",
												isActive &&
													"bg-elevado font-medium text-titulo max-md:after:absolute max-md:after:inset-x-2 max-md:after:bottom-0 max-md:after:h-0.5 max-md:after:bg-acento [&>svg]:text-acento",
											)
										}
									>
										<Icone aria-hidden="true" size={16} strokeWidth={1.75} className="max-md:hidden" />
										<span>
											{rotulo}
											{complemento ? <span className="max-md:sr-only"> {complemento}</span> : null}
										</span>
									</NavLink>
								</li>
							))}
						</ul>
					</nav>

					<div className="hidden flex-col gap-4 border-t border-borda px-5 py-5 md:flex">
						<dl className="text-12 text-secundario">
							<dt className="sr-only">Instalação</dt>
							<dd className="text-14 text-primario">{instalacao.conta}</dd>
							<dd>{plural(instalacao.repositorios.length, "repositório", "repositórios")} nesta instalação</dd>
							<dd className="mt-2">Protótipo com dados fictícios</dd>
						</dl>
						<div>
							<BotaoDeTema />
						</div>
					</div>
				</div>
			</aside>

			<Outlet />
		</div>
	);
}
