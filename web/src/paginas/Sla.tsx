import { useRef, useState } from "react";
import { useEscopo } from "@/componentes/ConsultaPrivada";
import { PaginaGestao, SecaoGestao, SeletorGestao, TabelaGestao, useFiltrosGestao, useGestao, useOpcoesRepos } from "@/componentes/Gestao";
import { Botao, Campo } from "@/componentes/Primitivos";
import { salvarSla } from "@/servicos/gestao";

export function Sla() {
	const opcoesRepos = useOpcoesRepos();
	const { q, atualizar } = useFiltrosGestao();
	const dados = useGestao();
	const repo = q.get("repositorio") ?? opcoesRepos[0]?.valor ?? "";
	return (
		<PaginaGestao titulo="Limites de SLA" subtitulo="Prazos de aviso e escalonamento por repositório">
			<div className="escopo-sla">
				<SeletorGestao rotulo="Repositório" valor={repo} aoMudar={(v) => atualizar("repositorio", v)} opcoes={opcoesRepos} />
			</div>
			{opcoesRepos.some((r) => r.valor === repo) && dados.sla[repo] ? (
				<FormularioSla key={repo} repo={repo} atual={dados.sla[repo]} />
			) : (
				<p>Limites indisponíveis para este repositório.</p>
			)}
		</PaginaGestao>
	);
}
function FormularioSla({ repo, atual }: { repo: string; atual: { primeiro: number; segundo: number } }) {
	const escopo = useEscopo(repo);
	const [primeiro, setPrimeiro] = useState(String(atual.primeiro));
	const [segundo, setSegundo] = useState(String(atual.segundo));
	const [erro, setErro] = useState("");
	const [salvo, setSalvo] = useState(false);
	const [enviando, setEnviando] = useState(false);
	const trava = useRef(false);
	async function salvar() {
		if (!salvarSla.disponivel || trava.current) return;
		trava.current = true;
		setEnviando(true);
		setErro("");
		setSalvo(false);
		try {
			await salvarSla.executar(escopo, Number(primeiro), Number(segundo));
			setSalvo(true);
		} catch (e) {
			setErro(e instanceof Error ? e.message : "Não foi possível salvar.");
		} finally {
			trava.current = false;
			setEnviando(false);
		}
	}
	const horas = (n: number) => `${n} ${n === 1 ? "hora" : "horas"}`;
	return (
		<>
			<SecaoGestao
				titulo="Primeiro aviso e escalonamento"
				ajuda="Ambos os prazos contam desde o início da espera. O segundo deve ser maior que o primeiro; eles não se somam."
				caixa
			>
				{!salvarSla.disponivel && <p className="apoio">A alteração dos prazos está indisponível.</p>}
				<form
					onSubmit={(e) => {
						e.preventDefault();
						void salvar();
					}}
					noValidate
				>
					<div className="prazos-sla">
						<Campo
							rotulo="Primeiro aviso (horas)"
							type="number"
							min={1}
							step={1}
							value={primeiro}
							onChange={(e) => {
								setPrimeiro(e.target.value);
								setSalvo(false);
							}}
						/>
						<Campo
							rotulo="Escalonamento (horas)"
							type="number"
							min={1}
							step={1}
							value={segundo}
							onChange={(e) => {
								setSegundo(e.target.value);
								setSalvo(false);
							}}
						/>
					</div>
					{erro && <p role="alert">{erro}</p>}
					<Botao type="submit" variante="primario" disabled={!salvarSla.disponivel || enviando}>
						{enviando ? "Salvando…" : "Salvar limites"}
					</Botao>
				</form>
				{salvo && (
					<p role="status" className="apoio">
						Limites salvos.
					</p>
				)}
			</SecaoGestao>
			<SecaoGestao titulo="Efeito para a equipe">
				<TabelaGestao
					legenda="Prazos de SLA"
					colunas={["Momento", "Acompanhamento"]}
					larguras={["300px", "720px"]}
					linhas={[
						{ id: "antes", celulas: [`Antes de ${horas(atual.primeiro)}`, "Pendência aparece na fila do dashboard"] },
						{ id: "primeiro", celulas: [`A partir de ${horas(atual.primeiro)}`, "Primeiro aviso"] },
						{ id: "segundo", celulas: [`A partir de ${horas(atual.segundo)}`, "Escalonamento"] },
					]}
				/>
			</SecaoGestao>
		</>
	);
}
