import { type ReactNode, useState } from "react";
export function ControlesDaLista({
	children,
	mobile,
	titulo = "Filtros e instalação",
}: {
	children: ReactNode;
	mobile: boolean;
	titulo?: string;
}) {
	const [aberto, setAberto] = useState(false);
	return (
		<details
			className="controles-lista-analises"
			open={!mobile || aberto}
			onToggle={(e) => {
				if (mobile) setAberto(e.currentTarget.open);
			}}
		>
			<summary>{titulo}</summary>
			<div className="conteudo-filtros-analises">{children}</div>
		</details>
	);
}
