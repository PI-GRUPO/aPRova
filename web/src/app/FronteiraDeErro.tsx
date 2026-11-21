import { Component, type ReactNode } from "react";
import { Botao, PainelDeEstado } from "@/componentes/Primitivos";
export class FronteiraDeErro extends Component<{ children: ReactNode }, { falhou: boolean }> {
	state = { falhou: false };
	static getDerivedStateFromError() {
		return { falhou: true };
	}
	render() {
		if (this.state.falhou)
			return (
				<div className="pagina">
					<h1>Não foi possível abrir o dashboard</h1>
					<PainelDeEstado erro titulo="Tente novamente">
						<p>Ocorreu um erro ao apresentar esta página.</p>
						<Botao
							onClick={() => {
								window.location.reload();
							}}
						>
							Recarregar página
						</Botao>
					</PainelDeEstado>
				</div>
			);
		return this.props.children;
	}
}
