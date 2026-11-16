import { lazy, Suspense, useEffect } from "react";
import { BrowserRouter, Route, Routes, useLocation } from "react-router";
import { Casca } from "@/componentes/Casca";
import { Conta, Entrar } from "@/paginas/Acesso";
import { AnaliseDemonstracao } from "@/paginas/AnaliseDemonstracao";
import { Analises } from "@/paginas/Analises";
import { Configuracao } from "@/paginas/Configuracao";
import { DetalheAnalise } from "@/paginas/DetalheAnalise";
import { Auditoria, Regras, Repositorios } from "@/paginas/LeiturasGestao";
import { DetalheDoRecibo, Recibos } from "@/paginas/Recibos";
import { RelatorioSemanalPagina } from "@/paginas/RelatorioSemanal";
import { Sla } from "@/paginas/Sla";
import { Sugestoes } from "@/paginas/Sugestoes";
import { VisaoGeral } from "@/paginas/VisaoGeral";
import { NaoEncontrada } from "./NaoEncontrada";
import { secoes } from "./rotas";
import { ProvedorDeSessao } from "./sessao";

const Harness = __HARNESS__ ? lazy(() => import("../../e2e/Harness")) : null;
function Titulo() {
	const { pathname } = useLocation();
	useEffect(() => {
		const secao = secoes.find((s) => s.caminho === pathname || (s.caminho !== "/" && pathname.startsWith(`${s.caminho}/`)));
		document.title = `${secao?.titulo ?? (pathname === "/entrar" ? "Entrar" : pathname === "/relatorio-semanal" ? "Relatório semanal" : "Página não encontrada")} · aPRova`;
	}, [pathname]);
	return null;
}
export function App() {
	return (
		<BrowserRouter>
			<ProvedorDeSessao>
				<Titulo />
				<Routes>
					<Route path="/entrar" element={<Entrar />} />
					<Route element={<Casca />}>
						{secoes.map((s) => (
							<Route
								key={s.caminho}
								path={s.caminho}
								element={
									s.caminho === "/recibos" ? (
										<Recibos />
									) : s.caminho === "/" ? (
										<VisaoGeral />
									) : s.caminho === "/analises" ? (
										<Analises />
									) : s.caminho === "/conta" ? (
										<Conta />
									) : (
										{
											"/repositorios": <Repositorios />,
											"/regras": <Regras />,
											"/auditoria": <Auditoria />,
											"/configuracao": <Configuracao />,
											"/sugestoes": <Sugestoes />,
											"/sla": <Sla />,
										}[s.caminho]
									)
								}
							/>
						))}
						<Route path="/relatorio-semanal" element={<RelatorioSemanalPagina />} />
						<Route path="/analises/demonstracao/:id?" element={<AnaliseDemonstracao />} />
						<Route path="/analises/:id" element={<DetalheAnalise />} />
						<Route path="/recibos/:id" element={<DetalheDoRecibo />} />
						{Harness && (
							<Route
								path="/__teste"
								element={
									<Suspense fallback={null}>
										<Harness />
									</Suspense>
								}
							/>
						)}
						<Route path="*" element={<NaoEncontrada />} />
					</Route>
				</Routes>
			</ProvedorDeSessao>
		</BrowserRouter>
	);
}
