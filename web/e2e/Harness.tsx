import { useState } from "react";
import { Botao, Campo, Modal, PainelDeEstado } from "@/componentes/Primitivos";
import { Selecao } from "@/componentes/Selecao";
import {
	CabecalhoDaTabela,
	Celula,
	Coluna,
	ColunaOrdenavel,
	CorpoDaTabela,
	LinhaDaTabela,
	type Sentido,
	Tabela,
} from "@/componentes/Tabela";
import { consultarJSON, ErroHTTP } from "@/servicos/http";
import { consultarProntidao, consultarSaude } from "@/servicos/saude";

const dados = [
	{ nome: "Cenário B", quantidade: 2 },
	{ nome: "Cenário A", quantidade: 1 },
];

export default function Harness() {
	const [aberto, definirAberto] = useState(false);
	const [sentido, definirSentido] = useState<Sentido>("crescente");
	const [busca, definirBusca] = useState("");
	const [grupo, definirGrupo] = useState("todos");
	const [valor, definirValor] = useState("");
	const [erro, definirErro] = useState("");
	const [resultado, definirResultado] = useState("Não consultado.");
	const [cancelamento, definirCancelamento] = useState<AbortController | null>(null);
	async function consultar(tipo: string) {
		const controlador = new AbortController();
		definirCancelamento(controlador);
		try {
			if (tipo === "health") await consultarSaude(controlador.signal);
			else if (tipo === "ready") await consultarProntidao(controlador.signal);
			else
				await consultarJSON("/api/teste", {
					signal: controlador.signal,
					timeout: tipo === "timeout" ? 30 : 10000,
					privada: tipo === "privada",
				});
			definirResultado("Serviço disponível.");
		} catch (e) {
			definirResultado(e instanceof ErroHTTP ? `${e.tipo}: ${e.message}${e.status ? ` (${e.status})` : ""}` : "Erro inesperado.");
		} finally {
			definirCancelamento(null);
		}
	}
	return (
		<div className="pagina">
			<h1>Ambiente isolado de testes</h1>
			<div className="controles">
				<Campo
					rotulo="Buscar cenário"
					value={busca}
					onChange={(e) => {
						definirBusca(e.target.value);
					}}
				/>
				<Selecao
					rotulo="Grupo"
					valor={grupo}
					aoMudar={definirGrupo}
					opcoes={[
						{ valor: "todos", rotulo: "Todos" },
						{ valor: "A", rotulo: "Somente A" },
					]}
				/>
				<Botao
					onClick={() => {
						definirBusca("");
						definirGrupo("todos");
					}}
				>
					Limpar filtros
				</Botao>
			</div>
			<div className="tabela-rolavel">
				<Tabela legenda="Cenários de teste">
					<CabecalhoDaTabela>
						<ColunaOrdenavel
							sentido={sentido}
							aoOrdenar={() => {
								definirSentido(sentido === "crescente" ? "decrescente" : "crescente");
							}}
						>
							Nome
						</ColunaOrdenavel>
						<Coluna>Quantidade</Coluna>
					</CabecalhoDaTabela>
					<CorpoDaTabela>
						{dados
							.filter((d) => d.nome.toLowerCase().includes(busca.toLowerCase()) && (grupo === "todos" || d.nome.endsWith(grupo)))
							.sort((a, b) => a.nome.localeCompare(b.nome) * (sentido === "crescente" ? 1 : -1))
							.map((d) => (
								<LinhaDaTabela key={d.nome}>
									<Celula>{d.nome}</Celula>
									<Celula>{d.quantidade}</Celula>
								</LinhaDaTabela>
							))}
					</CorpoDaTabela>
				</Tabela>
			</div>
			<Modal
				aberto={aberto}
				aoMudar={definirAberto}
				gatilho={<Botao>Abrir formulário</Botao>}
				titulo="Revisar formulário"
				descricao="Cenário de teste do componente, sem efeito no produto."
			>
				<form
					onSubmit={(e) => {
						e.preventDefault();
						if (!valor.trim()) definirErro("Informe um valor.");
						else {
							definirErro("");
							definirAberto(false);
						}
					}}
				>
					<Campo
						rotulo="Justificativa de teste"
						value={valor}
						onChange={(e) => {
							definirValor(e.target.value);
						}}
						erro={erro}
					/>
					<Botao type="submit">Confirmar</Botao>
				</form>
			</Modal>
			<div className="controles">
				{["health", "ready", "rede", "timeout", "privada"].map((t) => (
					<Botao key={t} onClick={() => void consultar(t)}>
						Consultar {t}
					</Botao>
				))}
				<Botao
					disabled={!cancelamento}
					onClick={() => {
						cancelamento?.abort();
					}}
				>
					Cancelar consulta
				</Botao>
			</div>
			<PainelDeEstado titulo="Estado da consulta">
				<p role="status">{resultado}</p>
			</PainelDeEstado>
		</div>
	);
}
