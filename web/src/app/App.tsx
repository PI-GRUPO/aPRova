import { lazy, Suspense, useEffect } from "react";
import { BrowserRouter, Route, Routes, useLocation } from "react-router";
import { Casca } from "@/componentes/Casca";
import { PainelDeEstado } from "@/componentes/Primitivos";
import { DetalheDoRecibo, Recibos } from "@/paginas/Recibos";
import { NaoEncontrada } from "./NaoEncontrada";
import { secoes } from "./rotas";

const Harness = __HARNESS__ ? lazy(() => import("../../e2e/Harness")) : null;
function Indisponivel({ titulo, detalhe = false }: { titulo: string; detalhe?: boolean }) {
	return (
		<div className="pagina">
			<h1>{titulo}</h1>
			<PainelDeEstado titulo="Disponível em breve">
				<p>
					{detalhe ? "A consulta deste detalhe ainda não está disponível." : "Esta área ainda está em desenvolvimento."} Os dados serão
					exibidos quando o acesso à sua conta estiver disponível.
				</p>
			</PainelDeEstado>
		</div>
	);
}
function Titulo() {
	const { pathname } = useLocation();
	useEffect(() => {
		const secao = secoes.find((s) => s.caminho === pathname || (s.caminho !== "/" && pathname.startsWith(`${s.caminho}/`)));
		document.title = `${secao?.titulo ?? (pathname === "/entrar" ? "Entrar" : "Página não encontrada")} · aPRova`;
	}, [pathname]);
	return null;
}
export function App() {
	return (
		<BrowserRouter>
			<Titulo />
			<Routes>
				<Route element={<Casca />}>
					{secoes.map((s) => (
						<Route key={s.caminho} path={s.caminho} element={s.caminho === "/recibos" ? <Recibos /> : <Indisponivel titulo={s.titulo} />} />
					))}
					<Route path="/analises/:id" element={<Indisponivel titulo="Detalhe da análise" detalhe />} />
					<Route path="/recibos/:id" element={<DetalheDoRecibo />} />
					<Route
						path="/entrar"
						element={
							<div className="pagina">
								<h1>Entrar com GitHub</h1>
								<PainelDeEstado titulo="Acesso em preparação">
									<p>O login com GitHub ainda não está disponível. Volte em breve para acessar seus repositórios.</p>
								</PainelDeEstado>
							</div>
						}
					/>
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
		</BrowserRouter>
	);
}
