import { BrowserRouter, Route, Routes } from "react-router";
import { Casca } from "@/componentes/Casca";
import { NaoEncontrada } from "@/telas/NaoEncontrada";
import { VisaoGeral } from "@/telas/VisaoGeral";

export function App() {
	return (
		<BrowserRouter>
			<Routes>
				<Route element={<Casca />}>
					<Route index element={<VisaoGeral />} />
					<Route path="*" element={<NaoEncontrada />} />
				</Route>
			</Routes>
		</BrowserRouter>
	);
}
