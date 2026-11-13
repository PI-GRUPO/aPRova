import { Link } from "react-router";
import { Pagina } from "@/componentes/Pagina";

export function NaoEncontrada() {
	return (
		<Pagina
			titulo="Página não encontrada"
			tituloDoDocumento="Página não encontrada"
			estado="dados"
			oQueCarrega="a página"
			vazio={{ titulo: "", texto: "" }}
			esqueleto={null}
		>
			<p className="max-w-[65ch] text-secundario">
				Este endereço não corresponde a nenhuma tela do dashboard. Se você chegou aqui pelo link de um recibo, confira se o id foi copiado
				inteiro. Para recomeçar, volte à <Link to="/">Visão geral</Link>.
			</p>
		</Pagina>
	);
}
