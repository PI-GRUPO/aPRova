import { ArrowRight } from "lucide-react";
import { Link } from "react-router";
import { nomeCurtoDoRepositorio, recibos } from "@/lib/dados";
import { dataEHora } from "@/lib/formatos";
import { Identificador } from "./Identificador";
import { SeloDeRisco } from "./SeloDeRisco";

function reciboMaisRecente() {
	const ordenados = [...recibos].sort((a, b) => new Date(b.signed_at).getTime() - new Date(a.signed_at).getTime());
	const recente = ordenados[0];
	if (!recente) return null;
	const anterior = recibos.find((recibo) => recibo.id === recente.recibo_anterior_id);
	return { recente, anterior };
}

export function UltimoRecibo() {
	const dados = reciboMaisRecente();
	if (!dados) return null;
	const { recente, anterior } = dados;

	return (
		<section
			aria-labelledby="titulo-do-ultimo-recibo"
			className="flex flex-wrap items-baseline gap-x-3 gap-y-1 border-t border-borda pt-3 text-12 text-secundario"
		>
			<h2 id="titulo-do-ultimo-recibo" className="text-12 font-medium text-primario">
				Último recibo assinado
			</h2>
			<span className="text-primario">
				{nomeCurtoDoRepositorio(recente.payload.repositorio)} <Identificador>#{recente.payload.numero_pr}</Identificador>
			</span>
			<SeloDeRisco nivel={recente.payload.nivel} variante="ponto" />
			<Identificador className="text-primario" title={recente.hash_sha256}>
				{recente.hash_sha256.slice(0, 16)}…
			</Identificador>
			<span>
				Ed25519
				{anterior ? (
					<>
						, encadeado a <Identificador title={anterior.hash_sha256}>{anterior.hash_sha256.slice(0, 8)}…</Identificador>
					</>
				) : null}
			</span>
			<span>{dataEHora(new Date(recente.signed_at))}</span>
			<Link to="/recibos" className="ml-auto inline-flex items-center gap-1">
				Ver os recibos
				<ArrowRight aria-hidden="true" size={14} strokeWidth={1.75} />
			</Link>
		</section>
	);
}
