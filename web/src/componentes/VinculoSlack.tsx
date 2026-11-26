import { useEffect, useRef, useState } from "react";
import { alterarVinculo, consultarVinculo } from "@/servicos/vinculo-slack";
import { Botao } from "./Primitivos";
export function VinculoSlack({ usuario }: { usuario: number }) {
	const [vinculo, definir] = useState<Awaited<ReturnType<typeof consultarVinculo>>>();
	const [erro, setErro] = useState("");
	const [revisao, atualizar] = useState(0);
	const [enviando, setEnviando] = useState(false);
	const trava = useRef(false);
	// biome-ignore lint/correctness/useExhaustiveDependencies: revisao solicita a leitura do vínculo persistido.
	useEffect(() => {
		const c = new AbortController();
		definir(undefined);
		setErro("");
		consultarVinculo(usuario, c.signal)
			.then((v) => {
				if (!c.signal.aborted) definir(v);
			})
			.catch(() => {
				if (!c.signal.aborted) setErro("Não foi possível consultar o vínculo.");
			});
		return () => {
			c.abort();
		};
	}, [revisao, usuario]);
	async function alterar() {
		if (trava.current || vinculo === undefined || !alterarVinculo.disponivel) return;
		trava.current = true;
		setEnviando(true);
		setErro("");
		try {
			await alterarVinculo.executar(vinculo ? "desvincular" : "conectar");
			atualizar((n) => n + 1);
		} catch {
			setErro("Não foi possível alterar o vínculo.");
		} finally {
			trava.current = false;
			setEnviando(false);
		}
	}
	return (
		<section className="painel-sessao">
			<h2>Conectar Slack</h2>
			<div>
				{erro ? (
					<p role="alert">{erro}</p>
				) : vinculo === undefined ? (
					<p role="status">Consultando vínculo…</p>
				) : (
					<p>{vinculo ? "Vínculo configurado para seu login GitHub." : "Nenhuma conta Slack vinculada."}</p>
				)}
				{erro && (
					<Botao
						onClick={() => {
							atualizar((n) => n + 1);
						}}
					>
						Consultar novamente
					</Botao>
				)}
				<Botao disabled={!alterarVinculo.disponivel || vinculo === undefined || enviando} onClick={() => void alterar()}>
					{vinculo ? "Desvincular conta" : "Conectar minha conta Slack"}
				</Botao>
			</div>
		</section>
	);
}
