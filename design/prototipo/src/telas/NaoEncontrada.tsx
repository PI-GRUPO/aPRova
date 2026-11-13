import { Link, useLocation } from "react-router";
import { Pagina } from "@/componentes/Pagina";

function Fragmentos({ caminho }: { caminho: string }) {
	const caminhoCurto = caminho.length > 28 ? `${caminho.slice(0, 27)}…` : caminho;

	return (
		<svg
			data-testid="fragmentos-404"
			aria-hidden="true"
			viewBox="0 0 1136 380"
			className="mt-8 block w-full border-y border-titulo font-sans select-none"
		>
			<g fontWeight={800} className="fill-elevado">
				<text x={-20} y={330} fontSize={420} letterSpacing={-24} transform="rotate(-6 300 190)">
					404
				</text>
			</g>
			<g fontWeight={800} className="fill-titulo">
				<text x={70} y={250} fontSize={280} transform="rotate(3 150 150)">
					4
				</text>
				<text x={600} y={300} fontSize={260} transform="rotate(90 690 210)">
					4
				</text>
			</g>
			<text
				x={340}
				y={270}
				fontSize={340}
				fontWeight={800}
				transform="rotate(-12 450 150)"
				className="fill-none stroke-secundario"
				strokeWidth={2}
			>
				0
			</text>
			<line x1={0} y1={286} x2={1136} y2={246} className="stroke-titulo" strokeWidth={1} />
			<line x1={520} y1={0} x2={480} y2={380} className="stroke-secundario" strokeWidth={1} />
			<g transform="rotate(2 1030 56)">
				<rect x={900} y={42} width={220} height={26} className="fill-titulo" />
				<text x={1010} y={60} fontSize={14} textAnchor="middle" className="fill-base font-mono">
					{caminhoCurto}
				</text>
			</g>
			<text
				x={210}
				y={236}
				fontSize={20}
				textDecoration="line-through"
				transform="rotate(-3 210 236)"
				className="fill-secundario font-mono"
			>
				{caminhoCurto}
			</text>
			<g fontSize={12} fontWeight={590} letterSpacing={0.8}>
				<text x={36} y={356} transform="rotate(-90 36 356)" className="fill-secundario">
					NÃO ENCONTRADA
				</text>
				<text x={996} y={338} className="fill-titulo">
					PÁGINA
				</text>
				<text x={746} y={152} transform="rotate(12 746 152)" className="fill-secundario">
					NENHUMA TELA AQUI
				</text>
			</g>
		</svg>
	);
}

export function NaoEncontrada() {
	const { pathname } = useLocation();

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
			<Fragmentos caminho={pathname} />
		</Pagina>
	);
}
